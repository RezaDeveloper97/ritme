<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

use App\Domain\Seo\Schema\Enums\ItemAvailability;

/**
 * A price for a product or app. Prices are in Iranian rial (ISO 4217 `IRR`) — convert toman ×10 before building.
 */
final readonly class OfferData
{
    public function __construct(
        public int|float $price,
        public string $priceCurrency = 'IRR',
        public ItemAvailability $availability = ItemAvailability::InStock,
        public ?string $url = null,
        public ?string $priceValidUntil = null,
        public bool $newCondition = true,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toNode(): array
    {
        return [
            '@type' => 'Offer',
            'price' => $this->price,
            'priceCurrency' => $this->priceCurrency,
            'availability' => $this->availability->uri(),
            'url' => $this->url,
            'priceValidUntil' => $this->priceValidUntil,
            'itemCondition' => $this->newCondition ? 'https://schema.org/NewCondition' : 'https://schema.org/UsedCondition',
        ];
    }
}
