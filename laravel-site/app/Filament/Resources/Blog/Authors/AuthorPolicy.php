<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Authors;

use App\Filament\Resources\Blog\BlogPolicy;

/**
 * See BlogPolicy: editors write, SEO managers edit the SEO tab only, everyone else is denied.
 */
final class AuthorPolicy extends BlogPolicy {}
