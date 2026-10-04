<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

/**
 * A resolved tool / service card; toProps() feeds x-stage.tools-block / x-stage.help-block (feature / service card
 * props).
 */
final readonly class StageCardData
{
    public function __construct(
        public string $href,
        public string $icon,
        public string $color,
        public string $title,
        public ?string $text = null,
        public ?string $where = null,
        public ?string $cta = null,
    ) {}

    /**
     * @return array{href: string, icon: string, color: string, title: string, text: string|null, where: string|null, cta: string|null}
     */
    public function toProps(): array
    {
        return [
            'href' => $this->href,
            'icon' => $this->icon,
            'color' => $this->color,
            'title' => $this->title,
            'text' => $this->text,
            'where' => $this->where,
            'cta' => $this->cta,
        ];
    }
}
