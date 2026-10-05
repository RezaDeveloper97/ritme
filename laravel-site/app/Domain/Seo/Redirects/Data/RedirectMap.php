<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Data;

use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\Support\RedirectPath;

/**
 * Every redirect in one cacheable structure: a hash map of exact sources (lower-cased normalised path → row) and a
 * short ordered list of compiled regex rows, tried only when no exact row matches. Cached as a plain array
 * (toArray / fromArray) so a code change never unserializes stale objects.
 */
final readonly class RedirectMap
{
    /**
     * @param  array<string, array{0: int, 1: string|null, 2: int}>  $exact  key => [id, target, code]
     * @param  list<array{0: int, 1: string, 2: string|null, 3: int}>  $regex  [id, compiled pattern, target, code]
     */
    public function __construct(public array $exact = [], public array $regex = []) {}

    public function match(string $path): ?RedirectMatch
    {
        $key = RedirectPath::key($path);

        if (isset($this->exact[$key])) {
            [$id, $target, $code] = $this->exact[$key];

            return new RedirectMatch($id, $target, RedirectCode::from($code));
        }

        $path = RedirectPath::normalize($path);
        foreach ($this->regex as [$id, $pattern, $target, $code]) {
            if (@preg_match($pattern, $path) !== 1) {
                continue;
            }

            $resolved = $target === null ? null : @preg_replace($pattern, $target, $path);

            return new RedirectMatch($id, is_string($resolved) ? self::confine($target, $resolved) : null, RedirectCode::from($code));
        }

        return null;
    }

    /**
     * L9-04: a captured group must never move a regex redirect to another host. A relative template stays on this
     * site (`/$1` with `/evil.com` captured would give the protocol-relative `//evil.com`, so leading slashes and
     * backslashes collapse to one «/»); an absolute template must keep its own host (`https://ritme.ir$1` with
     * `@evil.com` captured would put ritme.ir in the user-info) — otherwise there is no redirect (null → 404 path).
     */
    private static function confine(string $template, string $resolved): ?string
    {
        if (str_starts_with($template, '/')) {
            return '/'.ltrim($resolved, '/\\');
        }

        $expected = parse_url((string) preg_replace('/\$\{?\d+\}?|\\\\\d+/', '', $template), PHP_URL_HOST);
        $actual = parse_url($resolved, PHP_URL_HOST);
        $parts = parse_url($resolved);

        return is_string($expected) && is_string($actual) && strtolower($expected) === strtolower($actual)
            && is_array($parts) && ! isset($parts['user']) && ! isset($parts['pass']) && ! str_contains($resolved, '\\')
            ? $resolved
            : null;
    }

    public function isEmpty(): bool
    {
        return $this->exact === [] && $this->regex === [];
    }

    /**
     * @return array{exact: array<string, array{0: int, 1: string|null, 2: int}>, regex: list<array{0: int, 1: string, 2: string|null, 3: int}>}
     */
    public function toArray(): array
    {
        return ['exact' => $this->exact, 'regex' => $this->regex];
    }

    /**
     * @param  array{exact?: array<string, array{0: int, 1: string|null, 2: int}>, regex?: list<array{0: int, 1: string, 2: string|null, 3: int}>}  $data
     */
    public static function fromArray(array $data): self
    {
        return new self($data['exact'] ?? [], $data['regex'] ?? []);
    }
}
