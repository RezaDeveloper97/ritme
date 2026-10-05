<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Enums;

/**
 * Why an order was not placed. Copy: lang/fa/shop.php `checkout.problems.<value>`.
 */
enum CheckoutProblem: string
{
    case EmptyCart = 'empty_cart';
    case CartChanged = 'cart_changed';
    case Unavailable = 'unavailable';
    case OutOfStock = 'out_of_stock';
    case CodLimit = 'cod_limit';
}
