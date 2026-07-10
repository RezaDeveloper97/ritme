<?php

namespace App\Models;

use App\Enums\Amount;
use App\Enums\BloodColor;
use App\Enums\DischargeTexture;
use App\Enums\Intensity;
use App\Enums\Mood;
use App\Enums\PainIntensity;
use App\Enums\SexualActivity;
use App\Enums\SleepDuration;
use App\Enums\SleepQuality;
use App\Enums\Smell;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="DailyHealthLog",
 *     type="object",
 *
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="user_id", type="integer", example=1),
 *     @OA\Property(property="log_date", type="string", format="date", example="2024-12-07"),
 *     @OA\Property(property="bleeding_intensity", type="string", nullable=true, enum={"low","medium","high","very_high"}),
 *     @OA\Property(property="blood_color", type="string", nullable=true, enum={"bright_red","red","dark_red","brown"}),
 *     @OA\Property(property="has_clots", type="boolean", nullable=true),
 *     @OA\Property(property="spotting", type="boolean", nullable=true),
 *     @OA\Property(property="bleeding_smell", type="string", nullable=true, enum={"normal","slightly_unusual","strong_unpleasant"}),
 *     @OA\Property(property="headache_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="stomach_ache_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="pelvic_pain_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="breast_pain_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="back_pain_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="ovarian_pain_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="nausea_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="bloating_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="diarrhea", type="boolean", nullable=true),
 *     @OA\Property(property="constipation", type="boolean", nullable=true),
 *     @OA\Property(property="appetite_change", type="string", nullable=true, enum={"loss","gain","normal"}),
 *     @OA\Property(property="food_craving", type="boolean", nullable=true),
 *     @OA\Property(property="breast_sensitivity_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="vaginal_dryness", type="boolean", nullable=true),
 *     @OA\Property(property="vaginal_burning", type="boolean", nullable=true),
 *     @OA\Property(property="vaginal_itching", type="boolean", nullable=true),
 *     @OA\Property(property="vaginal_smell_change", type="boolean", nullable=true),
 *     @OA\Property(property="urination_change", type="string", nullable=true, enum={"increase","decrease","normal"}),
 *     @OA\Property(property="urination_burning_intensity", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="acne", type="boolean", nullable=true),
 *     @OA\Property(property="oily_skin", type="boolean", nullable=true),
 *     @OA\Property(property="hair_loss", type="boolean", nullable=true),
 *     @OA\Property(property="swelling", type="boolean", nullable=true),
 *     @OA\Property(property="fatigue", type="boolean", nullable=true),
 *     @OA\Property(property="dizziness", type="boolean", nullable=true),
 *     @OA\Property(property="hot_flashes", type="boolean", nullable=true),
 *     @OA\Property(property="chills", type="boolean", nullable=true),
 *     @OA\Property(property="moods", type="array", nullable=true, @OA\Items(type="string", enum={"happy","calm","angry","anxious","sad","frustrated","sensitive","bored"})),
 *     @OA\Property(property="sleep_duration", type="string", nullable=true, enum={"0_3","3_6","6_9","9_plus"}),
 *     @OA\Property(property="sleep_quality", type="string", nullable=true, enum={"good","medium","bad"}),
 *     @OA\Property(property="sexual_activities", type="array", nullable=true, @OA\Items(type="string", enum={"high_desire","protected_intercourse","unprotected_intercourse","no_desire","dryness","burning","pain_during_intercourse","bleeding_after_intercourse","lubricant_use"})),
 *     @OA\Property(property="weight", type="number", format="float", nullable=true, example=65.5),
 *     @OA\Property(property="basal_body_temperature", type="number", format="float", nullable=true, example=36.5),
 *     @OA\Property(property="heart_rate", type="integer", nullable=true, example=72, description="Resting heart rate (bpm)"),
 *     @OA\Property(property="systolic_pressure", type="integer", nullable=true, example=120, description="Systolic blood pressure (mmHg)"),
 *     @OA\Property(property="diastolic_pressure", type="integer", nullable=true, example=80, description="Diastolic blood pressure (mmHg)"),
 *     @OA\Property(property="blood_sugar", type="number", format="float", nullable=true, example=95.0, description="Blood sugar (mg/dl)"),
 *     @OA\Property(property="energy_level", type="string", nullable=true, enum={"very_low","low","medium","high","very_high"}),
 *     @OA\Property(property="discharge_color", type="string", nullable=true),
 *     @OA\Property(property="discharge_texture", type="string", nullable=true, enum={"watery","creamy","egg_white","thick"}),
 *     @OA\Property(property="discharge_amount", type="string", nullable=true, enum={"low","medium","high"}),
 *     @OA\Property(property="discharge_smell", type="string", nullable=true, enum={"normal","slightly_unusual","strong_unpleasant"}),
 *     @OA\Property(property="discharge_itching", type="boolean", nullable=true),
 *     @OA\Property(property="discharge_burning", type="boolean", nullable=true),
 *     @OA\Property(property="frequent_urination", type="boolean", nullable=true),
 *     @OA\Property(property="medications", type="object", nullable=true,
 *         @OA\Property(property="painkillers", type="string"),
 *         @OA\Property(property="hormonal_pills", type="string"),
 *         @OA\Property(property="antibiotics", type="string"),
 *         @OA\Property(property="supplements", type="string")
 *     ),
 *     @OA\Property(property="notes", type="string", nullable=true),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class DailyHealthLog extends Model
{
    protected $fillable = [
        'user_id',
        'log_date',

        // 1. Menstruation & Bleeding
        'bleeding_intensity',
        'blood_color',
        'has_clots',
        'spotting',
        'bleeding_smell',

        // 2. Symptoms - Pains
        'headache_intensity',
        'stomach_ache_intensity',
        'pelvic_pain_intensity',
        'breast_pain_intensity',
        'back_pain_intensity',
        'ovarian_pain_intensity',

        // 2. Symptoms - Digestive
        'nausea_intensity',
        'bloating_intensity',
        'diarrhea',
        'constipation',
        'appetite_change',
        'food_craving',

        // 2. Symptoms - Breast & Genital
        'breast_sensitivity_intensity',
        'vaginal_dryness',
        'vaginal_burning',
        'vaginal_itching',
        'vaginal_smell_change',
        'urination_change',
        'urination_burning_intensity',

        // 2. Symptoms - Skin & Hair
        'acne',
        'oily_skin',
        'hair_loss',
        'swelling',

        // 2. Symptoms - Energy & General
        'fatigue',
        'dizziness',
        'hot_flashes',
        'chills',

        // 3. Mood & Emotions
        'moods',

        // 4. Sleep
        'sleep_duration',
        'sleep_quality',

        // 5. Sexual Activity
        'sexual_activities',

        // 6. Vital Signs
        'weight',
        'basal_body_temperature',
        'heart_rate',
        'systolic_pressure',
        'diastolic_pressure',
        'blood_sugar',

        // Energy
        'energy_level',

        // 7. Vaginal Discharge
        'discharge_color',
        'discharge_texture',
        'discharge_amount',
        'discharge_smell',
        'discharge_itching',
        'discharge_burning',

        // 8. Digestive & Urinary
        'frequent_urination',

        // 9. Medications & Supplements
        'medications',

        // 10. Notes
        'notes',
    ];

    protected function casts(): array
    {
        return [
            // date:Y-m-d (not plain 'date') so repeated same-day upserts match the
            // existing row on every DB — a plain 'date' cast serializes with a time
            // component that SQLite stores verbatim, making updateOrCreate re-insert
            // and hit the unique(user_id, log_date) constraint. It also returns clean
            // Y-m-d strings the frontend keys its per-date cache on.
            'log_date' => 'date:Y-m-d',

            // Booleans
            'has_clots' => 'boolean',
            'spotting' => 'boolean',
            'diarrhea' => 'boolean',
            'constipation' => 'boolean',
            'food_craving' => 'boolean',
            'vaginal_dryness' => 'boolean',
            'vaginal_burning' => 'boolean',
            'vaginal_itching' => 'boolean',
            'vaginal_smell_change' => 'boolean',
            'acne' => 'boolean',
            'oily_skin' => 'boolean',
            'hair_loss' => 'boolean',
            'swelling' => 'boolean',
            'fatigue' => 'boolean',
            'dizziness' => 'boolean',
            'hot_flashes' => 'boolean',
            'chills' => 'boolean',
            'discharge_itching' => 'boolean',
            'discharge_burning' => 'boolean',
            'frequent_urination' => 'boolean',

            // Decimals
            'weight' => 'decimal:2',
            'basal_body_temperature' => 'decimal:2',
            'blood_sugar' => 'decimal:1',

            // Integers
            'heart_rate' => 'integer',
            'systolic_pressure' => 'integer',
            'diastolic_pressure' => 'integer',

            // JSON
            'moods' => 'array',
            'sexual_activities' => 'array',
            'medications' => 'array',
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
            'bleeding_intensity' => Intensity::values(),
            'blood_color' => BloodColor::values(),
            'bleeding_smell' => Smell::values(),
            'pain_intensity' => PainIntensity::values(),
            'nausea_intensity' => PainIntensity::values(),
            'bloating_intensity' => PainIntensity::values(),
            'appetite_change' => ['loss', 'gain', 'normal'],
            'urination_change' => ['increase', 'decrease', 'normal'],
            'moods' => Mood::values(),
            'sleep_duration' => SleepDuration::values(),
            'sleep_quality' => SleepQuality::values(),
            'sexual_activities' => SexualActivity::values(),
            'discharge_texture' => DischargeTexture::values(),
            'discharge_amount' => Amount::values(),
            'discharge_smell' => Smell::values(),
        ];
    }
}
