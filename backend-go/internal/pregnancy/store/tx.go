package store

// Hand-written (not sqlc output; `make sqlc` leaves it alone): lets a service that only holds a
// *Queries run several writes in one transaction without a separate *sql.DB dependency.

import (
	"context"
	"database/sql"
)

// BeginTx starts a transaction on the *sql.DB behind q and returns it with the Queries bound
// to it. When q cannot begin one (it already runs on a *sql.Tx, or on a test double) tx is nil
// and qtx is q itself, so callers write through qtx either way and commit only a non-nil tx.
func (q *Queries) BeginTx(ctx context.Context) (tx *sql.Tx, qtx *Queries, err error) {
	b, ok := q.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return nil, q, nil
	}
	if tx, err = b.BeginTx(ctx, nil); err != nil {
		return nil, nil, err
	}
	return tx, q.WithTx(tx), nil
}
