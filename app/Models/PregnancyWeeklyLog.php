<?php

namespace App\Models;

use App\Enums\MentalHealthStatus;
use App\Enums\SwellingLocation;
use App\Enums\SymptomSeverity;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="PregnancyWeeklyLog",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="log_date", type="string", format="date", example="2024-12-14"),
 *     @OA\Property(property="pregnancy_week", type="integer", example=12),
 *     @OA\Property(property="weight", type="number", format="float", nullable=true, example=65.5),
 *     @OA\Property(property="has_swelling", type="boolean", nullable=true),
 *     @OA\Property(property="swelling_locations", type="array", nullable=true, @OA\Items(type="string", enum={"feet","hands","face"})),
 *     @OA\Property(property="has_shortness_of_breath", type="boolean", nullable=true),
 *     @OA\Property(property="has_blood_pressure_device", type="boolean", example=false),
 *     @OA\Property(property="systolic_pressure", type="integer", nullable=true, example=120),
 *     @OA\Property(property="diastolic_pressure", type="integer", nullable=true, example=80),
 *     @OA\Property(property="fasting_blood_sugar", type="number", format="float", nullable=true),
 *     @OA\Property(property="post_meal_blood_sugar", type="number", format="float", nullable=true),
 *     @OA\Property(property="overall_mood", type="string", nullable=true, enum={"good","moderate","poor"}),
 *     @OA\Property(property="has_anxiety", type="boolean", nullable=true),
 *     @OA\Property(property="anxiety_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_mood_swings", type="boolean", nullable=true),
 *     @OA\Property(property="mood_swings_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_depression_feelings", type="boolean", nullable=true),
 *     @OA\Property(property="depression_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="notes", type="string", nullable=true),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PregnancyWeeklyLog extends Model
{
    protected $fillable = [
        'user_id',
        'log_date',
        'pregnancy_week',

        // Physical status
        'weight',
        'has_swelling',
        'swelling_locations',
        'has_shortness_of_breath',

        // Blood pressure
        'has_blood_pressure_device',
        'systolic_pressure',
        'diastolic_pressure',

        // Blood sugar
        'fasting_blood_sugar',
        'post_meal_blood_sugar',

        // Mental/Emotional
        'overall_mood',
        'has_anxiety',
        'anxiety_severity',
        'has_mood_swings',
        'mood_swings_severity',
        'has_depression_feelings',
        'depression_severity',

        'notes',
    ];

    protected function casts(): array
    {
        return [
            'log_date' => 'date',

            // Booleans
            'has_swelling' => 'boolean',
            'has_shortness_of_breath' => 'boolean',
            'has_blood_pressure_device' => 'boolean',
            'has_anxiety' => 'boolean',
            'has_mood_swings' => 'boolean',
            'has_depression_feelings' => 'boolean',

            // Decimals
            'weight' => 'decimal:2',
            'fasting_blood_sugar' => 'decimal:2',
            'post_meal_blood_sugar' => 'decimal:2',

            // JSON
            'swelling_locations' => 'array',
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
            'swelling_locations' => SwellingLocation::values(),
            'overall_mood' => MentalHealthStatus::values(),
            'severity' => SymptomSeverity::values(),
        ];
    }

    /**
     * Check for concerning blood pressure readings
     */
    public function hasHighBloodPressure(): bool
    {
        if ($this->systolic_pressure === null || $this->diastolic_pressure === null) {
            return false;
        }

        // High BP threshold during pregnancy: >140/90
        return $this->systolic_pressure >= 140 || $this->diastolic_pressure >= 90;
    }

    /**
     * Check for concerning blood sugar levels
     */
    public function hasHighBloodSugar(): bool
    {
        // Fasting glucose > 95 mg/dL or post-meal > 140 mg/dL
        return ($this->fasting_blood_sugar !== null && $this->fasting_blood_sugar > 95) ||
               ($this->post_meal_blood_sugar !== null && $this->post_meal_blood_sugar > 140);
    }
}
