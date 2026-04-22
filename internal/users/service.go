package users // Contains business rules that sit between handlers and repository.

import (
	"context" // Uses context.Context so requests can cancel service/repository work.
	"errors"  // Uses errors.New for domain errors in 2FA validation.
	"fmt"     // Uses fmt.Errorf to build user-facing and wrapped error messages.
	"log"     // Uses log.Printf to print demo 2FA code to server output.
	"strings" // Uses strings.TrimSpace for input validation checks.
	"time"    // Uses time.Now().UTC() for auth timestamp and TOTP calculations.

	"github.com/pquerna/otp"      // Uses OTP constants like digit size and algorithm enum.
	"github.com/pquerna/otp/totp" // Uses TOTP code generation and validation helpers.
	"golang.org/x/crypto/bcrypt"  // Uses bcrypt hash and compare functions for passwords.
)

type Service struct { // Owns business logic and calls repository methods for persistence.
	repo *Repository // Injected users repository used by all service methods.
}

func NewService(repo *Repository) *Service { // Constructor for service dependency wiring in main.go.
	return &Service{repo: repo} // Returns pointer so same service instance is reused by handler.
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (User, error) { // Validates input, prepares auth data, and creates a user.
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Phone) == "" || strings.TrimSpace(in.Password) == "" { // Ensures required fields are not blank/whitespace.
		return User{}, fmt.Errorf("name, email, phone and password are required") // Returns validation error before hitting database.
	}

	passwordHash, err := hashPassword(in.Password) // Converts raw password into secure bcrypt hash.
	if err != nil {                                // Handles hashing failure.
		return User{}, err // Propagates hash error to caller.
	}

	secret, err := generateSecret(in.Email) // Creates user-specific TOTP secret used for 2FA.
	if err != nil {                         // Handles secret generation failure.
		return User{}, err // Propagates generation error to caller.
	}

	return s.repo.Create(ctx, in, passwordHash, secret) // Persists new user with computed hash and secret.
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) { // Returns all users for list endpoint.
	return s.repo.List(ctx) // Delegates read operation to repository.
}

func (s *Service) GetUser(ctx context.Context, id int64) (User, error) { // Returns one user by ID.
	return s.repo.GetByID(ctx, id) // Delegates lookup to repository.
}

func (s *Service) UpdateUser(ctx context.Context, id int64, in UpdateUserInput) (User, error) { // Validates update payload and persists changes.
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Phone) == "" { // Prevents empty required fields.
		return User{}, fmt.Errorf("name, email and phone are required") // Returns validation message for bad input.
	}
	return s.repo.Update(ctx, id, in) // Calls repository to update row and return updated user.
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error { // Deletes one user by ID.
	return s.repo.Delete(ctx, id) // Delegates delete logic to repository.
}

func (s *Service) ChangePassword(ctx context.Context, id int64, oldPassword, newPassword string) error { // Verifies old password then saves new hash.
	if strings.TrimSpace(oldPassword) == "" || strings.TrimSpace(newPassword) == "" { // Checks both password fields are provided.
		return fmt.Errorf("old_password and new_password are required") // Returns validation error for missing values.
	}
	hash, _, _, _, err := s.repo.GetAuthData(ctx, id) // Loads current password hash (and other auth fields) from repository.
	if err != nil {                                   // Handles DB/not-found failures.
		return err // Propagates repository error directly.
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); compareErr != nil { // Compares provided old password with stored hash.
		return fmt.Errorf("old password is incorrect") // Rejects change when old password does not match.
	}

	newHash, err := hashPassword(newPassword) // Hashes new password before persistence.
	if err != nil {                           // Handles hashing failure.
		return err // Propagates hashing error.
	}
	return s.repo.UpdatePassword(ctx, id, newHash, time.Now().UTC()) // Stores new hash and sets password-changed timestamp in UTC.
}

func (s *Service) Login(ctx context.Context, id int64, password string) error { // Verifies password credentials and marks login timestamp.
	if strings.TrimSpace(password) == "" { // Validates password presence.
		return fmt.Errorf("password is required") // Returns clear error for missing password.
	}
	hash, _, _, _, err := s.repo.GetAuthData(ctx, id) // Fetches stored hash for this user.
	if err != nil {                                   // Handles repository failures like not-found.
		return err // Propagates error upward.
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); compareErr != nil { // Compares submitted password against stored hash.
		return fmt.Errorf("invalid credentials") // Returns generic auth failure message.
	}
	return s.repo.MarkLogin(ctx, id, time.Now().UTC()) // Records successful login time in database.
}

