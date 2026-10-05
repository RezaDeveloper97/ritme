<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Exceptions;

use App\Domain\Shop\Cart\Enums\CartProblem;
use RuntimeException;

/**
 * A cart change that could not be made (unknown product / variant, sold out, cart full …). Nothing was stored, except
 * that a line whose product vanished is dropped. `productSlug` lets the delivery layer send the shopper back.
 */
final class CartException extends RuntimeException
{
    /**
     * @param  array<string, string|int>  $params
     */
    public function __construct(public readonly CartProblem $problem, public readonly array $params = [], public readonly ?string $productSlug = null)
    {
        parent::__construct('Cart: '.$problem->value);
    }
}
