// Package auditchain makes the admin audit log (admin_audit_logs) tamper-evident (SPEC v6 §5.8, T-M10-33, D-182).
//
// Each row carries prev_hash (the row_hash of the row before it) and
//
//	row_hash = hex(sha256(prev_hash ‖ canonical JSON of admin_id, action, target_type, target_id, ip, detail, created_at))
//
// The id is not hashed: the row cannot change after the insert (the 00105 BEFORE UPDATE / DELETE triggers), and the
// writer, not the database, sets created_at, so the hash is known before the insert.
//
// Append serialises writers on the one row of admin_audit_chain_head (SELECT … FOR UPDATE) inside one short
// transaction: lock the head, hash, insert with prev_hash = last_hash, move the head, commit. Two concurrent appends
// therefore never read the same last_hash, so the chain cannot fork, and ids grow in chain order.
//
// Rows written before the chain started keep their NULL hashes. Genesis folds them, oldest first, from GenesisSeed
// into genesis_hash (and their last id into genesis_last_id); the chain continues from genesis_hash. The first Append
// runs it when it has not run yet, and the nightly admin.audit_verify job runs it too (it is idempotent).
//
// Verify walks the rows by id and reports the first break: a legacy row edited or deleted (genesis_mismatch), a row
// without hashes after genesis (unchained), a link that does not match the previous row (prev_mismatch — a deleted
// or reordered row), a row whose content no longer matches its hash (hash_mismatch — an edited row), or a head that
// does not end on the last row (head_mismatch — a deleted tail). A rewrite of every later row and the head is only
// caught against an anchor outside the database: the nightly job writes the head hash to the structured log.
package auditchain

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// GenesisSeed is where the genesis fold starts (also the genesis hash of an empty log).
var GenesisSeed = func() string {
	s := sha256.Sum256([]byte("ritme:admin_audit_logs:genesis:v1"))
	return hex.EncodeToString(s[:])
}()

// timeLayout is created_at in the canonical form (Asia/Tehran wall clock, whole seconds, like the column).
const timeLayout = "2006-01-02 15:04:05"

// batch is how many rows one walk step reads.
const batch = 1000

// ErrNoHead means the admin_audit_chain_head singleton row is missing (migration 00105 seeds it).
var ErrNoHead = errors.New("auditchain: admin_audit_chain_head row missing")

// Row is the hashed content of one audit row.
type Row struct {
	AdminID    sql.NullInt64
	Action     string
	TargetType string
	TargetID   uint64
	IP         string
	Detail     sql.NullString
	CreatedAt  sql.NullTime
}

// RowOf is the hashed content of a stored row.
func RowOf(r store.AdminAuditLog) Row {
	return Row{AdminID: r.AdminID, Action: r.Action, TargetType: r.TargetType, TargetID: r.TargetID, IP: r.Ip,
		Detail: r.Detail, CreatedAt: r.CreatedAt}
}

// canonical has a fixed field order; detail is hashed as the stored text (not re-encoded).
type canonical struct {
	AdminID    *int64  `json:"admin_id"`
	Action     string  `json:"action"`
	TargetType string  `json:"target_type"`
	TargetID   uint64  `json:"target_id"`
	IP         string  `json:"ip"`
	Detail     *string `json:"detail"`
	CreatedAt  *string `json:"created_at"`
}

// Canonical is the row's canonical JSON (the hashed bytes).
func Canonical(r Row) []byte {
	c := canonical{Action: r.Action, TargetType: r.TargetType, TargetID: r.TargetID, IP: r.IP}
	if r.AdminID.Valid {
		c.AdminID = &r.AdminID.Int64
	}
	if r.Detail.Valid {
		c.Detail = &r.Detail.String
	}
	if r.CreatedAt.Valid {
		s := r.CreatedAt.Time.In(civildate.Tehran).Format(timeLayout)
		c.CreatedAt = &s
	}
	b, err := json.Marshal(c)
	if err != nil { // a struct of strings and numbers always encodes
		panic(err)
	}
	return b
}

// Hash is hex(sha256(prev ‖ Canonical(r))).
func Hash(prev string, r Row) string {
	h := sha256.New()
	h.Write([]byte(prev))
	h.Write(Canonical(r))
	return hex.EncodeToString(h.Sum(nil))
}

