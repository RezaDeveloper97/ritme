<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\MentalHealthStatus;
use App\Enums\SwellingLocation;
use App\Enums\SymptomSeverity;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StorePregnancyWeeklyLogRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $swellingLocationValues = SwellingLocation::values();
        $moodValues = MentalHealthStatus::values();
        $severityValues = SymptomSeverity::values();

        return [
            'log_date' => ['required', 'date', 'before_or_equal:today'],
            'pregnancy_week' => ['required', 'integer', 'min:1', 'max:42'],

            // Physical status
            'weight' => ['nullable', 'numeric', 'min:30', 'max:200'],
            'has_swelling' => ['nullable', 'boolean'],
            'swelling_locations' => ['nullable', 'array'],
            'swelling_locations.*' => ['required', Rule::in($swellingLocationValues)],
            'has_shortness_of_breath' => ['nullable', 'boolean'],

            // Blood pressure
            'has_blood_pressure_device' => ['nullable', 'boolean'],
            'systolic_pressure' => ['nullable', 'integer', 'min:60', 'max:250'],
            'diastolic_pressure' => ['nullable', 'integer', 'min:40', 'max:150'],

            // Blood sugar
            'fasting_blood_sugar' => ['nullable', 'numeric', 'min:40', 'max:400'],
            'post_meal_blood_sugar' => ['nullable', 'numeric', 'min:40', 'max:500'],

            // Mental/Emotional
            'overall_mood' => ['nullable', Rule::in($moodValues)],
            'has_anxiety' => ['nullable', 'boolean'],
            'anxiety_severity' => ['nullable', Rule::in($severityValues)],
            'has_mood_swings' => ['nullable', 'boolean'],
            'mood_swings_severity' => ['nullable', Rule::in($severityValues)],
            'has_depression_feelings' => ['nullable', 'boolean'],
            'depression_severity' => ['nullable', Rule::in($severityValues)],

            'notes' => ['nullable', 'string', 'max:2000'],
        ];
    }

    public function messages(): array
    {
        return [
            'log_date.required' => 'تاریخ الزامی است',
            'log_date.date' => 'فرمت تاریخ نامعتبر است',
            'log_date.before_or_equal' => 'تاریخ نمی‌تواند در آینده باشد',
            'pregnancy_week.required' => 'هفته بارداری الزامی است',
            'pregnancy_week.min' => 'هفته بارداری باید حداقل 1 باشد',
            'pregnancy_week.max' => 'هفته بارداری نمی‌تواند بیشتر از 42 باشد',
            'weight.min' => 'وزن باید حداقل 30 کیلوگرم باشد',
            'weight.max' => 'وزن نمی‌تواند بیشتر از 200 کیلوگرم باشد',
            'systolic_pressure.min' => 'فشار خون سیستولیک نامعتبر است',
            'systolic_pressure.max' => 'فشار خون سیستولیک نامعتبر است',
            'diastolic_pressure.min' => 'فشار خون دیاستولیک نامعتبر است',
            'diastolic_pressure.max' => 'فشار خون دیاستولیک نامعتبر است',
            'notes.max' => 'یادداشت نمی‌تواند بیشتر از 2000 کاراکتر باشد',
        ];
    }
}
