<?php

namespace App\Models;

use App\Enums\AlertLevel;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="PregnancyAlert",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="alert_level", type="string", enum={"info","warning","emergency"}),
 *     @OA\Property(property="alert_type", type="string", example="symptom_based"),
 *     @OA\Property(property="title", type="string", example="Warning: Spotting Detected"),
 *     @OA\Property(property="message", type="string", example="You have reported spotting. Please monitor closely and consult your doctor if it continues."),
 *     @OA\Property(property="pregnancy_week", type="integer", nullable=true, example=12),
 *     @OA\Property(property="trigger_symptoms", type="object", nullable=true),
 *     @OA\Property(property="medical_history_flags", type="object", nullable=true),
 *     @OA\Property(property="is_read", type="boolean", example=false),
 *     @OA\Property(property="is_dismissed", type="boolean", example=false),
 *     @OA\Property(property="read_at", type="string", format="date-time", nullable=true),
 *     @OA\Property(property="dismissed_at", type="string", format="date-time", nullable=true),
 *     @OA\Property(property="recommended_actions", type="array", nullable=true, @OA\Items(type="string")),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PregnancyAlert extends Model
{
    protected $fillable = [
        'user_id',
        'alert_level',
        'alert_type',
        'title',
        'message',
        'pregnancy_week',
        'trigger_symptoms',
        'medical_history_flags',
        'is_read',
        'is_dismissed',
        'read_at',
        'dismissed_at',
        'recommended_actions',
    ];

    protected function casts(): array
    {
        return [
            'is_read' => 'boolean',
            'is_dismissed' => 'boolean',
            'read_at' => 'datetime',
            'dismissed_at' => 'datetime',
            'trigger_symptoms' => 'array',
            'medical_history_flags' => 'array',
            'recommended_actions' => 'array',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    /**
     * Get available enum values for API documentation/validation
     */
    public static function getEnumValues(): array
    {
        return [
            'alert_level' => AlertLevel::values(),
        ];
    }

    /**
     * Scope for unread alerts
     */
    public function scopeUnread($query)
    {
        return $query->where('is_read', false);
    }

    /**
     * Scope for active (not dismissed) alerts
     */
    public function scopeActive($query)
    {
        return $query->where('is_dismissed', false);
    }

    /**
     * Scope for specific alert level
     */
    public function scopeLevel($query, string $level)
    {
        return $query->where('alert_level', $level);
    }

    /**
     * Mark alert as read
     */
    public function markAsRead(): void
    {
        $this->update([
            'is_read' => true,
            'read_at' => now(),
        ]);
    }

    /**
     * Dismiss alert
     */
    public function dismiss(): void
    {
        $this->update([
            'is_dismissed' => true,
            'dismissed_at' => now(),
        ]);
    }
}
