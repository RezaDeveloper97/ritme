<?php

declare(strict_types=1);

namespace App\Domain\Faq\Contracts;

use App\Domain\Faq\Data\FaqGroupData;

/**
 * FAQ groups with their published items. Cached in the `faq` namespace (FaqObserver bumps it, and `pages`).
 */
interface FaqRepository
{
    /**
     * The group with this slug, or null when no such group exists (a group whose items are all unpublished comes back
     * with an empty item list, so callers can tell "hidden on purpose" from "not seeded").
     */
    public function group(string $slug): ?FaqGroupData;

    /**
     * Groups shown on /faq (`is_listed`) that have at least one published item, by sort order.
     *
     * @return list<FaqGroupData>
     */
    public function listed(): array;
}
