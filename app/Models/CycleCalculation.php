<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="CycleCalculation",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="calculation_date", type="string", format="date", example="2024-12-07"),
 *     @OA\Property(property="version", type="integer", example=1),
 *     @OA\Property(property="is_locked", type="boolean", example=false),
 *     @OA\Property(property="cycle_day", type="integer", example=14),
 *     @OA\Property(property="phase", type="string", example="ovulation"),
 *     @OA\Property(property="subphase", type="string", example="ovulation_window"),
 *     @OA\Property(property="estimated_ovulation_day", type="integer", example=14),
 *     @OA\Property(property="cycle_length_used", type="integer", example=28),
 *     @OA\Property(property="is_fertile_window", type="boolean", example=true),
 *     @OA\Property(property="is_pms_window", type="boolean", example=false),
 *     @OA\Property(property="is_period_tomorrow", type="boolean", example=false),
 *     @OA\Property(property="is_luteal_spotting", type="boolean", example=false),
 *     @OA\Property(property="cycle_score", type="number", format="float", example=0.70),
 *     @OA\Property(property="age_factor", type="number", format="float", example=0.85),
 *     @OA\Property(property="base_probability", type="number", format="float", example=0.30),
 *     @OA\Property(property="symptom_score", type="number", format="float", example=1.30),
 *     @OA\Property(property="final_probability", type="number", format="float", example=19.89),
 *     @OA\Property(property="cycle_variability", type="string", example="regular"),
 *     @OA\Property(property="uncertainty_range", type="integer", example=1),
 *     @OA\Property(property="text_flags", type="object"),
 *     @OA\Property(property="daily_tips", type="object"),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class CycleCalculation extends Model
{
    protected $fillable = [
        'user_id',
        'calculation_date',
        'version',
        'is_locked',
        'cycle_day',
        'phase',
        'subphase',
        'estimated_ovulation_day',
        'cycle_length_used',
        'is_fertile_window',
        'is_pms_window',
        'is_period_tomorrow',
        'is_luteal_spotting',
        'cycle_score',
        'age_factor',
        'base_probability',
        'symptom_score',
        'final_probability',
        'cycle_variability',
        'uncertainty_range',
        'text_flags',
        'daily_tips',
        'source_profile_data',
        'source_daily_log_data',
    ];

    protected function casts(): array
    {
        return [
            'calculation_date' => 'date',
            'is_locked' => 'boolean',
            'is_fertile_window' => 'boolean',
            'is_pms_window' => 'boolean',
            'is_period_tomorrow' => 'boolean',
            'is_luteal_spotting' => 'boolean',
            'cycle_score' => 'decimal:4',
            'age_factor' => 'decimal:4',
            'base_probability' => 'decimal:4',
            'symptom_score' => 'decimal:4',
            'final_probability' => 'decimal:2',
            'text_flags' => 'array',
            'daily_tips' => 'array',
            'source_profile_data' => 'array',
            'source_daily_log_data' => 'array',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    /**
     * Scope to get latest version for a date
     */
    public function scopeLatestVersion($query, $date)
    {
        return $query->where('calculation_date', $date)
            ->orderBy('version', 'desc')
            ->limit(1);
    }

    /**
     * Scope to get only unlocked calculations
     */
    public function scopeUnlocked($query)
    {
        return $query->where('is_locked', false);
    }

    /**
     * Scope to get only locked (historical) calculations
     */
    public function scopeLocked($query)
    {
        return $query->where('is_locked', true);
    }
}
