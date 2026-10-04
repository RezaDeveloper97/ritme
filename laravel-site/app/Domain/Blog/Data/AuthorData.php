<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

use App\Domain\Blog\Models\Author;
use App\Domain\Seo\Schema\Data\PersonData;

/**
 * An author or medical reviewer for views and JSON-LD. Avatar is a media id (resolved by <x-picture>).
 */
final readonly class AuthorData
{
    /**
     * @param  list<string>  $sameAs
     */
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public ?string $jobTitle = null,
        public ?string $credentials = null,
        public ?string $bio = null,
        public ?int $avatarMediaId = null,
        public array $sameAs = [],
        public bool $isMedicalReviewer = false,
    ) {}

    public static function fromModel(Author $author): self
    {
        return new self(
            id: $author->id,
            name: $author->name,
            slug: $author->slug,
            jobTitle: $author->job_title,
            credentials: $author->credentials,
            bio: $author->bio,
            avatarMediaId: $author->avatar_media_id,
            sameAs: array_values(array_filter(array_map(strval(...), $author->same_as ?? []), static fn (string $url): bool => $url !== '')),
            isMedicalReviewer: $author->is_medical_reviewer,
        );
    }

    /**
     * Person node data (author / reviewedBy). The page passes the author page URL and avatar URL it resolved.
     */
    public function toPerson(?string $url = null, ?string $imageUrl = null): PersonData
    {
        return new PersonData(
            name: $this->name,
            url: $url,
            jobTitle: $this->jobTitle,
            imageUrl: $imageUrl,
            sameAs: $this->sameAs,
            credential: $this->credentials,
            description: $this->bio,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return get_object_vars($this);
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: (int) $data['id'],
            name: (string) $data['name'],
            slug: (string) $data['slug'],
            jobTitle: isset($data['jobTitle']) ? (string) $data['jobTitle'] : null,
            credentials: isset($data['credentials']) ? (string) $data['credentials'] : null,
            bio: isset($data['bio']) ? (string) $data['bio'] : null,
            avatarMediaId: isset($data['avatarMediaId']) ? (int) $data['avatarMediaId'] : null,
            sameAs: array_values(array_map(strval(...), (array) ($data['sameAs'] ?? []))),
            isMedicalReviewer: (bool) ($data['isMedicalReviewer'] ?? false),
        );
    }
}
