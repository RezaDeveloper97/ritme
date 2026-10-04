<?php

declare(strict_types=1);

namespace App\Domain\Faq\Actions;

use App\Support\Cache\NamespaceVersions;

/**
 * Bumps `faq` and `pages` for writes that bypass model events: the admin's drag-sort (one mass UPDATE) and seeding
 * with muted events. Normal saves/deletes are covered by FaqObserver.
 */
final class InvalidateFaqCache
{
    public function __construct(private readonly NamespaceVersions $versions) {}

    public function __invoke(): void
    {
        $this->versions->bump('faq', 'pages');
    }
}
