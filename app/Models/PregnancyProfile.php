<?php

namespace App\Models;

use App\Enums\BloodType;
use App\Enums\ConfidenceLevel;
use App\Enums\PreExistingCondition;
use App\Enums\PregnancyAgeSource;
use App\Enums\RhFactor;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @OA\Schema(
 *     schema="PregnancyProfile",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="pregnancy_mode", type="boolean", example=true),
 *     @OA\Property(property="cycle_mode", type="boolean", example=false),
 *     @OA\Property(property="is_locked", type="boolean", example=false),
 *     @OA\Property(property="age_source", type="string", nullable=true, enum={"lmp","ultrasound","manual"}),
 *     @OA\Property(property="confidence_level", type="string", nullable=true, enum={"high","medium","low"}),
 *     @OA\Property(property="lmp_date", type="string", format="date", nullable=true),
 *     @OA\Property(property="ultrasound_date", type="string", format="date", nullable=true),
 *     @OA\Property(property="ultrasound_weeks", type="integer", nullable=true),
 *     @OA\Property(property="ultrasound_days", type="integer", nullable=true),
 *     @OA\Property(property="manual_weeks", type="integer", nullable=true),
 *     @OA\Property(property="manual_days", type="integer", nullable=true),
 *     @OA\Property(property="manual_entry_date", type="string", format="date", nullable=true),
 *     @OA\Property(property="estimated_due_date", type="string", format="date", nullable=true),
 *     @OA\Property(property="estimated_conception_date", type="string", format="date", nullable=true),
 *     @OA\Property(property="uncertainty_days", type="integer", example=3),
 *     @OA\Property(property="has_miscarriage_history", type="boolean", nullable=true),
 *     @OA\Property(property="has_high_risk_history", type="boolean", nullable=true),
 *     @OA\Property(property="pre_existing_conditions", type="array", nullable=true, @OA\Items(type="string", enum={"chronic_hypertension","diabetes","hypothyroidism","hyperthyroidism","none"})),
 *     @OA\Property(property="blood_type", type="string", nullable=true, enum={"A","B","AB","O"}),
 *     @OA\Property(property="rh_factor", type="string", nullable=true, enum={"positive","negative"}),
 *     @OA\Property(property="rh_negative_care_flag", type="boolean", example=false),
 *     @OA\Property(property="first_fetal_movement_date", type="string", format="date", nullable=true),
 *     @OA\Property(property="fetal_movement_felt", type="boolean", example=false),
 *     @OA\Property(property="onboarding_completed", type="boolean", example=false),
 *     @OA\Property(property="onboarding_completed_at", type="string", format="date-time", nullable=true),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PregnancyProfile extends Model
{
    protected $fillable = [
        'user_id',

        // Mode flags
        'pregnancy_mode',
        'cycle_mode',
        'is_locked',

        // Age source and calculation
        'age_source',
        'confidence_level',

        // LMP data
        'lmp_date',

        // Ultrasound data
        'ultrasound_date',
        'ultrasound_weeks',
        'ultrasound_days',

        // Manual entry data
        'manual_weeks',
        'manual_days',
        'manual_entry_date',

        // Calculated data
        'estimated_due_date',
        'estimated_conception_date',
        'uncertainty_days',

        // Pregnancy history
        'has_miscarriage_history',
        'has_high_risk_history',

        // Pre-existing conditions
        'pre_existing_conditions',

        // Blood information
        'blood_type',
        'rh_factor',
        'rh_negative_care_flag',

        // Fetal movement
        'first_fetal_movement_date',
        'fetal_movement_felt',

        // Onboarding
        'onboarding_completed',
        'onboarding_completed_at',
    ];

    protected function casts(): array
    {
        return [
            // Booleans
            'pregnancy_mode' => 'boolean',
            'cycle_mode' => 'boolean',
            'is_locked' => 'boolean',
            'has_miscarriage_history' => 'boolean',
            'has_high_risk_history' => 'boolean',
            'rh_negative_care_flag' => 'boolean',
            'fetal_movement_felt' => 'boolean',
            'onboarding_completed' => 'boolean',

            // Dates
            'lmp_date' => 'date',
            'ultrasound_date' => 'date',
            'manual_entry_date' => 'date',
            'estimated_due_date' => 'date',
            'estimated_conception_date' => 'date',
            'first_fetal_movement_date' => 'date',
            'onboarding_completed_at' => 'datetime',

            // JSON
            'pre_existing_conditions' => 'array',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function symptomLogs(): HasMany
    {
        return $this->hasMany(PregnancySymptomLog::class, 'user_id', 'user_id');
    }

    public function weeklyLogs(): HasMany
    {
        return $this->hasMany(PregnancyWeeklyLog::class, 'user_id', 'user_id');
    }

    public function fetalMovements(): HasMany
    {
        return $this->hasMany(PregnancyFetalMovement::class, 'user_id', 'user_id');
    }

    public function alerts(): HasMany
    {
        return $this->hasMany(PregnancyAlert::class, 'user_id', 'user_id');
    }

    /**
     * Get available enum values for API documentation/validation
     */
    public static function getEnumValues(): array
    {
        return [
            'age_source' => PregnancyAgeSource::values(),
            'confidence_level' => ConfidenceLevel::values(),
            'blood_type' => BloodType::values(),
            'rh_factor' => RhFactor::values(),
            'pre_existing_conditions' => PreExistingCondition::values(),
        ];
    }
}
