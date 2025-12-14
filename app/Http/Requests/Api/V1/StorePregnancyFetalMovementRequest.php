<?php

namespace App\Http\Requests\Api\V1;

use App\Enums\FetalMovementStatus;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StorePregnancyFetalMovementRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $movementStatusValues = FetalMovementStatus::values();

        return [
            'log_date' => ['required', 'date', 'before_or_equal:today'],
            'pregnancy_week' => ['required', 'integer', 'min:1', 'max:42'],
            'movement_status' => ['required', Rule::in($movementStatusValues)],
            'movement_count' => ['nullable', 'integer', 'min:0', 'max:100'],
            'first_movement_time' => ['nullable', 'date_format:H:i'],
            'last_movement_time' => ['nullable', 'date_format:H:i'],
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
            'movement_status.required' => 'وضعیت حرکت جنین الزامی است',
            'movement_status.in' => 'وضعیت حرکت جنین نامعتبر است',
            'movement_count.min' => 'تعداد حرکات نمی‌تواند منفی باشد',
            'movement_count.max' => 'تعداد حرکات نامعتبر است',
            'first_movement_time.date_format' => 'فرمت زمان نامعتبر است (HH:MM)',
            'last_movement_time.date_format' => 'فرمت زمان نامعتبر است (HH:MM)',
            'notes.max' => 'یادداشت نمی‌تواند بیشتر از 2000 کاراکتر باشد',
        ];
    }
}
