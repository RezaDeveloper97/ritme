<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

enum Severity: string
{
    case Error = 'error';
    case Warning = 'warning';
}
