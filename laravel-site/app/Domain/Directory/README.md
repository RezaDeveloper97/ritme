# Directory

Mother & child directory («خدمات مادر و کودک», L5): places, categories (schema.org LocalBusiness subtype), cities +
districts, amenities, services & prices, moderated reviews, landing copy per city / city × category (L5-01).

- Read side: `Contracts\PlaceRepository` (place page, slug history, `search(PlaceSearchCriteria)`, reviews, rating
  summary) and `Contracts\TaxonomyRepository` (cities, categories, amenities, landing copy, combos) → `Cached*`
  (`directory` ns, arrays) → `Eloquent*` → `Queries\SearchPlaces` (filters: city, district, category, child age,
  amenities AND, text, open-now; sorts: recommended, nearest, rating, price, newest — no paid placement),
  `Queries\LandingCombos`, `Queries\SitemapPlaces`. DTOs in `Data/`; landing copy falls back to `Support\LandingCopy`.
- Opening hours: `Support\OpeningHours` (JSON per weekday, ranges incl. past-midnight; `isOpenAt`, `closesAt`,
  `nextOpening`, Persian table `rows()`, `toSchema()`), `Enums\Weekday` (Persian names). Ages: `Support\AgeRange`.
  Maps: `Support\MapLinks` (geo:/Neshan/Balad/Google deep links — never an embedded map).
- Ratings: only approved, NON-demo reviews count (`PlaceReview::counted()`); `RecalculatePlaceRating` keeps
  `rating_avg`/`rating_count` on the place, `PlaceData::toLocalBusiness()` emits `aggregateRating` only when count ≥ 1.
- Write side: `PlaceObserver` (slug + history, hours/phones normalised), `PlaceServiceObserver` (`price_from`),
  `PlaceReviewObserver` (rating), `TaxonomyObserver`, `LandingObserver`; all bump `directory` (+ `sitemap` where URLs
  change) and `pages`. Pivots: `SyncPlaceAmenities`, `SyncPlaceGallery`. Reviews: `SubmitPlaceReview` (pending),
  `ModeratePlaceReviews`.
- Admin (L5-06, `app/Filament/Resources/Directory`): `SavePlace` (attributes + amenities + gallery, never status),
  `ChangePlaceStatus`, `Booking\Actions\ChangeBookingStatus` (new → confirmed|cancelled, confirmed → done|cancelled),
  `Booking\Actions\ExportBookingRequests` (CSV, masked mobile), `Join\Actions\ApproveJoinRequest` (→ draft place with
  data + photos, contact person not copied) / `RejectJoinRequest`. Every status change → activity log `directory`
  (`DirectoryActivity`). `directory_places.booking_mode` (`Join\Enums\BookingMode`): `phone` hides the booking form.
- Sitemaps: `Sitemap\PlaceSitemapProvider` (`directory-places`), `Sitemap\LandingSitemapProvider`
  (`directory-landings`, combos with ≥ 1 non-demo place). Search: `Search\PlaceSearchProvider` (`places`).
- Demo content: `php artisan db:seed --class=DirectorySeeder` (not in DatabaseSeeder; every place/review `is_demo`).
