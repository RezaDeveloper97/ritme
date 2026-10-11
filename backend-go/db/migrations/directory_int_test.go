package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const directoryVersion int64 = 81 // 00081_directory.sql

// CB-DIR-01: Up extends bloom's city-service tables without breaking a legacy (B-N10-05) listing + booking, adds the
// directory tables and seeds; a review is unique per booking; deleting a listing cascades its services / slots;
// Down removes exactly that and Up restores it.
func TestDirectoryMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	count := func(q string, args ...any) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, q, args...).Scan(&n))
		return n
	}
	exec := func(q string, args ...any) int64 {
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	tables := func() int {
		return count(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN
			('city_service_services', 'city_service_slots', 'city_service_reviews', 'city_service_reports', 'city_service_applications')`)
	}
	cols := func() int {
		return count(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
			AND ((table_name = 'city_service_listings' AND column_name IN ('status', 'lat', 'weekly_hours', 'rating_count'))
			  OR (table_name = 'city_service_bookings' AND column_name IN ('code', 'slot_id', 'child_id')))`)
	}
	seed := func() int {
		return count("SELECT COUNT(*) FROM catalog_items WHERE `group` IN ('dir_amenities', 'dir_age_ranges')") +
			count("SELECT COUNT(*) FROM city_service_categories WHERE slug IN ('music_art', 'postpartum_fitness', 'baby_massage', 'parenting_workshop', 'other')")
	}
	require.Equal(t, 5, tables())
	require.Equal(t, 7, cols())
	require.Equal(t, 7+6+5, seed())

	user := exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mother', '09129990081', NOW(), NOW())`)
	city := exec("INSERT INTO city_service_cities (slug, name) VALUES ('tehran-dir', '{\"fa\":\"تهران\"}')")
	var cat int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM city_service_categories WHERE slug = 'pool'`).Scan(&cat))

	// A bloom-shaped insert (no directory columns) still works and lands published.
	legacy := exec(`INSERT INTO city_service_listings (category_id, city_id, name, booking_mode) VALUES (?, ?, 'Legacy', 'request')`, cat, city)
	var status string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT status FROM city_service_listings WHERE id = ?`, legacy).Scan(&status))
	assert.Equal(t, "published", status)
	exec(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date)
		VALUES (?, ?, 'M', '09120000000', '2026-10-20')`, user, legacy)

	place := exec(`INSERT INTO city_service_listings (category_id, city_id, name, booking_mode, lat, lng, amenities)
		VALUES (?, ?, 'Ab-Pari', 'online', 35.757, 51.410, '["nursing_room"]')`, cat, city)
	svc := exec(`INSERT INTO city_service_services (listing_id, name, duration_minutes, age_min_months, age_max_months, price_toman)
		VALUES (?, 'Water intro', 45, 6, 18, 320000)`, place)
	slot := exec(`INSERT INTO city_service_slots (listing_id, service_id, starts_at, capacity) VALUES (?, ?, '2026-10-08 17:00:00', 6)`, place, svc)
	booking := exec(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, code, service_id, slot_id, status)
		VALUES (?, ?, 'M', '09120000000', '2026-10-08', 'D73104', ?, ?, 'completed')`, user, place, svc, slot)
	exec(`INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating, score_cleanliness) VALUES (?, ?, ?, 5, 5)`, place, booking, user)
	_, err = db.ExecContext(ctx, `INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating) VALUES (?, ?, ?, 4)`, place, booking, user)
	require.Error(t, err, "one review per booking")
	_, err = db.ExecContext(ctx, `INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, code)
		VALUES (?, ?, 'M', '09120000000', '2026-10-08', 'D73104')`, user, place)
	require.Error(t, err, "booking codes are unique")
	exec(`INSERT INTO city_service_applications (user_id, business_name, documents) VALUES (?, 'Ab-Pari', '{"licence":1}')`, user)

	// Deleting a slot keeps the booking (slot_id → NULL); deleting the place cascades services / slots / reviews.
	_, err = db.ExecContext(ctx, `DELETE FROM city_service_slots WHERE id = ?`, slot)
	require.NoError(t, err)
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM city_service_bookings WHERE id = ? AND slot_id IS NULL`, booking))
	_, err = db.ExecContext(ctx, `DELETE FROM city_service_bookings WHERE listing_id = ?`, place)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM city_service_listings WHERE id = ?`, place)
	require.NoError(t, err)
	assert.Zero(t, count(`SELECT COUNT(*) FROM city_service_services`))
	assert.Zero(t, count(`SELECT COUNT(*) FROM city_service_reviews`))

	_, err = p.DownTo(ctx, directoryVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())
	assert.Zero(t, cols())
	assert.Zero(t, seed())
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM city_service_bookings WHERE listing_id = ?`, legacy), "bloom rows survive Down")
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, tables())
	assert.Equal(t, 7, cols())
	assert.Equal(t, 7+6+5, seed())
}
