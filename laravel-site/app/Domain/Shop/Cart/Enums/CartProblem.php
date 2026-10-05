<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Enums;

/**
 * Why a cart change was refused or adjusted, or what changed on re-validation. The Persian copy lives in
 * `lang/fa/shop.php` under `cart.problems.<value>` (params: `name`, `count`, `old`, `new`).
 */
enum CartProblem: string
{
    case NotFound = 'not_found';
    case VariantRequired = 'variant_required';
    case VariantUnknown = 'variant_unknown';
    case OutOfStock = 'out_of_stock';
    case Limited = 'limited';
    case MaxQuantity = 'max_quantity';
    case AllInCart = 'all_in_cart';
    case TooManyLines = 'too_many_lines';
    case LineMissing = 'line_missing';
    case Removed = 'removed';
    case PriceChanged = 'price_changed';

    public function translationKey(): string
    {
        return 'shop.cart.problems.'.$this->value;
    }
}
