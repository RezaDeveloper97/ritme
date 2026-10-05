<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Exceptions;

use App\Domain\Shop\Cart\Data\CartNotice;
use App\Domain\Shop\Ordering\Enums\CheckoutProblem;
use RuntimeException;

/**
 * The order was refused (nothing stored, stock untouched). `problem` + `params` → a Persian message in the view;
 * `notices` = what the live cart re-check changed (price / quantity / removed lines, cart copy) for CartChanged.
 */
final class CheckoutException extends RuntimeException
{
    /**
     * @param  array<string, string|int>  $params
     * @param  list<CartNotice>  $notices
     */
    public function __construct(public readonly CheckoutProblem $problem, public readonly array $params = [], public readonly array $notices = [])
    {
        parent::__construct('Checkout refused: '.$problem->value);
    }
}
