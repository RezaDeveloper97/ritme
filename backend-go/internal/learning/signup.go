package learning

import (
	"context"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// SignupHook is the auth signup callback (auth.Handlers.OnSignup): the new account's pending grants become active,
// their duration counted from now (the request clock). auth bounds and isolates the call, so an error here never
// fails the login.
func (s *Service) SignupHook() func(ctx context.Context, userID uint64, mobile string, now time.Time) error {
	return func(ctx context.Context, userID uint64, mobile string, now time.Time) error {
		_, err := s.ClaimPending(ctx, userID, mobile, now.In(civildate.Tehran).Truncate(time.Second))
		return err
	}
}
