<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\BloodType;
use App\Enums\PreExistingCondition;
use App\Enums\PregnancyAgeSource;
use App\Enums\RhFactor;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class UpdatePregnancyProfileRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $ageSourceValues = PregnancyAgeSource::values();
        $bloodTypeValues = BloodType::values();
        $rhFactorValues = RhFactor::values();
        $preExistingConditionValues = PreExistingCondition::values();

        return [
            // Age source and calculation (optional for update)
            'age_source' => ['nullable', Rule::in($ageSourceValues)],

            // LMP data
            'lmp_date' => ['nullable', 'date', 'before_or_equal:today'],

            // Ultrasound data
            'ultrasound_date' => ['nullable', 'date', 'before_or_equal:today'],
            'ultrasound_weeks' => ['nullable', 'integer', 'min:1', 'max:42'],
            'ultrasound_days' => ['nullable', 'integer', 'min:0', 'max:6'],

            // Manual entry data
            'manual_weeks' => ['nullable', 'integer', 'min:1', 'max:42'],
            'manual_days' => ['nullable', 'integer', 'min:0', 'max:6'],

            // Pregnancy history
            'has_miscarriage_history' => ['nullable', 'boolean'],
            'has_high_risk_history' => ['nullable', 'boolean'],

            // Pre-existing conditions
            'pre_existing_conditions' => ['nullable', 'array'],
            'pre_existing_conditions.*' => ['required', Rule::in($preExistingConditionValues)],

            // Blood information
            'blood_type' => ['nullable', Rule::in($bloodTypeValues)],
            'rh_factor' => ['nullable', Rule::in($rhFactorValues)],

            // Fetal movement
            'fetal_movement_felt' => ['nullable', 'boolean'],
            'first_fetal_movement_date' => ['nullable', 'date', 'before_or_equal:today'],
        ];
    }

    public function messages(): array
    {
        return [
            'age_source.in' => 'منبع محاسبه سن بارداری نامعتبر است',
            'lmp_date.date' => 'فرمت تاریخ نامعتبر است',
            'lmp_date.before_or_equal' => 'تاریخ نمی‌تواند در آینده باشد',
            'ultrasound_weeks.min' => 'هفته بارداری باید حداقل 1 باشد',
            'ultrasound_weeks.max' => 'هفته بارداری نمی‌تواند بیشتر از 42 باشد',
            'ultrasound_days.min' => 'روز بارداری باید بین 0 تا 6 باشد',
            'ultrasound_days.max' => 'روز بارداری باید بین 0 تا 6 باشد',
        ];
    }
}
