<?php

namespace App\Models;

use App\Enums\FetalMovementStatus;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="PregnancyFetalMovement",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="log_date", type="string", format="date", example="2024-12-14"),
 *     @OA\Property(property="pregnancy_week", type="integer", example=24),
 *     @OA\Property(property="movement_status", type="string", enum={"not_felt_yet","felt","normal","reduced","increased","none"}),
 *     @OA\Property(property="movement_count", type="integer", nullable=true, example=10),
 *     @OA\Property(property="first_movement_time", type="string", format="time", nullable=true, example="09:30:00"),
 *     @OA\Property(property="last_movement_time", type="string", format="time", nullable=true, example="21:00:00"),
 *     @OA\Property(property="notes", type="string", nullable=true),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PregnancyFetalMovement extends Model
{
    protected $fillable = [
        'user_id',
        'log_date',
        'pregnancy_week',
        'movement_status',
        'movement_count',
        'first_movement_time',
        'last_movement_time',
        'notes',
    ];

    protected function casts(): array
    {
        return [
            'log_date' => 'date:Y-m-d',
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
            'movement_status' => FetalMovementStatus::values(),
        ];
    }

    /**
     * Check if movement tracking requires attention
     */
    public function requiresAttention(): bool
    {
        return $this->movement_status === FetalMovementStatus::REDUCED->value ||
               $this->movement_status === FetalMovementStatus::NONE->value;
    }
}
