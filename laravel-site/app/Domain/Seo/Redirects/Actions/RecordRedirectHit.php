<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Support\HitBuffer;

/**
 * Counts one use of a redirect in the cache buffer (FlushRedirectStats writes `hits` / `last_hit_at`).
 */
final class RecordRedirectHit
{
    public const BUCKET = 'redirect';

    public function __construct(private readonly HitBuffer $buffer) {}

    public function handle(int $redirectId): void
    {
        $this->buffer->hit(self::BUCKET, (string) $redirectId);
    }
}
