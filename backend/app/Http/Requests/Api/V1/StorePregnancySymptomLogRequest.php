<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\SymptomSeverity;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StorePregnancySymptomLogRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $severityValues = SymptomSeverity::values();

        return [
            'log_date' => ['required', 'date', 'before_or_equal:today'],

            // General symptoms
            'has_nausea' => ['nullable', 'boolean'],
            'nausea_severity' => ['nullable', Rule::in($severityValues)],
            'has_vomiting' => ['nullable', 'boolean'],
            'vomiting_severity' => ['nullable', Rule::in($severityValues)],
            'has_fatigue' => ['nullable', 'boolean'],
            'fatigue_severity' => ['nullable', Rule::in($severityValues)],
            'has_headache' => ['nullable', 'boolean'],
            'headache_severity' => ['nullable', Rule::in($severityValues)],
            'has_dizziness' => ['nullable', 'boolean'],
            'dizziness_severity' => ['nullable', Rule::in($severityValues)],
            'has_breast_pain' => ['nullable', 'boolean'],
            'breast_pain_severity' => ['nullable', Rule::in($severityValues)],

            // Abdominal/Pelvic symptoms
            'has_lower_abdominal_pain' => ['nullable', 'boolean'],
            'lower_abdominal_pain_severity' => ['nullable', Rule::in($severityValues)],
            'has_cramping' => ['nullable', 'boolean'],
            'cramping_severity' => ['nullable', Rule::in($severityValues)],
            'has_back_pain' => ['nullable', 'boolean'],
            'back_pain_severity' => ['nullable', Rule::in($severityValues)],
            'has_pelvic_pressure' => ['nullable', 'boolean'],
            'pelvic_pressure_severity' => ['nullable', Rule::in($severityValues)],

            // Sensitive symptoms
            'has_spotting' => ['nullable', 'boolean'],
            'spotting_severity' => ['nullable', Rule::in($severityValues)],
            'has_bleeding' => ['nullable', 'boolean'],
            'bleeding_severity' => ['nullable', Rule::in($severityValues)],
            'has_fluid_leakage' => ['nullable', 'boolean'],
            'fluid_leakage_severity' => ['nullable', Rule::in($severityValues)],
            'has_severe_sudden_pain' => ['nullable', 'boolean'],
            'severe_sudden_pain_severity' => ['nullable', Rule::in($severityValues)],

            'notes' => ['nullable', 'string', 'max:2000'],
        ];
    }

    public function messages(): array
    {
        return [
            'log_date.required' => 'تاریخ الزامی است',
            'log_date.date' => 'فرمت تاریخ نامعتبر است',
            'log_date.before_or_equal' => 'تاریخ نمی‌تواند در آینده باشد',
            'notes.max' => 'یادداشت نمی‌تواند بیشتر از 2000 کاراکتر باشد',
        ];
    }
}
