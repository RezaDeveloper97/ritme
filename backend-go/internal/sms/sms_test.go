package sms

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/config"
)

func cfg(env, provider string) *config.Config {
	return &config.Config{App: config.App{Env: env}, Companion: config.Companion{SMSProvider: provider, InviteTemplate: "companion-invite"}}
}

func TestNew_Providers(t *testing.T) {
	p, err := New(cfg("local", ""), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, config.CompanionSMSFake, p.Name(), "fake outside production")

	p, err = New(cfg("production", ""), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, config.CompanionSMSNone, p.Name(), "none in production")
	assert.False(t, p.Delivers())
	require.ErrorIs(t, p.SendInvite(context.Background(), "09121234567", "RT7K2M", true), ErrNotSent)

	_, err = New(cfg("production", "fake"), nil, nil)
	require.Error(t, err)

	p, err = New(cfg("production", "gateway"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, config.CompanionSMSGateway, p.Name())
	assert.True(t, p.Delivers())

	_, err = New(cfg("local", "pigeon"), nil, nil)
	require.Error(t, err)
}

func TestFake_LogsMaskedNumberNeverTheCode(t *testing.T) {
	var buf bytes.Buffer
	f := NewFake(slog.New(slog.NewTextHandler(&buf, nil)))
	require.NoError(t, f.SendInvite(context.Background(), "09121234567", "RT7K2M", false))
	assert.Contains(t, buf.String(), "0912****567")
	assert.NotContains(t, buf.String(), "09121234567")
	assert.NotContains(t, buf.String(), "RT7K2M")
	assert.Equal(t, []string{"09121234567"}, f.Sent())
	assert.Empty(t, f.Sent())
}

type recorder struct{ templates []string }

func (*recorder) Name() string { return "rec" }

func (r *recorder) SendOTP(_ context.Context, _, _, template string) error {
	r.templates = append(r.templates, template)
	return nil
}

// CB-PRIV-01: a discreet owner's invite goes out with the neutral template when the deployment has one.
func TestGateway_DiscreetPicksNeutralTemplate(t *testing.T) {
	rec := &recorder{}
	g := &Gateway{provider: rec, template: "companion-invite", neutral: "companion-invite-neutral"}
	require.NoError(t, g.SendInvite(context.Background(), "09121234567", "RT7K2M", true))
	require.NoError(t, g.SendInvite(context.Background(), "09121234567", "RT7K2M", false))
	assert.Equal(t, []string{"companion-invite-neutral", "companion-invite"}, rec.templates)

	// No neutral template configured: a discreet owner's invite is not sent at all (no fallback), a plain one is.
	g.neutral = ""
	require.ErrorIs(t, g.SendInvite(context.Background(), "09121234567", "RT7K2M", true), ErrNotSent)
	require.NoError(t, g.SendInvite(context.Background(), "09121234567", "RT7K2M", false))
	assert.Equal(t, []string{"companion-invite-neutral", "companion-invite", "companion-invite"}, rec.templates)

	f := NewFake(nil)
	require.NoError(t, f.SendInvite(context.Background(), "09121234567", "RT7K2M", true))
	assert.Equal(t, []Invite{{Mobile: "09121234567", Discreet: true}}, f.Invites())
}
