<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\AgeRange;
use App\Domain\Directory\Support\MapLinks;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Seo\Schema\Data\AggregateRatingData;
use App\Domain\Seo\Schema\Data\LocalBusinessData;
use App\Domain\Seo\Schema\Data\PostalAddressData;
use App\Support\Text\PersianDigits;
use Carbon\CarbonImmutable;

/**
 * A published place for its page (L5-03). Media are ids for <x-picture>: `coverMediaId` then the ordered gallery.
 * `ratingAvg`/`ratingCount` come from real approved reviews only; `isDemo` marks placeholder businesses; `phoneOnly`
 * places (BookingMode::Phone) take no booking requests — the page shows their number instead of the form.
 */
final readonly class PlaceData
{
    /**
     * @param  list<string>  $phones
     * @param  array<string, mixed>  $openingHours
     * @param  list<int>  $galleryMediaIds
     * @param  list<AmenityData>  $amenities
     * @param  list<PlaceServiceData>  $services
     */
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public CategoryData $category,
        public CityData $city,
        public ?DistrictData $district,
        public ?string $summary,
        public ?string $description,
        public ?string $address,
        public ?string $postalCode,
        public ?float $latitude,
        public ?float $longitude,
        public array $phones,
        public ?string $website,
        public ?int $ageMinMonths,
        public ?int $ageMaxMonths,
        public array $openingHours,
        public ?string $rules,
        public ?string $cancellationPolicy,
        public ?int $coverMediaId,
        public array $galleryMediaIds,
        public array $amenities,
        public array $services,
        public bool $isVerified,
        public bool $isDemo,
        public float $ratingAvg,
        public int $ratingCount,
        public ?int $priceFrom,
        public CarbonImmutable $updatedAt,
        public bool $phoneOnly = false,
    ) {}

    /**
     * Expects `category`, `city`, `district`, `amenities`, `services` and `gallery` loaded.
     */
    public static function fromModel(Place $place): self
    {
        return new self(
            id: $place->id,
            name: $place->name,
            slug: $place->slug,
            category: CategoryData::fromModel($place->category),
            city: CityData::fromModel($place->city),
            district: $place->district === null ? null : DistrictData::fromModel($place->district),
            summary: $place->summary,
            description: $place->description,
            address: $place->address,
            postalCode: $place->postal_code,
            latitude: $place->latitude,
            longitude: $place->longitude,
            phones: $place->phones ?? [],
            website: $place->website,
            ageMinMonths: $place->age_min_months,
            ageMaxMonths: $place->age_max_months,
            openingHours: $place->openingHours()->toArray(),
            rules: $place->rules,
            cancellationPolicy: $place->cancellation_policy,
            coverMediaId: $place->cover_media_id,
            galleryMediaIds: $place->gallery->map(static fn ($media): int => (int) $media->getKey())->values()->all(),
            amenities: $place->amenities->sortBy('sort_order')->map(AmenityData::fromModel(...))->values()->all(),
            services: $place->services->map(PlaceServiceData::fromModel(...))->values()->all(),
            isVerified: $place->is_verified,
            isDemo: $place->is_demo,
            ratingAvg: $place->rating_avg,
            ratingCount: $place->rating_count,
            priceFrom: $place->price_from,
            updatedAt: ($place->updated_at ?? $place->created_at ?? now())->toImmutable(),
            phoneOnly: $place->booking_mode === BookingMode::Phone,
        );
    }

    public function ageRange(): AgeRange
    {
        return new AgeRange($this->ageMinMonths, $this->ageMaxMonths);
    }

    public function openingHours(): OpeningHours
    {
        return OpeningHours::fromArray($this->openingHours);
    }

    /**
     * Cover first, then the gallery (no duplicates) — the order of the page gallery and of schema `image`.
     *
     * @return list<int>
     */
    public function imageMediaIds(): array
    {
        $ids = $this->coverMediaId === null ? $this->galleryMediaIds : [$this->coverMediaId, ...$this->galleryMediaIds];

        return array_values(array_unique($ids));
    }

    /**
     * The rules text as list items (one per non-empty line).
     *
     * @return list<string>
     */
    public function ruleLines(): array
    {
        $lines = preg_split('/\R/u', (string) $this->rules) ?: [];

        return array_values(array_filter(array_map(static fn (string $line): string => trim($line, " \t-•"), $lines), static fn (string $line): bool => $line !== ''));
    }

    public function mapLinks(): ?MapLinks
    {
        return $this->latitude === null || $this->longitude === null ? null : MapLinks::at($this->latitude, $this->longitude, $this->name);
    }

    /** Aggregate of real reviews, or null when there are none. */
    public function rating(): ?AggregateRatingData
    {
        return $this->ratingCount < 1 ? null : new AggregateRatingData($this->ratingAvg, $this->ratingCount, $this->ratingCount);
    }

    /**
     * schema.org priceRange from the announced service prices, e.g. «۳۲۰٬۰۰۰ تا ۲٬۴۰۰٬۰۰۰ تومان».
     */
    public function priceRange(): ?string
    {
        $prices = array_values(array_filter(array_map(static fn (PlaceServiceData $s): ?int => $s->price, $this->services), static fn (?int $p): bool => $p !== null && $p > 0));
        if ($prices === []) {
            return null;
        }

        $min = min($prices);
        $max = max($prices);

        return $min === $max
            ? PersianDigits::number($min).' تومان'
            : PersianDigits::number($min).' تا '.PersianDigits::number($max).' تومان';
    }

    /**
     * LocalBusiness node data (subtype from the category). `$imageUrls` are absolute URLs of imageMediaIds().
     *
     * @param  list<string>  $imageUrls
     */
    public function toLocalBusiness(string $url, array $imageUrls = []): LocalBusinessData
    {
        return new LocalBusinessData(
            url: $url,
            name: $this->name,
            address: new PostalAddressData(
                streetAddress: implode('، ', array_filter([trim((string) $this->address), $this->district->name ?? ''], static fn (string $part): bool => $part !== '')),
                addressLocality: $this->city->name,
                addressRegion: $this->city->province,
                postalCode: $this->postalCode,
            ),
            type: $this->category->schemaType,
            description: $this->summary ?? $this->description,
            telephone: $this->phones[0] ?? null,
            imageUrls: $imageUrls,
            latitude: $this->latitude,
            longitude: $this->longitude,
            openingHours: $this->openingHours()->toSchema(),
            priceRange: $this->priceRange(),
            sameAs: $this->website === null ? [] : [$this->website],
            rating: $this->rating(),
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'name' => $this->name, 'slug' => $this->slug,
            'category' => $this->category->toArray(), 'city' => $this->city->toArray(), 'district' => $this->district?->toArray(),
            'summary' => $this->summary, 'description' => $this->description, 'address' => $this->address,
            'postalCode' => $this->postalCode, 'latitude' => $this->latitude, 'longitude' => $this->longitude,
            'phones' => $this->phones, 'website' => $this->website,
            'ageMinMonths' => $this->ageMinMonths, 'ageMaxMonths' => $this->ageMaxMonths,
            'openingHours' => $this->openingHours, 'rules' => $this->rules, 'cancellationPolicy' => $this->cancellationPolicy,
            'coverMediaId' => $this->coverMediaId, 'galleryMediaIds' => $this->galleryMediaIds,
            'amenities' => array_map(static fn (AmenityData $a): array => $a->toArray(), $this->amenities),
            'services' => array_map(static fn (PlaceServiceData $s): array => $s->toArray(), $this->services),
            'isVerified' => $this->isVerified, 'isDemo' => $this->isDemo,
            'ratingAvg' => $this->ratingAvg, 'ratingCount' => $this->ratingCount, 'priceFrom' => $this->priceFrom,
            'updatedAt' => $this->updatedAt->toIso8601String(),
            'phoneOnly' => $this->phoneOnly,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $int = static fn (string $key): ?int => isset($data[$key]) ? (int) $data[$key] : null;
        $float = static fn (string $key): ?float => isset($data[$key]) ? (float) $data[$key] : null;
        $string = static fn (string $key): ?string => isset($data[$key]) ? (string) $data[$key] : null;
        /** @var array<string, mixed> $category */
        $category = $data['category'];
        /** @var array<string, mixed> $city */
        $city = $data['city'];
        /** @var array<string, mixed>|null $district */
        $district = $data['district'] ?? null;
        /** @var list<array<string, mixed>> $amenities */
        $amenities = (array) ($data['amenities'] ?? []);
        /** @var list<array<string, mixed>> $services */
        $services = (array) ($data['services'] ?? []);

        return new self(
            id: (int) $data['id'],
            name: (string) $data['name'],
            slug: (string) $data['slug'],
            category: CategoryData::fromArray($category),
            city: CityData::fromArray($city),
            district: $district === null ? null : DistrictData::fromArray($district),
            summary: $string('summary'),
            description: $string('description'),
            address: $string('address'),
            postalCode: $string('postalCode'),
            latitude: $float('latitude'),
            longitude: $float('longitude'),
            phones: array_values(array_map(strval(...), (array) ($data['phones'] ?? []))),
            website: $string('website'),
            ageMinMonths: $int('ageMinMonths'),
            ageMaxMonths: $int('ageMaxMonths'),
            openingHours: (array) ($data['openingHours'] ?? []),
            rules: $string('rules'),
            cancellationPolicy: $string('cancellationPolicy'),
            coverMediaId: $int('coverMediaId'),
            galleryMediaIds: array_values(array_map(intval(...), (array) ($data['galleryMediaIds'] ?? []))),
            amenities: array_map(AmenityData::fromArray(...), $amenities),
            services: array_map(PlaceServiceData::fromArray(...), $services),
            isVerified: (bool) $data['isVerified'],
            isDemo: (bool) ($data['isDemo'] ?? false),
            ratingAvg: (float) $data['ratingAvg'],
            ratingCount: (int) $data['ratingCount'],
            priceFrom: $int('priceFrom'),
            updatedAt: CarbonImmutable::parse((string) $data['updatedAt']),
            phoneOnly: (bool) ($data['phoneOnly'] ?? false),
        );
    }
}
