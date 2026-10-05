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

            return new RedirectMatch($id, is_string($resolved) ? $resolved : null, RedirectCode::from($code));
        }

        return null;
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
