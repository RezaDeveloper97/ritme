<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Support;

use Illuminate\Contracts\Session\Session;

/**
 * Checkout state in the visitor's session:
 *  - the one-time form token (idempotency): issued when the checkout page renders, consumed when an order is placed —
 *    a second submit of the same form finds the order by the token's hash instead of placing another one;
 *  - the codes of the orders placed in this session (the order page shows the recipient rows only to their owner).
 */
final class CheckoutSession
{
    public const TOKEN_KEY = 'shop_checkout_token';

    public const ORDERS_KEY = 'shop_orders';

    private const MAX_ORDERS = 20;

    public function __construct(private readonly Session $session) {}

    /** The current form token (kept across re-renders so a back/forward or a validation error keeps working). */
    public function token(): string
    {
        $token = $this->session->get(self::TOKEN_KEY);
        if (! is_string($token) || strlen($token) !== 40) {
            $token = bin2hex(random_bytes(20));
            $this->session->put(self::TOKEN_KEY, $token);
        }

        return $token;
    }

    public function isCurrent(string $token): bool
    {
        $current = $this->session->get(self::TOKEN_KEY);

        return is_string($current) && $token !== '' && hash_equals($current, $token);
    }

    public function remember(string $code): void
    {
        $codes = $this->codes();
        $codes[] = $code;
        $this->session->put(self::ORDERS_KEY, array_slice(array_values(array_unique($codes)), -self::MAX_ORDERS));
        $this->session->forget(self::TOKEN_KEY); // the next checkout gets a fresh token
    }

    public function owns(string $code): bool
    {
        return in_array($code, $this->codes(), true);
    }

    public static function key(string $token): string
    {
        return hash('sha256', 'shop-checkout|'.$token);
    }

    /**
     * @return list<string>
     */
    private function codes(): array
    {
        $codes = $this->session->get(self::ORDERS_KEY);

        return is_array($codes) ? array_values(array_filter($codes, 'is_string')) : [];
    }
}
