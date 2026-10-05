<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Data;

final readonly class ImportResult
{
    /**
     * @param  list<string>  $errors  first errors, "line N: message"
     */
    public function __construct(public int $created, public int $updated, public int $skipped, public array $errors) {}
}
