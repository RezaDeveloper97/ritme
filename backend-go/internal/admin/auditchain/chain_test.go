package auditchain

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func sample() Row {
	return Row{
		AdminID: sql.NullInt64{Int64: 3, Valid: true}, Action: "user.block", TargetType: "user", TargetID: 11,
		IP: "10.0.0.1", Detail: sql.NullString{String: `{"b":1,"a":"x"}`, Valid: true},
		CreatedAt: sql.NullTime{Time: time.Date(2026, 10, 29, 9, 30, 5, 0, civildate.Tehran), Valid: true},
	}
}

func TestCanonical_FixedOrderAndForm(t *testing.T) {
	assert.Equal(t, `{"admin_id":3,"action":"user.block","target_type":"user","target_id":11,"ip":"10.0.0.1",`+
		`"detail":"{\"b\":1,\"a\":\"x\"}","created_at":"2026-10-29 09:30:05"}`, string(Canonical(sample())))
	// NULLs are explicit; created_at is the Tehran wall clock whatever the time's location.
	r := sample()
	r.AdminID, r.Detail = sql.NullInt64{}, sql.NullString{}
	r.CreatedAt.Time = r.CreatedAt.Time.UTC()
	assert.Equal(t, `{"admin_id":null,"action":"user.block","target_type":"user","target_id":11,"ip":"10.0.0.1",`+
		`"detail":null,"created_at":"2026-10-29 09:30:05"}`, string(Canonical(r)))
}

func TestHash_ChainsOnPrevAndContent(t *testing.T) {
	h := Hash(GenesisSeed, sample())
	assert.Len(t, h, 64)
	assert.Equal(t, h, Hash(GenesisSeed, sample()), "deterministic")
	assert.NotEqual(t, h, Hash("", sample()), "prev is hashed")
	for name, mut := range map[string]func(*Row){
		"admin":   func(r *Row) { r.AdminID.Int64 = 4 },
		"action":  func(r *Row) { r.Action = "user.unblock" },
		"type":    func(r *Row) { r.TargetType = "admin" },
		"target":  func(r *Row) { r.TargetID = 12 },
		"ip":      func(r *Row) { r.IP = "10.0.0.2" },
		"detail":  func(r *Row) { r.Detail.String = `{"b":2,"a":"x"}` },
		"created": func(r *Row) { r.CreatedAt.Time = r.CreatedAt.Time.Add(time.Second) },
		"null":    func(r *Row) { r.Detail.Valid = false },
	} {
		r := sample()
		mut(&r)
		assert.NotEqual(t, h, Hash(GenesisSeed, r), name)
	}
	assert.Len(t, GenesisSeed, 64)
}
