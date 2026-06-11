package testutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

// FakeCustomerStore is an in-memory customer store for auth tests.
type FakeCustomerStore struct {
	Customers map[string]*db.Customer
	Orders    map[string]*db.Order
}

func NewFakeCustomerStore() *FakeCustomerStore {
	return &FakeCustomerStore{
		Customers: make(map[string]*db.Customer),
		Orders:    make(map[string]*db.Order),
	}
}

func (f *FakeCustomerStore) CreateCustomer(_ context.Context, id, email, passwordHash string) (*db.Customer, error) {
	for _, c := range f.Customers {
		if strings.EqualFold(c.Email, email) {
			return nil, fmt.Errorf("duplicate email")
		}
	}
	customer := &db.Customer{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	f.Customers[id] = customer
	return customer, nil
}

func (f *FakeCustomerStore) GetCustomerByEmail(_ context.Context, email string) (*db.Customer, error) {
	for _, c := range f.Customers {
		if strings.EqualFold(c.Email, email) {
			return c, nil
		}
	}
	return nil, nil
}

func (f *FakeCustomerStore) GetCustomerByID(_ context.Context, id string) (*db.Customer, error) {
	if c := f.Customers[id]; c != nil {
		return c, nil
	}
	return nil, nil
}

func (f *FakeCustomerStore) LinkOrdersToCustomer(_ context.Context, customerID, email string) error {
	for _, order := range f.Orders {
		if order.CustomerID == nil && order.CustomerEmail != nil &&
			strings.EqualFold(*order.CustomerEmail, email) {
			id := customerID
			order.CustomerID = &id
		}
	}
	return nil
}
