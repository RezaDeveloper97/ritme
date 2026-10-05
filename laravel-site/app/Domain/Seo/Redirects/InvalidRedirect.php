<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects;

use DomainException;

/**
 * A redirect that may not be saved (invalid path / pattern, duplicate source, loop). `field` names the form field.
 */
final class InvalidRedirect extends DomainException
{
    public function __construct(public readonly string $field, string $message)
    {
        parent::__construct($message);
    }
}
