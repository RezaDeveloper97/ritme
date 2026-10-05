<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

/**
 * The cart aggregate: lines keyed by `p{product}[-v{variant}]` in insertion order. Holds no prices the client could
 * set and no catalog data — only what fits a session comfortably. `fromArray()` is defensive (a tampered or outdated
 * session payload is cleaned, never trusted): junk rows are skipped, quantities clamped to 1…MAX_QUANTITY, at most
 * MAX_LINES lines.
 */
final class Cart
{
    public const MAX_LINES = 30;

    public const MAX_QUANTITY = 10;

    private const VERSION = 1;

    /** @var array<string, CartLine> */
    private array $lines = [];

    /**
     * @param  iterable<CartLine>  $lines
     */
    public function __construct(iterable $lines = [])
    {
        foreach ($lines as $line) {
            if (count($this->lines) >= self::MAX_LINES) {
                break;
            }
            $this->put($line);
        }
    }

    /**
     * @return list<CartLine>
     */
    public function lines(): array
    {
        return array_values($this->lines);
    }

    public function line(string $key): ?CartLine
    {
        return $this->lines[$key] ?? null;
    }

    public function has(string $key): bool
    {
        return isset($this->lines[$key]);
    }

    public function quantityOf(string $key): int
    {
        return $this->lines[$key]->quantity ?? 0;
    }

    /** Adds or replaces a line (quantity clamped to 1…MAX_QUANTITY; ≤ 0 removes it). */
    public function put(CartLine $line): void
    {
        if ($line->quantity < 1) {
            $this->remove($line->key());

            return;
        }

        $this->lines[$line->key()] = $line->quantity > self::MAX_QUANTITY ? $line->withQuantity(self::MAX_QUANTITY) : $line;
    }

    public function remove(string $key): void
    {
        unset($this->lines[$key]);
    }

    public function clear(): void
    {
        $this->lines = [];
    }

    public function isEmpty(): bool
    {
        return $this->lines === [];
    }

    public function isFull(): bool
    {
        return count($this->lines) >= self::MAX_LINES;
    }

    public function lineCount(): int
    {
        return count($this->lines);
    }

    /** Number of items (sum of the quantities). */
    public function count(): int
    {
        return array_sum(array_map(static fn (CartLine $line): int => $line->quantity, $this->lines));
    }

    /**
     * @return list<int>
     */
    public function productIds(): array
    {
        return array_values(array_unique(array_map(static fn (CartLine $line): int => $line->productId, array_values($this->lines))));
    }

    /**
     * Compact session payload: {v, l: [[product, variant|null, quantity, unit price rial], …]}.
     *
     * @return array{v: int, l: list<array{0: int, 1: int|null, 2: int, 3: int}>}
     */
    public function toArray(): array
    {
        return [
            'v' => self::VERSION,
            'l' => array_map(static fn (CartLine $l): array => [$l->productId, $l->variantId, $l->quantity, $l->unitPriceRial], array_values($this->lines)),
        ];
    }

    public static function fromArray(mixed $data): self
    {
        if (! is_array($data) || ($data['v'] ?? null) !== self::VERSION || ! is_array($data['l'] ?? null)) {
            return new self;
        }

        $lines = [];
        foreach ($data['l'] as $row) {
            if (! is_array($row) || ! is_int($row[0] ?? null) || $row[0] < 1 || ! is_int($row[2] ?? null) || $row[2] < 1) {
                continue;
            }
            $variant = $row[1] ?? null;
            if ($variant !== null && (! is_int($variant) || $variant < 1)) {
                continue;
            }
            $price = $row[3] ?? 0;
            $lines[] = new CartLine($row[0], $variant, $row[2], is_int($price) && $price >= 0 ? $price : 0);
        }

        return new self($lines);
    }
}
