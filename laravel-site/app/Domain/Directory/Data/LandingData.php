<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\Landing;

/**
 * Copy of a directory landing page. From the repository fields may be empty; LandingCopy::resolve() fills them with
 * the default templates.
 */
final readonly class LandingData
{
    public function __construct(
        public int $cityId,
        public ?int $categoryId,
        public ?string $h1,
        public ?string $title,
        public ?string $description,
        public ?string $intro,
    ) {}

    public static function fromModel(Landing $landing): self
    {
        return new self($landing->city_id, $landing->category_id, $landing->h1, $landing->meta_title, $landing->meta_description, $landing->intro);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'cityId' => $this->cityId, 'categoryId' => $this->categoryId, 'h1' => $this->h1, 'title' => $this->title,
            'description' => $this->description, 'intro' => $this->intro,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $string = static fn (string $key): ?string => isset($data[$key]) ? (string) $data[$key] : null;

        return new self(
            cityId: (int) $data['cityId'],
            categoryId: isset($data['categoryId']) ? (int) $data['categoryId'] : null,
            h1: $string('h1'),
            title: $string('title'),
            description: $string('description'),
            intro: $string('intro'),
        );
    }
}