// Chain appends to and verifies the log.
type Chain struct {
	db *sql.DB
}

// New wires the chain on db.
func New(db *sql.DB) *Chain { return &Chain{db: db} }

// Append writes r as the next chained row and returns its id. Any error leaves nothing written.
func (c *Chain) Append(ctx context.Context, r Row) (uint64, error) {
	var id uint64
	err := c.inTx(ctx, func(q *store.Queries) error {
		head, err := lockHead(ctx, q)
		if err != nil {
			return err
		}
		if !head.GenesisLastID.Valid {
			if head, err = genesisLocked(ctx, q); err != nil {
				return err
			}
		}
		prev := head.LastHash.String
		h := Hash(prev, r)
		lastID, err := q.InsertChainedAdminAuditLog(ctx, store.InsertChainedAdminAuditLogParams{
			AdminID: r.AdminID, Action: r.Action, TargetType: r.TargetType, TargetID: r.TargetID, Ip: r.IP,
			Detail: r.Detail, CreatedAt: r.CreatedAt,
			PrevHash: sql.NullString{String: prev, Valid: true}, RowHash: sql.NullString{String: h, Valid: true},
		})
		if err != nil {
			return err
		}
		if err := q.MoveAdminAuditChainHead(ctx, store.MoveAdminAuditChainHeadParams{
			LastID: sql.NullInt64{Int64: lastID, Valid: true}, LastHash: sql.NullString{String: h, Valid: true},
		}); err != nil {
			return err
		}
		id = uint64(lastID) //nolint:gosec // auto-increment id
		return nil
	})
	return id, err
}

// EnsureGenesis hashes the pre-chain rows into the head once (a no-op when that already happened).
func (c *Chain) EnsureGenesis(ctx context.Context) error {
	return c.inTx(ctx, func(q *store.Queries) error {
		head, err := lockHead(ctx, q)
		if err != nil || head.GenesisLastID.Valid {
			return err
		}
		_, err = genesisLocked(ctx, q)
		return err
	})
}

func (c *Chain) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(store.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func lockHead(ctx context.Context, q *store.Queries) (store.AdminAuditChainHead, error) {
	head, err := q.LockAdminAuditChainHead(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return head, ErrNoHead
	}
	return head, err
}

// genesisLocked folds every row (all written before the chain started) into the head. The head is locked.
func genesisLocked(ctx context.Context, q *store.Queries) (store.AdminAuditChainHead, error) {
	h, last, _, err := fold(ctx, q, 0, func(uint64) bool { return true })
	if err != nil {
		return store.AdminAuditChainHead{}, err
	}
	p := store.SetAdminAuditChainGenesisParams{
		GenesisLastID: sql.NullInt64{Int64: int64(last), Valid: true}, //nolint:gosec // ids fit int64
		GenesisHash:   sql.NullString{String: h, Valid: true},
	}
	if err := q.SetAdminAuditChainGenesis(ctx, p); err != nil {
		return store.AdminAuditChainHead{}, err
	}
	return store.AdminAuditChainHead{ID: 1, LastID: p.GenesisLastID, LastHash: p.GenesisHash,
		GenesisLastID: p.GenesisLastID, GenesisHash: p.GenesisHash}, nil
}

// fold hashes the rows after afterID while keep(id) holds, from GenesisSeed: the hash, the last folded id and the
// number of rows.
func fold(ctx context.Context, q *store.Queries, afterID uint64, keep func(uint64) bool) (string, uint64, int, error) {
	h, last, n := GenesisSeed, afterID, 0
	for {
		rows, err := q.ListAdminAuditLogsForChain(ctx, store.ListAdminAuditLogsForChainParams{AfterID: last, Limit: batch})
		if err != nil {
			return "", 0, 0, err
		}
		for _, r := range rows {
			if !keep(r.ID) {
				return h, last, n, nil
			}
			h, last, n = Hash(h, RowOf(r)), r.ID, n+1
		}
		if len(rows) < batch {
			return h, last, n, nil
		}
	}
}

// Break reasons.
const (
	BreakGenesis   = "genesis_mismatch" // a pre-chain row was edited or deleted
	BreakUnchained = "unchained"        // a row without hashes after genesis
	BreakPrev      = "prev_mismatch"    // prev_hash is not the previous row's row_hash (a row was deleted)
	BreakHash      = "hash_mismatch"    // the row's content no longer matches its row_hash (it was edited)
	BreakHead      = "head_mismatch"    // the head does not end on the last row (the tail was deleted or rewritten)
)

// Break is the first place the chain does not hold.
type Break struct {
	ID     uint64 `json:"id"`
	Reason string `json:"reason"`
}

// Report is the result of a verification.
type Report struct {
	OK            bool   `json:"ok"`
	Started       bool   `json:"started"` // genesis has run
	From          uint64 `json:"from"`
	Checked       int    `json:"checked"`
	GenesisLastID uint64 `json:"genesis_last_id"`
	HeadID        uint64 `json:"head_id"`
	HeadHash      string `json:"head_hash"`
	Break         *Break `json:"break"`
}

// Verify walks the rows from id `from` (0 = everything, including the genesis fold) inside one read-only snapshot.
func (c *Chain) Verify(ctx context.Context, from uint64) (Report, error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = tx.Rollback() }()
	rep, err := verify(ctx, store.New(tx), from)
	if err != nil {
		return Report{}, err
	}
	return rep, tx.Commit()
}

