package customers // Defines customer-domain data shapes shared by handler, service, and repository.

import "time" // Uses time.Time for created_at and updated_at fields.

type Customer struct { // Represents one customer record returned by the API.
	ID        int64     `json:"id"`         // Database primary key for customer row.
	Name      string    `json:"name"`       // Customer display name.
	Email     string    `json:"email"`      // Customer email, used as unique contact.
	Phone     string    `json:"phone"`      // Customer phone contact.
	CreatedAt time.Time `json:"created_at"` // Timestamp when customer was created.
	UpdatedAt time.Time `json:"updated_at"` // Timestamp when customer was last updated.
}

type CreateCustomerInput struct { // Request payload for create endpoint.
	Name  string `json:"name"`  // Name submitted by client.
	Email string `json:"email"` // Email submitted by client.
	Phone string `json:"phone"` // Phone submitted by client.
}

type UpdateCustomerInput struct { // Request payload for update endpoint.
	Name  string `json:"name"`  // Updated name submitted by client.
	Email string `json:"email"` // Updated email submitted by client.
	Phone string `json:"phone"` // Updated phone submitted by client.
}
