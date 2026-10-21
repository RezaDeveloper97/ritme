package content_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// countingRows counts ListLiveMessageContents calls.
type countingRows struct {
	q *store.Queries
	n atomic.Int64
}

func (c *countingRows) ListLiveMessageContents(ctx context.Context, locale string) ([]store.ListLiveMessageContentsRow, error) {
	c.n.Add(1)
	return c.q.ListLiveMessageContents(ctx, locale)
}

func title(t *testing.T, p content.Payload) any {
	t.Helper()
	v, _ := p.Field("title")
	return v
}

// Message contents are read once per locale per process; an admin edit that publishes message_contents after its
// commit is visible on the very next request (a new Repository).
func TestRepository_ProcessSnapshotAndAdminEdit(t *testing.T) {
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })
	db := testdb.New(t)
	configcache.Default.Invalidate(content.Table)
	ctx := context.Background()
	_, err := db.Exec("INSERT INTO message_contents (`group`, item_key, locale, payload, created_at, updated_at) VALUES ('t13', 'k', 'fa', '{\"title\":\"one\"}', NOW(), NOW())")
	require.NoError(t, err)
	src := &countingRows{q: store.New(db)}

	for range 3 { // three requests
		p, err := content.NewRepository(src).Resolve(ctx, "t13", "k", "fa")
		require.NoError(t, err)
		assert.Equal(t, "one", title(t, p))
	}
	_, _ = content.NewRepository(src).Resolve(ctx, "t13", "k", "en")
	assert.Equal(t, int64(2), src.n.Load(), "one load per locale for the whole process")

	update := func(v string) {
		_, err := db.Exec("UPDATE message_contents SET payload = ? WHERE `group` = 't13'", `{"title":"`+v+`"}`)
		require.NoError(t, err)
	}
	update("two")
	configcache.Changed(ctx, content.Table) // what every admin writer does after its commit
	p, err := content.NewRepository(src).Resolve(ctx, "t13", "k", "fa")
	require.NoError(t, err)
	assert.Equal(t, "two", title(t, p), "next request of the same process")

	// Off (the go test default): every request reads the table, as before T-M9-13.
	configcache.SetEnabled(false)
	update("three")
	p, _ = content.NewRepository(src).Resolve(ctx, "t13", "k", "fa")
	assert.Equal(t, "three", title(t, p))
}
