<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\LocalBusinessData;
use App\Domain\Seo\Schema\Data\OpeningHoursData;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * A directory place (`#place`): LocalBusiness or a subtype (ChildCare, SportsActivityLocation, MedicalClinic …) with
 * PostalAddress, geo and OpeningHoursSpecification.
 */
final class LocalBusinessNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(LocalBusinessData $place): array
    {
        return Node::clean([
            '@type' => $place->type->value,
            '@id' => SchemaIds::place($place->url),
            'name' => $place->name,
            'description' => $place->description,
            'url' => $place->url,
            'telephone' => $place->telephone,
            'image' => Node::strings($place->imageUrls),
            'address' => $place->address->toNode(),
            'geo' => $place->latitude !== null && $place->longitude !== null ? [
                '@type' => 'GeoCoordinates',
                'latitude' => round($place->latitude, 6),
                'longitude' => round($place->longitude, 6),
            ] : null,
            'openingHoursSpecification' => array_map(
                static fn (OpeningHoursData $hours): array => $hours->toNode(),
                $place->openingHours,
            ),
            'priceRange' => $place->priceRange,
            'sameAs' => Node::strings($place->sameAs),
            'aggregateRating' => $place->rating?->toNode(),
            'mainEntityOfPage' => Node::ref(SchemaIds::webPage($place->url)),
        ]);
    }
}
