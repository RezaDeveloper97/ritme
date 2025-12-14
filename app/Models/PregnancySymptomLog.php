<?php

namespace App\Models;

use App\Enums\SymptomSeverity;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="PregnancySymptomLog",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="log_date", type="string", format="date", example="2024-12-14"),
 *     @OA\Property(property="has_nausea", type="boolean", nullable=true),
 *     @OA\Property(property="nausea_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_vomiting", type="boolean", nullable=true),
 *     @OA\Property(property="vomiting_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_fatigue", type="boolean", nullable=true),
 *     @OA\Property(property="fatigue_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_headache", type="boolean", nullable=true),
 *     @OA\Property(property="headache_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_dizziness", type="boolean", nullable=true),
 *     @OA\Property(property="dizziness_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_breast_pain", type="boolean", nullable=true),
 *     @OA\Property(property="breast_pain_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_lower_abdominal_pain", type="boolean", nullable=true),
 *     @OA\Property(property="lower_abdominal_pain_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_cramping", type="boolean", nullable=true),
 *     @OA\Property(property="cramping_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_back_pain", type="boolean", nullable=true),
 *     @OA\Property(property="back_pain_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_pelvic_pressure", type="boolean", nullable=true),
 *     @OA\Property(property="pelvic_pressure_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_spotting", type="boolean", nullable=true),
 *     @OA\Property(property="spotting_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_bleeding", type="boolean", nullable=true),
 *     @OA\Property(property="bleeding_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_fluid_leakage", type="boolean", nullable=true),
 *     @OA\Property(property="fluid_leakage_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="has_severe_sudden_pain", type="boolean", nullable=true),
 *     @OA\Property(property="severe_sudden_pain_severity", type="string", nullable=true, enum={"mild","moderate","severe"}),
 *     @OA\Property(property="notes", type="string", nullable=true),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PregnancySymptomLog extends Model
{
    protected $fillable = [
        'user_id',
        'log_date',

        // General symptoms
        'has_nausea',
        'nausea_severity',
        'has_vomiting',
        'vomiting_severity',
        'has_fatigue',
        'fatigue_severity',
        'has_headache',
        'headache_severity',
        'has_dizziness',
        'dizziness_severity',
        'has_breast_pain',
        'breast_pain_severity',

        // Abdominal/Pelvic symptoms
        'has_lower_abdominal_pain',
        'lower_abdominal_pain_severity',
        'has_cramping',
        'cramping_severity',
        'has_back_pain',
        'back_pain_severity',
        'has_pelvic_pressure',
        'pelvic_pressure_severity',

        // Sensitive symptoms
        'has_spotting',
        'spotting_severity',
        'has_bleeding',
        'bleeding_severity',
        'has_fluid_leakage',
        'fluid_leakage_severity',
        'has_severe_sudden_pain',
        'severe_sudden_pain_severity',

        'notes',
    ];

    protected function casts(): array
    {
        return [
            'log_date' => 'date',

            // Booleans
            'has_nausea' => 'boolean',
            'has_vomiting' => 'boolean',
            'has_fatigue' => 'boolean',
            'has_headache' => 'boolean',
            'has_dizziness' => 'boolean',
            'has_breast_pain' => 'boolean',
            'has_lower_abdominal_pain' => 'boolean',
            'has_cramping' => 'boolean',
            'has_back_pain' => 'boolean',
            'has_pelvic_pressure' => 'boolean',
            'has_spotting' => 'boolean',
            'has_bleeding' => 'boolean',
            'has_fluid_leakage' => 'boolean',
            'has_severe_sudden_pain' => 'boolean',
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
            'severity' => SymptomSeverity::values(),
        ];
    }

    /**
     * Check if any critical/sensitive symptoms are present
     */
    public function hasCriticalSymptoms(): bool
    {
        return $this->has_spotting ||
               $this->has_bleeding ||
               $this->has_fluid_leakage ||
               $this->has_severe_sudden_pain;
    }

    /**
     * Get critical symptoms for alert generation
     */
    public function getCriticalSymptoms(): array
    {
        $critical = [];

        if ($this->has_spotting) {
            $critical['spotting'] = $this->spotting_severity;
        }
        if ($this->has_bleeding) {
            $critical['bleeding'] = $this->bleeding_severity;
        }
        if ($this->has_fluid_leakage) {
            $critical['fluid_leakage'] = $this->fluid_leakage_severity;
        }
        if ($this->has_severe_sudden_pain) {
            $critical['severe_sudden_pain'] = $this->severe_sudden_pain_severity;
        }

        return $critical;
    }
}
