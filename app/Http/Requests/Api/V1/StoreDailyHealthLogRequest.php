<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\Amount;
use App\Enums\BloodColor;
use App\Enums\DischargeTexture;
use App\Enums\Intensity;
use App\Enums\Mood;
use App\Enums\PainIntensity;
use App\Enums\SleepDuration;
use App\Enums\SleepQuality;
use App\Enums\Smell;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StoreDailyHealthLogRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $intensityValues = Intensity::values();
        $painIntensityValues = PainIntensity::values();
        $amountValues = Amount::values();
        $bloodColorValues = BloodColor::values();
        $smellValues = Smell::values();
        $sleepDurationValues = SleepDuration::values();
        $sleepQualityValues = SleepQuality::values();
        $moodValues = Mood::values();
        $dischargeTextureValues = DischargeTexture::values();

        return [
            'log_date' => ['required', 'date', 'before_or_equal:today'],

            // =========================================
            // 1. قاعدگی و خون‌ریزی
            // =========================================
            'bleeding_intensity' => ['nullable', Rule::in($intensityValues)],
            'blood_color' => ['nullable', Rule::in($bloodColorValues)],
            'has_clots' => ['nullable', 'boolean'],
            'spotting' => ['nullable', 'boolean'],
            'bleeding_smell' => ['nullable', Rule::in($smellValues)],

            // =========================================
            // 2. علائم - دردها
            // =========================================
            'headache_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'stomach_ache_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'pelvic_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'breast_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'back_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'ovarian_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],

            // =========================================
            // 2. علائم - گوارشی
            // =========================================
            'nausea_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'bloating_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'diarrhea' => ['nullable', 'boolean'],
            'constipation' => ['nullable', 'boolean'],
            'appetite_change' => ['nullable', Rule::in(['loss', 'gain', 'normal'])],
            'food_craving' => ['nullable', 'boolean'],

            // =========================================
            // 2. علائم - سینه و تناسلی
            // =========================================
            'breast_sensitivity_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'vaginal_dryness' => ['nullable', 'boolean'],
            'vaginal_burning' => ['nullable', 'boolean'],
            'vaginal_itching' => ['nullable', 'boolean'],
            'vaginal_smell_change' => ['nullable', 'boolean'],
            'urination_change' => ['nullable', Rule::in(['increase', 'decrease', 'normal'])],
            'urination_burning_intensity' => ['nullable', Rule::in($painIntensityValues)],

            // =========================================
            // 2. علائم - پوست و مو
            // =========================================
            'acne' => ['nullable', 'boolean'],
            'oily_skin' => ['nullable', 'boolean'],
            'hair_loss' => ['nullable', 'boolean'],
            'swelling' => ['nullable', 'boolean'],

            // =========================================
            // 2. علائم - انرژی و عمومی
            // =========================================
            'fatigue' => ['nullable', 'boolean'],
            'dizziness' => ['nullable', 'boolean'],
            'hot_flashes' => ['nullable', 'boolean'],
            'chills' => ['nullable', 'boolean'],

            // =========================================
            // 3. مود و احساسات
            // =========================================
            'moods' => ['nullable', 'array'],
            'moods.*' => ['required', Rule::in($moodValues)],

            // =========================================
            // 4. خواب
            // =========================================
            'sleep_duration' => ['nullable', Rule::in($sleepDurationValues)],
            'sleep_quality' => ['nullable', Rule::in($sleepQualityValues)],

            // =========================================
            // 5. فعالیت جنسی
            // =========================================
            'sexual_activity_level' => ['nullable', Rule::in($intensityValues)],

            // =========================================
            // 6. علائم حیاتی
            // =========================================
            'weight' => ['nullable', 'numeric', 'min:20', 'max:300'],
            'basal_body_temperature' => ['nullable', 'numeric', 'min:35', 'max:42'],

            // =========================================
            // 7. ترشحات واژن
            // =========================================
            'discharge_color' => ['nullable', 'string', 'max:50'],
            'discharge_texture' => ['nullable', Rule::in($dischargeTextureValues)],
            'discharge_amount' => ['nullable', Rule::in($amountValues)],
            'discharge_smell' => ['nullable', Rule::in($smellValues)],
            'discharge_itching' => ['nullable', 'boolean'],
            'discharge_burning' => ['nullable', 'boolean'],

            // =========================================
            // 8. گوارش و ادراری
            // =========================================
            'frequent_urination' => ['nullable', 'boolean'],

            // =========================================
            // 9. دارو و مکمل‌ها
            // =========================================
            'medications' => ['nullable', 'array'],
            'medications.painkillers' => ['nullable', 'string', 'max:255'],
            'medications.hormonal_pills' => ['nullable', 'string', 'max:255'],
            'medications.antibiotics' => ['nullable', 'string', 'max:255'],
            'medications.supplements' => ['nullable', 'string', 'max:255'],

            // =========================================
            // 10. یادداشت
            // =========================================
            'notes' => ['nullable', 'string', 'max:2000'],
        ];
    }

    public function messages(): array
    {
        return [
            'log_date.required' => 'تاریخ الزامی است',
            'log_date.date' => 'فرمت تاریخ نامعتبر است',
            'log_date.before_or_equal' => 'تاریخ نمی‌تواند در آینده باشد',
            'weight.min' => 'وزن باید حداقل ۲۰ کیلوگرم باشد',
            'weight.max' => 'وزن نمی‌تواند بیشتر از ۳۰۰ کیلوگرم باشد',
            'basal_body_temperature.min' => 'دمای بدن باید حداقل ۳۵ درجه باشد',
            'basal_body_temperature.max' => 'دمای بدن نمی‌تواند بیشتر از ۴۲ درجه باشد',
            'notes.max' => 'یادداشت نمی‌تواند بیشتر از ۲۰۰۰ کاراکتر باشد',
        ];
    }
}
