<?php

declare(strict_types=1);

namespace App\Domain\Seo\Support;

use Stringable;

/**
 * A robots meta directive: index/noindex, follow/nofollow plus extra directives (max-image-preview:large…).
 * Parsed from / rendered to the comma-separated form stored in seo_meta.robots.
 */
final readonly class Robots implements Stringable
{
    public const DEFAULT_EXTRAS = ['max-image-preview:large', 'max-snippet:-1', 'max-video-preview:-1'];

    /**
     * @param  list<string>  $extras
     */
    public function __construct(
        public bool $index = true,
        public bool $follow = true,
        public array $extras = self::DEFAULT_EXTRAS,
    ) {}

    public static function default(): self
    {
        return new self;
    }

    /**
     * Unknown/omitted index or follow directives fall back to index / follow; `none` = noindex,nofollow,
     * `all` = index,follow.
     */
    public static function parse(?string $value): self
    {
        $index = true;
        $follow = true;
        $extras = [];

        foreach (explode(',', strtolower((string) $value)) as $directive) {
            $directive = trim($directive);
            match (true) {
                $directive === '' => null,
                $directive === 'index', $directive === 'all' => null,
                $directive === 'follow' => null,
                $directive === 'noindex' => $index = false,
                $directive === 'nofollow' => $follow = false,
                $directive === 'none' => [$index, $follow] = [false, false],
                default => $extras[] = $directive,
            };
        }

        return new self($index, $follow, array_values(array_unique($extras)));
    }

    public function withNoindex(): self
    {
        return new self(false, $this->follow, $this->extras);
    }

    public function __toString(): string
    {
        $directives = [$this->index ? 'index' : 'noindex', $this->follow ? 'follow' : 'nofollow'];

        // Snippet/preview directives only matter for indexable pages.
        return implode(',', $this->index ? [...$directives, ...$this->extras] : $directives);
    }
}
