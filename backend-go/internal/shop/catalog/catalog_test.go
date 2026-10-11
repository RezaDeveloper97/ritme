package catalog

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/shop/store"
)

var fa = Locale{Code: "fa", Default: "fa"}
var en = Locale{Code: "en", Default: "fa"}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func variant(id uint64, size, color string, price uint64, stock uint32) store.ShopProductVariant {
	return store.ShopProductVariant{ID: id, Size: ns(size), Color: ns(color), Price: price, Stock: stock, IsActive: true}
}

func item(id uint64, title string, sold uint32, rating string, attrs, skins string, vs ...store.ShopProductVariant) *Item {
	return &Item{Product: store.ShopProduct{ID: id, Title: title, PurchaseCount: sold, RatingAvg: ns(rating),
		RatingCount: map[bool]uint32{true: 10}[rating != ""], Attributes: ns(attrs), SkinTypes: ns(skins),
		CreatedAt: sql.NullTime{Time: time.Date(2026, 10, int(id), 9, 0, 0, 0, time.UTC), Valid: true}}, Variants: vs}
}

const (
	m03    = `{"fa":"۰ تا ۳ ماه","en":"0-3 m"}`
	m36    = `{"fa":"۳ تا ۶ ماه","en":"3-6 m"}`
	cream  = `{"fa":"شیری","en":"Cream"}`
	cotton = `{"material":{"fa":"پنبه ۱۰۰٪","en":"100% cotton"},"wash":{"fa":"۳۰ درجه","en":"30°C"}}`
)

func fixture() []*Item {
	return []*Item{
		item(1, `{"fa":"بادی نخی","en":"Cotton bodysuit"}`, 1800, "4.70", cotton, "",
			variant(11, m03, cream, 485000, 0), variant(12, m36, cream, 495000, 3)),
		item(2, `{"fa":"سرهمی خواب","en":"Sleepsuit"}`, 90, "4.80", "", "",
			variant(21, m36, "", 390000, 5)),
		item(3, `{"fa":"ضدآفتاب","en":"Sunscreen"}`, 400, "", "", `["oily","combination"]`,
			variant(31, "", "", 550000, 2)),
	}
}

func ids(items []*Item) []uint64 {
	out := []uint64{}
	for _, it := range items {
		out = append(out, it.Product.ID)
	}
	return out
}

func TestLocale_Pick(t *testing.T) {
	assert.Equal(t, "3-6 m", en.Pick(m36))
	assert.Equal(t, "۳ تا ۶ ماه", en.Value(m36), "filter values are the default-language text")
	assert.Equal(t, "plain", en.Pick(`"plain"`))
	assert.Empty(t, en.Pick(""))
	assert.Empty(t, en.Pick("null"))
}

func TestItem_CardPrice(t *testing.T) {
	it := fixture()[0]
	assert.EqualValues(t, 495000, it.Price(), "the cheapest IN-STOCK variant wins over a cheaper sold-out one")
	assert.True(t, it.InStock())
	assert.InDelta(t, 4.7, it.Rating(), 0.0001)
	assert.Nil(t, fixture()[2].Rating(), "no reviews → null")

	sale := item(9, `{"fa":"x"}`, 0, "", "", "", store.ShopProductVariant{ID: 91, Price: 100,
		CompareAtPrice: sql.NullInt64{Int64: 150, Valid: true}})
	assert.EqualValues(t, 150, sale.CompareAt())
	assert.False(t, sale.InStock())
	sale.Variants[0].CompareAtPrice.Int64 = 90
	assert.Nil(t, sale.CompareAt(), "a compare-at below the price is not shown")
}

func TestFilter_ApplyAndSort(t *testing.T) {
	all := fixture()
	f := Filter{Sort: SortPopular}
	assert.Equal(t, []uint64{1, 3, 2}, ids(f.Apply(fa, all)))

	f = Filter{Sizes: []string{"۳ تا ۶ ماه"}}
	assert.Equal(t, []uint64{1, 2}, ids(f.Apply(en, all)), "size matches the default-language value in any UI language")

	f = Filter{Sizes: []string{"۰ تا ۳ ماه"}, PriceMin: 490000}
	assert.Empty(t, f.Apply(fa, all), "size and price must hold on the SAME variant")

	f = Filter{Materials: []string{"پنبه ۱۰۰٪"}}
	assert.Equal(t, []uint64{1}, ids(f.Apply(fa, all)))

	f = Filter{SkinTypes: []string{"dry", "oily"}}
	assert.Equal(t, []uint64{3}, ids(f.Apply(fa, all)))

	f = Filter{Query: "sleep"}
	assert.Equal(t, []uint64{2}, ids(f.Apply(en, all)))
	f = Filter{Query: "سرهمی"}
	assert.Equal(t, []uint64{2}, ids(f.Apply(en, all)), "search also matches the default-language title")

	f = Filter{PriceMax: 400000}
	assert.Equal(t, []uint64{2}, ids(f.Apply(fa, all)))

	assert.Equal(t, []uint64{2, 1, 3}, ids((&Filter{Sort: SortPriceAsc}).Apply(fa, all)))
	assert.Equal(t, []uint64{3, 1, 2}, ids((&Filter{Sort: SortPriceDesc}).Apply(fa, all)))
	assert.Equal(t, []uint64{2, 1, 3}, ids((&Filter{Sort: SortRating}).Apply(fa, all)))
	assert.Equal(t, []uint64{3, 2, 1}, ids((&Filter{Sort: SortNewest}).Apply(fa, all)))
}

func TestBuildFacets(t *testing.T) {
	f := BuildFacets(en, fixture())
	require.Len(t, f.Sizes, 2)
	assert.Equal(t, FacetOption{Value: "۰ تا ۳ ماه", Label: "0-3 m", Count: 1}, f.Sizes[0])
	assert.Equal(t, FacetOption{Value: "۳ تا ۶ ماه", Label: "3-6 m", Count: 2}, f.Sizes[1], "counted per product")
	assert.Equal(t, []FacetOption{{Value: "شیری", Label: "Cream", Count: 1}}, f.Colors)
	assert.Equal(t, []FacetOption{{Value: "پنبه ۱۰۰٪", Label: "100% cotton", Count: 1}}, f.Materials)
	assert.Len(t, f.SkinTypes, 2)
	assert.EqualValues(t, 390000, f.PriceMin)
	assert.EqualValues(t, 550000, f.PriceMax)
	assert.Empty(t, BuildFacets(en, nil).Sizes)
}

func TestAttributesAndSkinTypes(t *testing.T) {
	out, err := json.Marshal(attributesJSON(en, `{"zeta":"z","wash":{"fa":"ش","en":"W"},"material":{"fa":"پ","en":"M"}}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"material":"M","wash":"W","zeta":"z"}`, string(out))
	assert.Equal(t, `{"material":"M","wash":"W","zeta":"z"}`, string(out), "board rows first, then alphabetical")
	assert.Empty(t, SkinTypes("not json"))
	assert.Equal(t, []string{"a", "b"}, SplitList(" a, b ,a,,"))
	assert.Equal(t, []string{"Cotton", "x"}, badges(en, `[{"fa":"پ","en":"Cotton"},"x",""]`))
}

func TestLangFiles(t *testing.T) {
	err := fs.WalkDir(langFS, "lang", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := langFS.ReadFile(path)
		require.NoError(t, err)
		assert.True(t, json.Valid(raw), path)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "The shop is not available in teen mode.", T("errors.teen_commercial_blocked", "en"))
	assert.Equal(t, "فروشگاه در حالت نوجوان در دسترس نیست.", T("errors.teen_commercial_blocked", "fa"))
}
