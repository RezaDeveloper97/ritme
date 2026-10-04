<?php

declare(strict_types=1);

use App\Domain\Directory\Actions\ModeratePlaceReviews;
use App\Domain\Directory\Actions\SubmitPlaceReview;
use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\ReviewSubmission;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceReview;
use Illuminate\Database\Eloquent\ModelNotFoundException;

it('aggregates only approved, non-demo reviews into the place rating', function (): void {
    $place = Place::factory()->published()->create();
    PlaceReview::factory()->approved()->create(['place_id' => $place->id, 'rating' => 5]);
    PlaceReview::factory()->approved()->create(['place_id' => $place->id, 'rating' => 4]);
    PlaceReview::factory()->approved()->create(['place_id' => $place->id, 'rating' => 3]);
    PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 1]);                 // pending
    PlaceReview::factory()->rejected()->create(['place_id' => $place->id, 'rating' => 1]);
    PlaceReview::factory()->approved()->demo()->create(['place_id' => $place->id, 'rating' => 1]);

    $place->refresh();
    expect($place->rating_count)->toBe(3)
        ->and($place->rating_avg)->toBe(4.0);

    $data = app(PlaceRepository::class)->findPublishedBySlug($place->slug);
    expect($data?->rating()?->ratingCount)->toBe(3)
        ->and($data?->rating()?->ratingValue)->toBe(4.0);
});

it('recalculates on moderation, edits and deletion', function (): void {
    $place = Place::factory()->published()->create();
    $a = PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 5]);
    $b = PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 2]);
    expect($place->refresh()->rating_count)->toBe(0);

    expect(app(ModeratePlaceReviews::class)->handle([$a->id, $b->id], ReviewStatus::Approved))->toBe(2)
        ->and($place->refresh()->rating_count)->toBe(2)
        ->and($place->rating_avg)->toBe(3.5)
        ->and($a->refresh()->approved_at)->not->toBeNull();

    $b->refresh()->update(['status' => ReviewStatus::Rejected]);
    expect($place->refresh()->rating_avg)->toBe(5.0)->and($b->refresh()->approved_at)->toBeNull();

    $a->delete();
    expect($place->refresh()->rating_count)->toBe(0)->and($place->rating_avg)->toBe(0.0)
        ->and(app(ModeratePlaceReviews::class)->handle([$b->id], ReviewStatus::Rejected))->toBe(0);
});

it('never emits a rating for a place with only demo reviews', function (): void {
    $place = Place::factory()->published()->demo()->create();
    PlaceReview::factory()->approved()->demo()->create(['place_id' => $place->id, 'rating' => 5]);
    $repo = app(PlaceRepository::class);

    $data = $repo->findPublishedBySlug($place->slug);
    expect($data?->ratingCount)->toBe(0)
        ->and($data?->rating())->toBeNull()
        ->and($data?->toLocalBusiness('https://ritme.ir/x')->rating)->toBeNull()
        ->and($repo->ratingSummary($place->id)->isEmpty())->toBeTrue()
        ->and($repo->reviews($place->id)->items[0]->isDemo)->toBeTrue();
});

it('summarises per-aspect averages and lists approved reviews newest first', function (): void {
    $place = Place::factory()->published()->create();
    PlaceReview::factory()->approved()->create(['place_id' => $place->id, 'rating' => 5, 'aspects' => ['cleanliness' => 5, 'staff' => 4], 'approved_at' => now()->subDays(3)]);
    PlaceReview::factory()->approved()->create(['place_id' => $place->id, 'rating' => 4, 'aspects' => ['cleanliness' => 4, 'bogus' => 9], 'approved_at' => now()->subDay()]);
    PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 1]);
    $repo = app(PlaceRepository::class);

    $summary = $repo->ratingSummary($place->id);
    expect($summary->average)->toBe(4.5)
        ->and($summary->count)->toBe(2)
        ->and($summary->aspects)->toBe(['cleanliness' => 4.5, 'staff' => 4.0])
        ->and($summary->toSchema()?->ratingCount)->toBe(2);

    $reviews = $repo->reviews($place->id, 1, 1);
    expect($reviews->total)->toBe(2)
        ->and($reviews->lastPage())->toBe(2)
        ->and($reviews->items[0]->rating)->toBe(4);
});

it('stores submitted reviews as pending, trimmed, with a hashed IP', function (): void {
    $place = Place::factory()->published()->create(['slug' => 'ab-pari']);

    $review = app(SubmitPlaceReview::class)->handle(new ReviewSubmission(
        placeSlug: 'ab-pari', authorName: '  مادر   رها ', rating: 5, body: ' <b>عالی</b> بود ',
        aspects: ['staff' => 5, 'cleanliness' => 9, 'bogus' => 3], ip: '203.0.113.7',
    ));

    expect($review->status)->toBe(ReviewStatus::Pending)
        ->and($review->author_name)->toBe('مادر رها')
        ->and($review->body)->toBe('عالی بود')
        ->and($review->aspects)->toBe(['staff' => 5])
        ->and($review->ip_hash)->toHaveLength(64)
        ->and($review->ip_hash)->not->toContain('203.0.113.7')
        ->and($place->refresh()->rating_count)->toBe(0)
        ->and(app(PlaceRepository::class)->reviews($place->id)->total)->toBe(0);
});

it('rejects reviews of unpublished places and invalid ratings', function (): void {
    Place::factory()->create(['slug' => 'draft']);
    Place::factory()->published()->create(['slug' => 'live']);
    $submit = app(SubmitPlaceReview::class);

    expect(fn () => $submit->handle(new ReviewSubmission('draft', 'نام', 5, 'متن')))->toThrow(ModelNotFoundException::class)
        ->and(fn () => $submit->handle(new ReviewSubmission('live', 'نام', 6, 'متن')))->toThrow(InvalidArgumentException::class)
        ->and(fn () => $submit->handle(new ReviewSubmission('live', ' ', 5, 'متن')))->toThrow(InvalidArgumentException::class);
});
