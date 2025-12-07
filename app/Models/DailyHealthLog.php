<?php

namespace App\Models;

use App\Enums\Amount;
use App\Enums\BloodColor;
use App\Enums\DischargeTexture;
use App\Enums\Intensity;
use App\Enums\Mood;
use App\Enums\PainIntensity;
use App\Enums\SleepDuration;
use App\Enums\SleepQuality;
use App\Enums\Smell;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class DailyHealthLog extends Model
{
    protected $fillable = [
        'user_id',
        'log_date',

        // 1. قاعدگی و خون‌ریزی
        'bleeding_intensity',
        'blood_color',
        'has_clots',
        'spotting',
        'bleeding_smell',

        // 2. علائم - دردها
        'headache_intensity',
        'stomach_ache_intensity',
        'pelvic_pain_intensity',
        'breast_pain_intensity',
        'back_pain_intensity',
        'ovarian_pain_intensity',

        // 2. علائم - گوارشی
        'nausea_intensity',
        'bloating_intensity',
        'diarrhea',
        'constipation',
        'appetite_change',
        'food_craving',

        // 2. علائم - سینه و تناسلی
        'breast_sensitivity_intensity',
        'vaginal_dryness',
        'vaginal_burning',
        'vaginal_itching',
        'vaginal_smell_change',
        'urination_change',
        'urination_burning_intensity',

        // 2. علائم - پوست و مو
        'acne',
        'oily_skin',
        'hair_loss',
        'swelling',

        // 2. علائم - انرژی و عمومی
        'fatigue',
        'dizziness',
        'hot_flashes',
        'chills',

        // 3. مود و احساسات
        'moods',

        // 4. خواب
        'sleep_duration',
        'sleep_quality',

        // 5. فعالیت جنسی
        'sexual_activity_level',

        // 6. علائم حیاتی
        'weight',
        'basal_body_temperature',

        // 7. ترشحات واژن
        'discharge_color',
        'discharge_texture',
        'discharge_amount',
        'discharge_smell',
        'discharge_itching',
        'discharge_burning',

        // 8. گوارش و ادراری
        'frequent_urination',

        // 9. دارو و مکمل
        'medications',

        // 10. یادداشت
        'notes',
    ];

    protected function casts(): array
    {
        return [
            'log_date' => 'date',

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

            // JSON
            'moods' => 'array',
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
            'sexual_activity_level' => Intensity::values(),
            'discharge_texture' => DischargeTexture::values(),
            'discharge_amount' => Amount::values(),
            'discharge_smell' => Smell::values(),
        ];
    }
}
