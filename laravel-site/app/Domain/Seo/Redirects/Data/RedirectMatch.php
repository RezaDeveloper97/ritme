<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Data;

use App\Domain\Seo\Redirects\Enums\RedirectCode;

final readonly class RedirectMatch
{
    /**
     * @param  string|null  $target  decoded local path (`/x?y`) or absolute URL; null for 410
     */
    public function __construct(public int $id, public ?string $target, public RedirectCode $code) {}
}
