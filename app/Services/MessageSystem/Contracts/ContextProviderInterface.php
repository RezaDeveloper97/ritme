<?php

namespace App\Services\MessageSystem\Contracts;

use App\Models\User;
use App\Services\MessageSystem\Core\MessageContext;
use Carbon\Carbon;

interface ContextProviderInterface
{
    /**
     * Build message context for a user on a specific date
     */
    public function buildContext(User $user, Carbon $date, string $locale = 'en'): MessageContext;

    /**
     * Detect the current mode for the user
     */
    public function detectMode(User $user): string;
}
