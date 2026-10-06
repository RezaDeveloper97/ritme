package healthrecord

import (
	"context"
	"time"
)

// AttachFilesForTest runs the file-attach step of a create / update on its own (tests of the insert error mapping).
func (s *Documents) AttachFilesForTest(ctx context.Context, userID, docID uint64, ids []uint64, now time.Time) error {
	return insertFiles(ctx, s.q, userID, docID, ids, now)
}
