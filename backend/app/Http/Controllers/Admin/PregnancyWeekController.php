<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\PregnancyWeeklyContent;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class PregnancyWeekController extends Controller
{
    /** The ten bilingual content modules stored per week. */
    public const FIELDS = [
        'fetal_development' => 'رشد جنین',
        'mother_body_changes' => 'تغییرات بدن مادر',
        'dos_and_donts' => 'باید‌ها و نبایدها',
        'care_plan' => 'برنامه مراقبت',
        'body_adaptation' => 'سازگاری بدن',
        'emotional_status' => 'وضعیت احساسی',
        'key_nutrition' => 'تغذیه کلیدی',
        'physical_activity' => 'فعالیت بدنی',
        'tests_and_checkups' => 'آزمایش‌ها و معاینات',
        'faq' => 'پرسش‌های متداول',
    ];

    public function index(): View
    {
        $existing = PregnancyWeeklyContent::pluck('id', 'week_number');
        $weeks = collect(range(1, 40))->map(fn ($w) => [
            'week' => $w,
            'id' => $existing[$w] ?? null,
        ]);

        return view('admin.pregnancy-weeks.index', compact('weeks'));
    }

    public function create(Request $request): View
    {
        return view('admin.pregnancy-weeks.form', [
            'week' => new PregnancyWeeklyContent(['week_number' => (int) $request->query('week', 1)]),
            'fields' => self::FIELDS,
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        PregnancyWeeklyContent::create($this->validated($request));

        return redirect()->route('admin.pregnancy-weeks.index')->with('status', 'محتوای هفته ذخیره شد.');
    }

    public function edit(PregnancyWeeklyContent $pregnancyWeek): View
    {
        return view('admin.pregnancy-weeks.form', [
            'week' => $pregnancyWeek,
            'fields' => self::FIELDS,
        ]);
    }

    public function update(Request $request, PregnancyWeeklyContent $pregnancyWeek): RedirectResponse
    {
        $pregnancyWeek->update($this->validated($request, $pregnancyWeek));

        return redirect()->route('admin.pregnancy-weeks.index')->with('status', 'محتوای هفته به‌روزرسانی شد.');
    }

    public function destroy(PregnancyWeeklyContent $pregnancyWeek): RedirectResponse
    {
        $pregnancyWeek->delete();

        return redirect()->route('admin.pregnancy-weeks.index')->with('status', 'محتوای هفته حذف شد.');
    }

    private function validated(Request $request, ?PregnancyWeeklyContent $week = null): array
    {
        $rules = [
            'week_number' => ['required', 'integer', 'min:1', 'max:42', Rule::unique('pregnancy_weekly_content', 'week_number')->ignore($week)],
        ];
        foreach (array_keys(self::FIELDS) as $field) {
            $rules[$field] = ['nullable', 'array'];
        }

        return $request->validate($rules);
    }
}
