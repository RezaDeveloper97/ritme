<?php

namespace App\Services\HomePage\Contracts;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * A single, self-contained home-page widget ("item").
 *
 * Each section is independent: it declares the modes it supports, its display
 * order, and how to build itself from the shared HomeContext. Adding, removing
 * or reordering a section never touches any other section — this is what makes
 * every item on the page individually replaceable.
 */
interface HomeSectionInterface
{
    /**
     * Stable unique key (also used as the {section} route parameter).
     */
    public function key(): string;

    /**
     * Display order within the page (ascending).
     */
    public function order(): int;

    /**
     * Whether this section should appear for the given context
     * (e.g. only in cycle mode, only when a profile exists).
     */
    public function supports(HomeContext $context): bool;

    /**
     * Build the section payload. Returning null omits the section entirely
     * (e.g. nothing to show right now).
     */
    public function build(HomeContext $context): ?HomeSection;
}
