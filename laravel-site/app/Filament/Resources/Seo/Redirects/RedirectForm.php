<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects;

use App\Domain\Seo\Redirects\InvalidRedirect;
use Illuminate\Validation\ValidationException;

/**
 * Turns a domain InvalidRedirect into a form validation error on the right field.
 */
final class RedirectForm
{
    public static function fail(InvalidRedirect $e, string $prefix = 'data.'): ValidationException
    {
        return ValidationException::withMessages([$prefix.$e->field => $e->getMessage()]);
    }
}
