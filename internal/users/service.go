package users

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (User, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Phone) == "" || strings.TrimSpace(in.Password) == "" {
		return User{}, fmt.Errorf("name, email, phone and password are required")
	}

	passwordHash, err := hashPassword(in.Password)
	if err != nil {
		return User{}, err
	}

	secret, err := generateSecret(in.Email)
	if err != nil {
		return User{}, err
	}

	return s.repo.Create(ctx, in, passwordHash, secret)
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetUser(ctx context.Context, id int64) (User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateUser(ctx context.Context, id int64, in UpdateUserInput) (User, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Phone) == "" {
		return User{}, fmt.Errorf("name, email and phone are required")
	}
	return s.repo.Update(ctx, id, in)
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ChangePassword(ctx context.Context, id int64, oldPassword, newPassword string) error {
	if strings.TrimSpace(oldPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return fmt.Errorf("old_password and new_password are required")
	}
	hash, _, _, _, err := s.repo.GetAuthData(ctx, id)
	if err != nil {
		return err
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); compareErr != nil {
		return fmt.Errorf("old password is incorrect")
	}

	newHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, newHash, time.Now().UTC())
}

func (s *Service) Login(ctx context.Context, id int64, password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required")
	}
	hash, _, _, _, err := s.repo.GetAuthData(ctx, id)
	if err != nil {
		return err
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); compareErr != nil {
		return fmt.Errorf("invalid credentials")
	}
	return s.repo.MarkLogin(ctx, id, time.Now().UTC())
}

func (s *Service) SendTOTP(ctx context.Context, id int64) error {
	_, secret, email, phone, err := s.repo.GetAuthData(ctx, id)
	if err != nil {
		return err
	}

	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    60,
		Skew:      0,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return fmt.Errorf("generate totp code: %w", err)
	}

	log.Printf("2FA code for user_id=%d email=%s phone=%s code=%s valid_for_seconds=60", id, email, phone, code)
	return nil
}

func (s *Service) VerifyTOTP(ctx context.Context, id int64, code string) error {
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("code is required")
	}
	_, secret, _, _, err := s.repo.GetAuthData(ctx, id)
	if err != nil {
		return err
	}

	ok, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    60,
		Skew:      0,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return fmt.Errorf("validate totp code: %w", err)
	}
	if !ok {
		return errors.New("invalid or expired code")
	}
	return nil
}

func hashPassword(raw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

func generateSecret(email string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GoCRMLearningAPI",
		AccountName: email,
		Period:      60,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return key.Secret(), nil
}
