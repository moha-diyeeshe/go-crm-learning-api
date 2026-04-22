package customers // Contains business validation and orchestration for customers module.

import (
	"context" // Uses context.Context for request-scoped cancellation and deadlines.
	"fmt"     // Uses fmt.Errorf for clear validation and wrapped error messages.
	"strings" // Uses strings.TrimSpace for required-field validation.
)

type Service struct { // Holds dependencies used by customer business logic.
	repo *Repository // Repository dependency used for all DB persistence.
}

func NewService(repo *Repository) *Service { // Constructor for dependency injection from main.go.
	return &Service{repo: repo} // Returns shared service pointer.
}

func (s *Service) CreateCustomer(ctx context.Context, in CreateCustomerInput) (Customer, error) { // Validates create payload and persists new customer.
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Phone) == "" { // Ensures required fields are present.
		return Customer{}, fmt.Errorf("name, email and phone are required") // Returns validation error for missing fields.
	}
	return s.repo.Create(ctx, in) // Delegates actual insert to repository.
}

func (s *Service) ListCustomers(ctx context.Context) ([]Customer, error) { // Returns all customers for list endpoint.
	return s.repo.List(ctx) // Delegates list query to repository.
}

func (s *Service) GetCustomer(ctx context.Context, id int64) (Customer, error) { // Returns one customer by ID.
	return s.repo.GetByID(ctx, id) // Delegates lookup to repository.
}

func (s *Service) UpdateCustomer(ctx context.Context, id int64, in UpdateCustomerInput) (Customer, error) { // Validates update payload and persists changes.
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Phone) == "" { // Ensures required fields remain populated.
		return Customer{}, fmt.Errorf("name, email and phone are required") // Returns validation error on bad payload.
	}
	return s.repo.Update(ctx, id, in) // Delegates update operation to repository.
}

func (s *Service) DeleteCustomer(ctx context.Context, id int64) error { // Deletes one customer by ID.
	return s.repo.Delete(ctx, id) // Delegates delete operation to repository.
}
