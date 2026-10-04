<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts;

use App\Filament\Resources\Blog\BlogPolicy;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;

/**
 * See BlogPolicy: editors write, SEO managers edit the SEO tab only, everyone else is denied.
 */
final class PostPolicy extends BlogPolicy
{
    /**
     * «Duplicate post» creates a new draft: editors only.
     */
    public function replicate(User $user, Model $record): bool
    {
        return $this->allows($user);
    }
}
