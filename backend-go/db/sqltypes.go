package db

import (
	"database/sql"
	"encoding/json"
)

// NullRawJSON is what sqlc generates for NULL-able JSON columns (sqlc.yaml overrides).
// A bare json.RawMessage cannot scan SQL NULL, and sqlc's go_type cannot spell a generic type,
// hence this alias. Valid=false ⇔ the column is NULL; V holds the raw JSON otherwise.
type NullRawJSON = sql.Null[json.RawMessage]
