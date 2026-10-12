package order

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/shop/store"
)

func TestPII_FailsClosedWithoutKey(t *testing.T) {
	disabled, err := loss.NewNoteBox(nil, nil, true)
	require.NoError(t, err)
	for name, svc := range map[string]*Service{"nil": NewService(nil), "disabled": NewService(nil).WithSealer(disabled)} {
		_, err := svc.SaveAddress(context.Background(), 1, 0, AddressInput{RecipientName: "S", RecipientPhone: "09120000000",
			City: "T", AddressLine: "x"}, time.Now())
		assert.True(t, IsStorage(err), name+": no write without the key")
		_, err = svc.OpenAddress(store.ShopAddress{UserID: 1, PiiEnc: sql.NullString{String: "blob", Valid: true}})
		assert.True(t, IsStorage(err), name)
		_, err = svc.OpenOrder(store.ShopOrder{UserID: 1, PiiEnc: sql.NullString{String: "blob", Valid: true}})
		assert.True(t, IsStorage(err), name)
		legacy, err := svc.OpenAddress(store.ShopAddress{UserID: 1, RecipientPhone: "09120000000"})
		require.NoError(t, err, name+": a legacy plain row needs no key")
		assert.Equal(t, "09120000000", legacy.RecipientPhone)
	}
}

func TestPII_RoundTripBoundToOwner(t *testing.T) {
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	svc := NewService(nil).WithSealer(box)
	enc, err := svc.sealJSON(orderPII{Name: "Sara", Phone: "09121234567", Address: "Valiasr", PostalCode: "1234567890"}, orderAD(7))
	require.NoError(t, err)
	assert.NotContains(t, enc.String, "0912")
	o, err := svc.OpenOrder(store.ShopOrder{UserID: 7, PiiEnc: enc})
	require.NoError(t, err)
	assert.Equal(t, "Sara", o.RecipientName)
	assert.Equal(t, "09121234567", o.RecipientPhone)
	assert.Equal(t, "Valiasr", o.AddressText)
	assert.Equal(t, "1234567890", o.PostalCode.String)
	_, err = svc.OpenOrder(store.ShopOrder{UserID: 8, PiiEnc: enc})
	assert.True(t, IsStorage(err), "another owner's row does not open the blob")
}
