<?php

namespace App\Models;

use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class UserProfile extends Model
{
    use HasFactory;

    protected $fillable = [
        'user_id',
        'birthday',
        'weight',
        'height',
        'period_duration',
        'cycle_duration',
        'last_period_start',
        'user_goal',
        'subscription_type',
        'calculation_status',
        'calculation_started_at',
        'calculation_completed_at',
        'calculation_version',
    ];

    protected function casts(): array
    {
        return [
            'birthday' => 'date',
            'weight' => 'decimal:2',
            'height' => 'integer',
            'period_duration' => 'integer',
            'cycle_duration' => 'integer',
            'last_period_start' => 'date',
            'calculation_started_at' => 'datetime',
            'calculation_completed_at' => 'datetime',
            'calculation_version' => 'integer',
        ];
    }

    /**
     * Get the user that owns the profile
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    /**
     * Check if user is trying to conceive
     */
    public function isTTC(): bool
    {
        return $this->user_goal === UserGoal::TTC->value;
    }

    /**
     * Check if user has premium subscription
     */
    public function isPremium(): bool
    {
        return $this->subscription_type === SubscriptionType::PREMIUM->value;
    }

    /**
     * Get available enum values for user goal and subscription
     */
    public static function getGoalEnumValues(): array
    {
        return [
            'user_goal' => UserGoal::values(),
            'subscription_type' => SubscriptionType::values(),
        ];
    }
}
