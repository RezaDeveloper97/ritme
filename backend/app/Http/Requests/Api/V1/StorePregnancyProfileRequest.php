<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\BloodType;
use App\Enums\ConfidenceLevel;
use App\Enums\PreExistingCondition;
use App\Enums\PregnancyAgeSource;
use App\Enums\RhFactor;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StorePregnancyProfileRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $ageSourceValues = PregnancyAgeSource::values();
        $confidenceValues = ConfidenceLevel::values();
        $bloodTypeValues = BloodType::values();
        $rhFactorValues = RhFactor::values();
        $preExistingConditionValues = PreExistingCondition::values();

        return [
            // Age source and calculation
            'age_source' => ['required', Rule::in($ageSourceValues)],

            // LMP data (required if age_source is 'lmp')
            'lmp_date' => ['nullable', 'required_if:age_source,lmp', 'date', 'before_or_equal:today'],

            // Ultrasound data (required if age_source is 'ultrasound')
            'ultrasound_date' => ['nullable', 'required_if:age_source,ultrasound', 'date', 'before_or_equal:today'],
            'ultrasound_weeks' => ['nullable', 'required_if:age_source,ultrasound', 'integer', 'min:1', 'max:42'],
            'ultrasound_days' => ['nullable', 'required_if:age_source,ultrasound', 'integer', 'min:0', 'max:6'],

            // Manual entry data (required if age_source is 'manual')
            'manual_weeks' => ['nullable', 'required_if:age_source,manual', 'integer', 'min:1', 'max:42'],
            'manual_days' => ['nullable', 'required_if:age_source,manual', 'integer', 'min:0', 'max:6'],

            // Pregnancy history (optional)
            'has_miscarriage_history' => ['nullable', 'boolean'],
            'has_high_risk_history' => ['nullable', 'boolean'],

            // Pre-existing conditions (optional)
            'pre_existing_conditions' => ['nullable', 'array'],
            'pre_existing_conditions.*' => ['required', Rule::in($preExistingConditionValues)],

            // Blood information (optional)
            'blood_type' => ['nullable', Rule::in($bloodTypeValues)],
            'rh_factor' => ['nullable', Rule::in($rhFactorValues)],
        ];
    }

    public function messages(): array
    {
        return [
            'age_source.required' => 'منبع محاسبه سن بارداری الزامی است',
            'age_source.in' => 'منبع محاسبه سن بارداری نامعتبر است',
            'lmp_date.required_if' => 'تاریخ آخرین پریود برای این روش محاسبه الزامی است',
            'lmp_date.date' => 'فرمت تاریخ نامعتبر است',
            'lmp_date.before_or_equal' => 'تاریخ نمی‌تواند در آینده باشد',
            'ultrasound_date.required_if' => 'تاریخ سونوگرافی برای این روش محاسبه الزامی است',
            'ultrasound_weeks.required_if' => 'هفته بارداری اعلام‌شده در سونو الزامی است',
            'ultrasound_weeks.min' => 'هفته بارداری باید حداقل 1 باشد',
            'ultrasound_weeks.max' => 'هفته بارداری نمی‌تواند بیشتر از 42 باشد',
            'ultrasound_days.required_if' => 'روز بارداری اعلام‌شده در سونو الزامی است',
            'ultrasound_days.min' => 'روز بارداری باید بین 0 تا 6 باشد',
            'ultrasound_days.max' => 'روز بارداری باید بین 0 تا 6 باشد',
            'manual_weeks.required_if' => 'هفته بارداری برای ورود دستی الزامی است',
            'manual_days.required_if' => 'روز بارداری برای ورود دستی الزامی است',
        ];
    }
}
