<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\Amount;
use App\Enums\BloodColor;
use App\Enums\DischargeTexture;
use App\Enums\EnergyLevel;
use App\Enums\Intensity;
use App\Enums\Mood;
use App\Enums\PainIntensity;
use App\Enums\SexualActivity;
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
        $sexualActivityValues = SexualActivity::values();

        return [
            'log_date' => ['required', 'date', 'before_or_equal:today'],

            // =========================================
            // 1. Menstruation & Bleeding
            // =========================================
            'bleeding_intensity' => ['nullable', Rule::in($intensityValues)],
            'blood_color' => ['nullable', Rule::in($bloodColorValues)],
            'has_clots' => ['nullable', 'boolean'],
            'spotting' => ['nullable', 'boolean'],
            'bleeding_smell' => ['nullable', Rule::in($smellValues)],

            // =========================================
            // 2. Symptoms - Pains
            // =========================================
            'headache_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'stomach_ache_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'pelvic_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'breast_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'back_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'ovarian_pain_intensity' => ['nullable', Rule::in($painIntensityValues)],

            // =========================================
            // 2. Symptoms - Digestive
            // =========================================
            'nausea_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'bloating_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'diarrhea' => ['nullable', 'boolean'],
            'constipation' => ['nullable', 'boolean'],
            'appetite_change' => ['nullable', Rule::in(['loss', 'gain', 'normal'])],
            'food_craving' => ['nullable', 'boolean'],

            // =========================================
            // 2. Symptoms - Breast & Genital
            // =========================================
            'breast_sensitivity_intensity' => ['nullable', Rule::in($painIntensityValues)],
            'vaginal_dryness' => ['nullable', 'boolean'],
            'vaginal_burning' => ['nullable', 'boolean'],
            'vaginal_itching' => ['nullable', 'boolean'],
            'vaginal_smell_change' => ['nullable', 'boolean'],
            'urination_change' => ['nullable', Rule::in(['increase', 'decrease', 'normal'])],
            'urination_burning_intensity' => ['nullable', Rule::in($painIntensityValues)],

            // =========================================
            // 2. Symptoms - Skin & Hair
            // =========================================
            'acne' => ['nullable', 'boolean'],
            'oily_skin' => ['nullable', 'boolean'],
            'hair_loss' => ['nullable', 'boolean'],
            'swelling' => ['nullable', 'boolean'],

            // =========================================
            // 2. Symptoms - Energy & General
            // =========================================
            'fatigue' => ['nullable', 'boolean'],
            'dizziness' => ['nullable', 'boolean'],
            'hot_flashes' => ['nullable', 'boolean'],
            'chills' => ['nullable', 'boolean'],

            // =========================================
            // 3. Mood & Emotions
            // =========================================
            'moods' => ['nullable', 'array'],
            'moods.*' => ['required', Rule::in($moodValues)],

            // =========================================
            // 4. Sleep
            // =========================================
            'sleep_duration' => ['nullable', Rule::in($sleepDurationValues)],
            'sleep_quality' => ['nullable', Rule::in($sleepQualityValues)],

            // =========================================
            // 5. Sexual Activity
            // =========================================
            'sexual_activities' => ['nullable', 'array'],
            'sexual_activities.*' => ['required', Rule::in($sexualActivityValues)],

            // =========================================
            // 6. Vital Signs
            // =========================================
            'weight' => ['nullable', 'numeric', 'min:20', 'max:300'],
            'basal_body_temperature' => ['nullable', 'numeric', 'min:35', 'max:42'],
            'heart_rate' => ['nullable', 'integer', 'min:30', 'max:250'],
            'systolic_pressure' => ['nullable', 'integer', 'min:50', 'max:300'],
            'diastolic_pressure' => ['nullable', 'integer', 'min:30', 'max:200'],
            'blood_sugar' => ['nullable', 'numeric', 'min:20', 'max:600'],
            'energy_level' => ['nullable', Rule::in(EnergyLevel::values())],

            // =========================================
            // 7. Vaginal Discharge
            // =========================================
            'discharge_color' => ['nullable', 'string', 'max:50'],
            'discharge_texture' => ['nullable', Rule::in($dischargeTextureValues)],
            'discharge_amount' => ['nullable', Rule::in($amountValues)],
            'discharge_smell' => ['nullable', Rule::in($smellValues)],
            'discharge_itching' => ['nullable', 'boolean'],
            'discharge_burning' => ['nullable', 'boolean'],

            // =========================================
            // 8. Digestive & Urinary
            // =========================================
            'frequent_urination' => ['nullable', 'boolean'],

            // =========================================
            // 9. Medications & Supplements
            // =========================================
            'medications' => ['nullable', 'array'],
            'medications.painkillers' => ['nullable', 'string', 'max:255'],
            'medications.hormonal_pills' => ['nullable', 'string', 'max:255'],
            'medications.antibiotics' => ['nullable', 'string', 'max:255'],
            'medications.supplements' => ['nullable', 'string', 'max:255'],

            // =========================================
            // 10. Notes
            // =========================================
            'notes' => ['nullable', 'string', 'max:2000'],
        ];
    }

    public function messages(): array
    {
        return [
            'log_date.required' => 'Date is required',
            'log_date.date' => 'Invalid date format',
            'log_date.before_or_equal' => 'Date cannot be in the future',
            'weight.min' => 'Weight must be at least 20 kg',
            'weight.max' => 'Weight cannot exceed 300 kg',
            'basal_body_temperature.min' => 'Temperature must be at least 35°C',
            'basal_body_temperature.max' => 'Temperature cannot exceed 42°C',
            'heart_rate.min' => 'Heart rate must be at least 30 bpm',
            'heart_rate.max' => 'Heart rate cannot exceed 250 bpm',
            'systolic_pressure.min' => 'Systolic pressure must be at least 50 mmHg',
            'systolic_pressure.max' => 'Systolic pressure cannot exceed 300 mmHg',
            'diastolic_pressure.min' => 'Diastolic pressure must be at least 30 mmHg',
            'diastolic_pressure.max' => 'Diastolic pressure cannot exceed 200 mmHg',
            'blood_sugar.min' => 'Blood sugar must be at least 20 mg/dl',
            'blood_sugar.max' => 'Blood sugar cannot exceed 600 mg/dl',
            'notes.max' => 'Notes cannot exceed 2000 characters',
        ];
    }
}
