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
	require.ErrorIs(t, p.SendInvite(context.Background(), "09121234567", "RT7K2M"), ErrNotSent)

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
	require.NoError(t, f.SendInvite(context.Background(), "09121234567", "RT7K2M"))
	assert.Contains(t, buf.String(), "0912****567")
	assert.NotContains(t, buf.String(), "09121234567")
	assert.NotContains(t, buf.String(), "RT7K2M")
	assert.Equal(t, []string{"09121234567"}, f.Sent())
	assert.Empty(t, f.Sent())
}
