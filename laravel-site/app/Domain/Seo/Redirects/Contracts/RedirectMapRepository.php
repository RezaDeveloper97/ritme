<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Contracts;

use App\Domain\Seo\Redirects\Data\RedirectMap;

interface RedirectMapRepository
{
    public function map(): RedirectMap;
}