func verify(ctx context.Context, q *store.Queries, from uint64) (Report, error) {
	rep := Report{From: from}
	head, err := q.GetAdminAuditChainHead(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return rep, ErrNoHead
	}
	if err != nil {
		return rep, err
	}
	rep.HeadID, rep.HeadHash = uint64(head.LastID.Int64), head.LastHash.String //nolint:gosec // ids fit
	if !head.GenesisLastID.Valid {
		rep.OK = true // nothing chained yet
		return rep, nil
	}
	rep.Started = true
	gl := uint64(head.GenesisLastID.Int64) //nolint:gosec // ids fit
	rep.GenesisLastID = gl
	fail := func(id uint64, reason string) (Report, error) {
		rep.Break = &Break{ID: id, Reason: reason}
		return rep, nil
	}

	expected, after := head.GenesisHash.String, gl
	if from <= gl+1 {
		h, _, n, ferr := fold(ctx, q, 0, func(id uint64) bool { return id <= gl })
		if ferr != nil {
			return rep, ferr
		}
		rep.Checked += n
		if h != head.GenesisHash.String {
			return fail(gl, BreakGenesis)
		}
	} else {
		anchor, aerr := q.LastAdminAuditLogBefore(ctx, from)
		switch {
		case errors.Is(aerr, sql.ErrNoRows): // no row before from: the chain starts at genesis
		case aerr != nil:
			return rep, aerr
		case anchor.ID <= gl:
		case !anchor.RowHash.Valid:
			return fail(anchor.ID, BreakUnchained)
		default:
			expected, after = anchor.RowHash.String, anchor.ID
		}
	}

	headID := rep.HeadID
	lastID := after
	for {
		rows, lerr := q.ListAdminAuditLogsForChain(ctx, store.ListAdminAuditLogsForChainParams{AfterID: after, Limit: batch})
		if lerr != nil {
			return rep, lerr
		}
		for _, r := range rows {
			after = r.ID
			if !r.PrevHash.Valid || !r.RowHash.Valid {
				return fail(r.ID, BreakUnchained)
			}
			if r.ID > headID { // a chained row past the head: the head was moved back
				return fail(r.ID, BreakHead)
			}
			if r.PrevHash.String != expected {
				return fail(r.ID, BreakPrev)
			}
			if Hash(r.PrevHash.String, RowOf(r)) != r.RowHash.String {
				return fail(r.ID, BreakHash)
			}
			expected, lastID = r.RowHash.String, r.ID
			rep.Checked++
		}
		if len(rows) < batch {
			break
		}
	}
	if lastID != headID || expected != head.LastHash.String {
		return fail(headID, BreakHead)
	}
	rep.OK = true
	return rep, nil
}

// String is a one-line summary for logs (no row content).
func (r Report) String() string {
	if r.Break != nil {
		return fmt.Sprintf("broken at %d (%s)", r.Break.ID, r.Break.Reason)
	}
	return fmt.Sprintf("ok, %d rows, head %d", r.Checked, r.HeadID)
}
