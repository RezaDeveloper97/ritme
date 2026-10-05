<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing;

use InvalidArgumentException;

/** Indexing settings rejected by SaveIndexingSettings; errors() maps setting key => Persian messages. */
final class InvalidIndexingSettings extends InvalidArgumentException
{
    /**
     * @param  array<string, list<string>>  $errors
     */
    public function __construct(private readonly array $errors)
    {
        parent::__construct('Invalid indexing settings: '.implode(', ', array_keys($errors)));
    }

    /**
     * @return array<string, list<string>>
     */
    public function errors(): array
    {
        return $this->errors;
    }
}
