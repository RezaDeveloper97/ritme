<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

/**
 * An article author or (medical) reviewer. `credential` is the professional qualification shown to readers
 * (e.g. «متخصص زنان و زایمان»), emitted as hasCredential.
 */
final readonly class PersonData
{
    /**
     * @param  list<string>  $sameAs  profile URLs (LinkedIn, medical council page …)
     * @param  list<string>  $knowsAbout  topics of expertise
     */
    public function __construct(
        public string $name,
        public ?string $url = null,
        public ?string $jobTitle = null,
        public ?string $imageUrl = null,
        public array $sameAs = [],
        public array $knowsAbout = [],
        public ?string $credential = null,
        public ?string $description = null,
    ) {}
}
