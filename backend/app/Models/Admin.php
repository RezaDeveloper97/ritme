<?php

namespace App\Models;

use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;

/**
 * Admin panel account. Authenticates via the `admin` session guard, entirely
 * separate from the OTP-based {@see User} model.
 */
class Admin extends Authenticatable
{
    use Notifiable;

    protected $fillable = [
        'name',
        'email',
        'password',
        'role',
        'is_active',
        'last_login_at',
    ];

    protected $hidden = [
        'password',
        'remember_token',
    ];

    protected function casts(): array
    {
        return [
            'password' => 'hashed',
            'is_active' => 'boolean',
            'last_login_at' => 'datetime',
        ];
    }

    public const ROLE_SUPER = 'super';
    public const ROLE_EDITOR = 'editor';

    /**
     * Super admins may manage other admins; editors are limited to content.
     */
    public function isSuper(): bool
    {
        return $this->role === self::ROLE_SUPER;
    }
}
