<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

final readonly class PostalAddressData
{
    public function __construct(
        public string $streetAddress,
        public string $addressLocality,
        public ?string $addressRegion = null,
        public ?string $postalCode = null,
        public string $addressCountry = 'IR',
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toNode(): array
    {
        return [
            '@type' => 'PostalAddress',
            'streetAddress' => $this->streetAddress,
            'addressLocality' => $this->addressLocality,
            'addressRegion' => $this->addressRegion,
            'postalCode' => $this->postalCode,
            'addressCountry' => $this->addressCountry,
        ];
    }
}
