<?php

declare(strict_types=1);

namespace App\Domain\Seo\Data;

/**
 * The title / description a static page renders when its seo_meta row leaves them empty (the page controller's
 * copy from lang/fa). `$titleIsComplete`: the title already carries the brand and is used verbatim; otherwise the
 * site title template («%s — ریتمی») is applied, exactly as the controller does.
 */
final readonly class StaticPageDefaults
{
    public function __construct(
        public string $title,
        public bool $titleIsComplete,
        public string $description,
    ) {}
}
