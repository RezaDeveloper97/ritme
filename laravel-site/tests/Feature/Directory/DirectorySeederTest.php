<?php

declare(strict_types=1);

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use Database\Seeders\DirectorySeeder;

it('seeds the design places idempotently, all marked as demo', function (): void {
    $this->seed(DirectorySeeder::class);
    $this->seed(DirectorySeeder::class);

    $place = app(PlaceRepository::class)->findPublishedBySlug(DirectorySeeder::DEMO_PLACE_SLUG);
    $taxonomy = app(TaxonomyRepository::class);

    expect(Place::query()->count())->toBe(6)
        ->and(Place::query()->where('is_demo', false)->count())->toBe(0)
        ->and(PlaceReview::query()->where('is_demo', false)->count())->toBe(0)
        ->and(count($taxonomy->categories()))->toBe(8)
        ->and($taxonomy->findCity('tehran')?->districts)->toHaveCount(5)
        ->and($place?->name)->toBe('استخر مادر و کودک آب‌پری')
        ->and($place?->isDemo)->toBeTrue()
        ->and($place?->category->schemaType)->toBe(LocalBusinessType::SportsActivityLocation)
        ->and($place?->services)->toHaveCount(4)
        ->and($place?->priceFrom)->toBe(320_000)
        ->and($place?->ageRange()->label())->toBe('۶ ماه تا ۴ سال')
        ->and(array_column($place?->openingHours()->rows() ?? [], 'hours'))->toBe(['۹ تا ۲۰', '۹ تا ۲۰', '۹ تا ۲۰', '۹ تا ۲۰', '۹ تا ۲۰', '۹ تا ۱۴', 'تعطیل']);
});

it('never seeds a rating: demo reviews are shown labelled but not counted', function (): void {
    $this->seed(DirectorySeeder::class);
    $repo = app(PlaceRepository::class);
    $place = $repo->findPublishedBySlug(DirectorySeeder::DEMO_PLACE_SLUG);

    expect(Place::query()->where('rating_count', '>', 0)->count())->toBe(0)
        ->and($place?->rating())->toBeNull()
        ->and($repo->reviews($place->id ?? 0)->total)->toBe(2)
        ->and(collect($repo->reviews($place->id ?? 0)->items)->every(static fn ($r): bool => $r->isDemo))->toBeTrue()
        ->and($repo->ratingSummary($place->id ?? 0)->isEmpty())->toBeTrue();
});

it('keeps demo copy inside the content red lines', function (): void {
    $this->seed(DirectorySeeder::class);

    $text = Place::query()->get()->map(static fn (Place $p): string => implode(' ', [$p->name, $p->summary, $p->description, $p->rules, $p->cancellation_policy]))->implode(' ')
        .' '.PlaceReview::query()->pluck('body')->implode(' ');

    foreach (['حتماً', 'حتما', 'قطعاً', 'قطعا', 'دقیق‌ترین', 'دقیقترین', 'تضمینی', 'تشخیص', 'بهترین'] as $word) {
        expect($text)->not->toContain($word);
    }
});
