package testutil

import "context"

type FakeHealth struct {
	Err error
}

func (f FakeHealth) Ping(_ context.Context) error {
	return f.Err
}
