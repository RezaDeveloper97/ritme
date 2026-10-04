<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\PlaceService;
use App\Domain\Directory\Support\AgeRange;

/**
 * A service row of the place page («آشنایی با آب · ۴۵ دقیقه · ۶ تا ۱۸ ماه · گروهی — ۳۲۰ هزار تومان»).
 * `price` is whole Toman as announced by the place (null = ask the place).
 */
final readonly class PlaceServiceData
{
    public function __construct(
        public int $id,
        public string $name,
        public ?int $durationMinutes = null,
        public ?int $price = null,
        public ?string $priceUnit = null,
        public ?string $details = null,
        public ?int $ageMinMonths = null,
        public ?int $ageMaxMonths = null,
    ) {}

    public static function fromModel(PlaceService $service): self
    {
        return new self(
            id: $service->id,
            name: $service->name,
            durationMinutes: $service->duration_minutes,
            price: $service->price,
            priceUnit: $service->price_unit,
            details: $service->details,
            ageMinMonths: $service->age_min_months,
            ageMaxMonths: $service->age_max_months,
        );
    }

    public function ageRange(): AgeRange
    {
        return new AgeRange($this->ageMinMonths, $this->ageMaxMonths);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'name' => $this->name, 'durationMinutes' => $this->durationMinutes, 'price' => $this->price,
            'priceUnit' => $this->priceUnit, 'details' => $this->details,
            'ageMinMonths' => $this->ageMinMonths, 'ageMaxMonths' => $this->ageMaxMonths,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $int = static fn (string $key): ?int => isset($data[$key]) ? (int) $data[$key] : null;

        return new self(
            id: (int) $data['id'],
            name: (string) $data['name'],
            durationMinutes: $int('durationMinutes'),
            price: $int('price'),
            priceUnit: isset($data['priceUnit']) ? (string) $data['priceUnit'] : null,
            details: isset($data['details']) ? (string) $data['details'] : null,
            ageMinMonths: $int('ageMinMonths'),
            ageMaxMonths: $int('ageMaxMonths'),
        );
    }
}
