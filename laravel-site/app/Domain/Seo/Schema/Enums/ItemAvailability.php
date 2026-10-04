<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Enums;

enum ItemAvailability: string
{
    case InStock = 'InStock';
    case OutOfStock = 'OutOfStock';
    case PreOrder = 'PreOrder';
    case BackOrder = 'BackOrder';
    case LimitedAvailability = 'LimitedAvailability';
    case Discontinued = 'Discontinued';

    public function uri(): string
    {
        return 'https://schema.org/'.$this->value;
    }
}
