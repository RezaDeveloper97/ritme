<?php

declare(strict_types=1);

namespace App\Domain\Content\Data;

/**
 * One main-nav link. `active` marks the item of the current section; `current` is true only when the link points
 * at the current page itself (aria-current="page"); a section match without it gets aria-current="true".
 */
final readonly class NavLinkData
{
    public function __construct(
        public string $label,
        public string $url,
        public bool $active = false,
        public bool $current = false,
        public bool $chevron = false,
    ) {}

    public function ariaCurrent(): ?string
    {
        return $this->current ? 'page' : ($this->active ? 'true' : null);
    }
}