func (s *Service) LoginByEmail(ctx context.Context, email, password string) (int64, error) { // Verifies email/password credentials and returns authenticated user ID.
	if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" { // Ensures both credentials are provided.
		return 0, fmt.Errorf("email and password are required") // Returns validation error for missing credentials.
	}
	userID, hash, err := s.repo.GetAuthDataByEmail(ctx, email) // Loads user ID and password hash using unique email.
	if err != nil {                                            // Handles repository failures.
		if errors.Is(err, ErrUserNotFound) { // Normalizes unknown email to generic auth error.
			return 0, fmt.Errorf("invalid credentials") // Prevents user-enumeration via detailed errors.
		}
		return 0, err // Propagates unexpected repository errors.
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); compareErr != nil { // Compares submitted password to stored hash.
		return 0, fmt.Errorf("invalid credentials") // Returns generic auth error on mismatch.
	}
	if err := s.repo.MarkLogin(ctx, userID, time.Now().UTC()); err != nil { // Marks successful login timestamp for audit.
		return 0, err // Propagates mark-login persistence error.
	}
	return userID, nil // Returns authenticated user ID for token issuance.
}

func (s *Service) SendTOTP(ctx context.Context, id int64) error { // Generates a current TOTP code for demonstration flow.
	_, secret, email, phone, err := s.repo.GetAuthData(ctx, id) // Reads TOTP secret and contact values from DB.
	if err != nil {                                             // Handles missing user or query errors.
		return err // Propagates repository error.
	}

	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{ // Generates one TOTP code using stored secret and current UTC time.
		Period:    60,                // Code remains valid for 60 seconds.
		Skew:      0,                 // Disallows clock-window tolerance in this learning example.
		Digits:    otp.DigitsSix,     // Produces six-digit verification code.
		Algorithm: otp.AlgorithmSHA1, // Uses SHA1 as standard TOTP algorithm.
	})
	if err != nil { // Handles generation failure.
		return fmt.Errorf("generate totp code: %w", err) // Wraps error with operation context.
	}

	log.Printf("2FA code for user_id=%d email=%s phone=%s code=%s valid_for_seconds=60", id, email, phone, code) // Prints demo code to server logs.
	return nil                                                                                                   // Returns success after simulated send step.
}

func (s *Service) VerifyTOTP(ctx context.Context, id int64, code string) error { // Validates submitted TOTP code against stored secret.
	if strings.TrimSpace(code) == "" { // Ensures client provided a code value.
		return fmt.Errorf("code is required") // Returns validation error for empty input.
	}
	_, secret, _, _, err := s.repo.GetAuthData(ctx, id) // Reads TOTP secret from repository for this user.
	if err != nil {                                     // Handles DB/not-found errors.
		return err // Propagates repository error.
	}

	ok, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{ // Validates code using same options as generation.
		Period:    60,                // Expects 60-second step duration.
		Skew:      0,                 // Requires exact current time step in this strict example.
		Digits:    otp.DigitsSix,     // Expects six-digit token length.
		Algorithm: otp.AlgorithmSHA1, // Expects SHA1-based token generation.
	})
	if err != nil { // Handles validation engine errors.
		return fmt.Errorf("validate totp code: %w", err) // Wraps technical validation failure.
	}
	if !ok { // Checks boolean result when code does not match.
		return errors.New("invalid or expired code") // Returns domain error for incorrect/expired code.
	}
	return nil // Signals successful 2FA verification.
}

func hashPassword(raw string) (string, error) { // Helper converts plaintext password into bcrypt hash string.
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost) // Generates secure hash bytes with default work factor.
	if err != nil {                                                            // Handles rare hashing/runtime errors.
		return "", fmt.Errorf("hash password: %w", err) // Wraps and returns hash operation error.
	}
	return string(bytes), nil // Converts hash bytes to string for DB storage.
}

func generateSecret(email string) (string, error) { // Helper creates a new TOTP secret for a user account.
	key, err := totp.Generate(totp.GenerateOpts{ // Builds a TOTP key object using provided options.
		Issuer:      "GoCRMLearningAPI", // Issuer label shown in authenticator apps.
		AccountName: email,              // Account label tied to this user's email.
		Period:      60,                 // Sets token period to 60 seconds.
		SecretSize:  20,                 // Uses 20-byte secret size for key material.
		Digits:      otp.DigitsSix,      // Configures six-digit codes.
		Algorithm:   otp.AlgorithmSHA1,  // Configures SHA1 algorithm for compatibility.
	})
	if err != nil { // Handles secret generation failure.
		return "", fmt.Errorf("generate totp secret: %w", err) // Wraps generation error.
	}
	return key.Secret(), nil // Returns secret string saved into database.
}
