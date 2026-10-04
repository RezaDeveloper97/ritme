<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Models;

use App\Domain\Newsletter\Enums\SubscriptionStatus;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;
use Illuminate\Support\Str;

/**
 * A newsletter address (double opt-in). Only active subscribers (confirmed, not unsubscribed) may be mailed.
 *
 * @property int $id
 * @property string $email
 * @property string|null $source
 * @property string $token
 * @property Carbon $consent_at
 * @property Carbon|null $confirmation_sent_at
 * @property Carbon|null $confirmed_at
 * @property Carbon|null $unsubscribed_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Subscriber extends Model
{
    protected $table = 'newsletter_subscribers';

    protected $fillable = ['email', 'source', 'token', 'consent_at', 'confirmation_sent_at', 'confirmed_at', 'unsubscribed_at'];

    protected $hidden = ['token'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'consent_at' => 'datetime',
            'confirmation_sent_at' => 'datetime',
            'confirmed_at' => 'datetime',
            'unsubscribed_at' => 'datetime',
        ];
    }

    public static function newToken(): string
    {
        return Str::random(48);
    }

    public static function normalizeEmail(string $email): string
    {
        return mb_strtolower(trim($email));
    }

    public function status(): SubscriptionStatus
    {
        return match (true) {
            $this->unsubscribed_at !== null => SubscriptionStatus::Unsubscribed,
            $this->confirmed_at !== null => SubscriptionStatus::Active,
            default => SubscriptionStatus::Pending,
        };
    }

    /**
     * Confirmed and not unsubscribed.
     *
     * @param  Builder<self>  $query
     */
    public function scopeActive(Builder $query): void
    {
        $query->whereNotNull('confirmed_at')->whereNull('unsubscribed_at');
    }
}
