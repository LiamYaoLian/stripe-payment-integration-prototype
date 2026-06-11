package testutil

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

// FakeAuthStore is an in-memory auth store for tests.
type FakeAuthStore struct {
	mu                       sync.Mutex
	Customers                map[string]*db.Customer
	CustomersByEmail         map[string]string
	SessionsByTokenHash      map[string]*db.UserSession
	SessionsByID             map[string]*db.UserSession
	PasswordResetByTokenHash map[string]string
	EmailVerifyByTokenHash   map[string]string
	RateLimitBuckets         map[string]int
}

func NewFakeAuthStore() *FakeAuthStore {
	return &FakeAuthStore{
		Customers:                make(map[string]*db.Customer),
		CustomersByEmail:         make(map[string]string),
		SessionsByTokenHash:      make(map[string]*db.UserSession),
		SessionsByID:             make(map[string]*db.UserSession),
		PasswordResetByTokenHash: make(map[string]string),
		EmailVerifyByTokenHash:   make(map[string]string),
		RateLimitBuckets:         make(map[string]int),
	}
}

func (f *FakeAuthStore) CreateCustomer(_ context.Context, id, email, passwordHash string) (*db.Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, customerID := range f.CustomersByEmail {
		if strings.EqualFold(f.Customers[customerID].Email, email) {
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
	f.CustomersByEmail[strings.ToLower(email)] = id
	return customer, nil
}

func (f *FakeAuthStore) GetCustomerByEmail(_ context.Context, email string) (*db.Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id := f.CustomersByEmail[strings.ToLower(email)]; id != "" {
		return f.Customers[id], nil
	}
	return nil, nil
}

func (f *FakeAuthStore) GetCustomerByID(_ context.Context, id string) (*db.Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if customer := f.Customers[id]; customer != nil {
		return customer, nil
	}
	return nil, nil
}

func (f *FakeAuthStore) UpdateCustomerPassword(_ context.Context, customerID, passwordHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if customer := f.Customers[customerID]; customer != nil {
		customer.PasswordHash = passwordHash
		return nil
	}
	return fmt.Errorf("customer not found")
}

func (f *FakeAuthStore) MarkCustomerEmailVerified(_ context.Context, customerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if customer := f.Customers[customerID]; customer != nil {
		now := time.Now().UTC()
		customer.EmailVerifiedAt = &now
		return nil
	}
	return fmt.Errorf("customer not found")
}

func (f *FakeAuthStore) CreateUserSession(_ context.Context, id, tokenHash, customerID string, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	customer := f.Customers[customerID]
	if customer == nil {
		return fmt.Errorf("customer not found")
	}
	session := &db.UserSession{
		ID:         id,
		CustomerID: customerID,
		Email:      customer.Email,
		ExpiresAt:  expiresAt,
	}
	f.SessionsByID[id] = session
	f.SessionsByTokenHash[tokenHash] = session
	return nil
}

func (f *FakeAuthStore) GetUserSessionByTokenHash(_ context.Context, tokenHash string) (*db.UserSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	session := f.SessionsByTokenHash[tokenHash]
	if session == nil || session.ExpiresAt.Before(time.Now().UTC()) {
		return nil, nil
	}
	return session, nil
}

func (f *FakeAuthStore) TouchUserSession(_ context.Context, sessionID string, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if session := f.SessionsByID[sessionID]; session != nil {
		session.ExpiresAt = expiresAt
	}
	return nil
}

func (f *FakeAuthStore) RevokeUserSession(_ context.Context, sessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.SessionsByID, sessionID)
	for hash, session := range f.SessionsByTokenHash {
		if session.ID == sessionID {
			delete(f.SessionsByTokenHash, hash)
		}
	}
	return nil
}

func (f *FakeAuthStore) RevokeAllUserSessions(_ context.Context, customerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for hash, session := range f.SessionsByTokenHash {
		if session.CustomerID == customerID {
			delete(f.SessionsByTokenHash, hash)
			delete(f.SessionsByID, session.ID)
		}
	}
	return nil
}

func (f *FakeAuthStore) CreatePasswordResetToken(_ context.Context, _, tokenHash, customerID string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PasswordResetByTokenHash[tokenHash] = customerID
	return nil
}

func (f *FakeAuthStore) ConsumePasswordResetToken(_ context.Context, tokenHash string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	customerID := f.PasswordResetByTokenHash[tokenHash]
	delete(f.PasswordResetByTokenHash, tokenHash)
	return customerID, nil
}

func (f *FakeAuthStore) CreateEmailVerificationToken(_ context.Context, _, tokenHash, customerID string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.EmailVerifyByTokenHash[tokenHash] = customerID
	return nil
}

func (f *FakeAuthStore) ConsumeEmailVerificationToken(_ context.Context, tokenHash string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	customerID := f.EmailVerifyByTokenHash[tokenHash]
	delete(f.EmailVerifyByTokenHash, tokenHash)
	return customerID, nil
}

func (f *FakeAuthStore) IncrementRateLimitBucket(_ context.Context, bucketKey string, _ time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RateLimitBuckets[bucketKey]++
	return f.RateLimitBuckets[bucketKey], nil
}

// StorePasswordResetToken helps tests seed reset tokens.
func (f *FakeAuthStore) StorePasswordResetToken(token, customerID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PasswordResetByTokenHash[auth.HashOpaqueToken(token)] = customerID
}
