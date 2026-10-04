<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

/**
 * Structure of one tool / service card on a stage page. Copy (title, text) is `lang/fa/stages/<slug>.php`
 * `tools.<key>` / `help.<key>`; `where` adds the «روی سایت» / «در اپ» tag of tool cards.
 */
final readonly class CardSpec
{
    public const ON_SITE = 'on_site';

    public const IN_APP = 'in_app';

    /**
     * @param  string  $color  stage colour key (cycle, ttc, …) or primary / muted — see x-ui.icon-tile
     * @param  string  $route  named route of the link target
     */
    public function __construct(
        public string $key,
        public string $icon,
        public string $color,
        public string $route,
        public string $fragment = '',
        public ?string $where = null,
    ) {}
}
