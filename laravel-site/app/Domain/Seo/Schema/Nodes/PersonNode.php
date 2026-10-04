<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * An author / reviewer, embedded where it is used (with a stable `@id` so engines merge it across pages).
 */
final class PersonNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(PersonData $person, string $siteUrl): array
    {
        return Node::clean([
            '@type' => 'Person',
            '@id' => SchemaIds::person($siteUrl, $person->url ?? $person->name),
            'name' => $person->name,
            'url' => $person->url,
            'jobTitle' => $person->jobTitle,
            'description' => $person->description,
            'image' => $person->imageUrl,
            'sameAs' => Node::strings($person->sameAs),
            'knowsAbout' => Node::strings($person->knowsAbout),
            'hasCredential' => $person->credential !== null ? [
                '@type' => 'EducationalOccupationalCredential',
                'credentialCategory' => $person->credential,
            ] : null,
        ]);
    }
}
