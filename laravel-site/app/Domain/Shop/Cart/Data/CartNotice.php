<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

use App\Domain\Shop\Cart\Enums\CartProblem;

/**
 * Something the shopper should know about the cart (a line dropped, a quantity reduced, a price changed). The view
 * translates `problem` with `params`.
 */
final readonly class CartNotice
{
    /**
     * @param  array<string, string|int>  $params
     */
    public function __construct(public CartProblem $problem, public array $params = []) {}
}
