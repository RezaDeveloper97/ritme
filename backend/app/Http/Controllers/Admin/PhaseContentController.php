<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CycleSubphase;
use App\Http\Controllers\Controller;
use App\Models\PhaseContent;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

/**
 * Admin CRUD for the DB-driven Phase Details content. One editable row per
 * fine-grained cycle sub-phase (CycleSubphase), each holding nine bilingual
 * sections. Mirrors PregnancyWeekController.
 */
class PhaseContentController extends Controller
{
    /** The nine bilingual content sections stored per phase (field => Persian label). */
    public const FIELDS = [
        'symptom_prediction' => 'پیش‌بینی علائم',
        'vaginal_discharge' => 'ترشحات واژن',
        'fertility' => 'باروری',
        'hormonal_changes' => 'تغییرات هورمونی',
        'sex_tips' => 'نکات رابطه جنسی',
        'nutrition' => 'تغذیه',
        'exercise' => 'ورزش',
        'skin_care' => 'مراقبت از پوست',
        'sleep' => 'خواب',
    ];

    public function index(): View
    {
        $existing = PhaseContent::pluck('id', 'phase');
        $phases = collect(CycleSubphase::options())->map(fn (array $option) => $option + [
            'id' => $existing[$option['value']] ?? null,
        ]);

        // A row stored under a non-canonical key (seeded before the aliases were
        // collapsed) still needs a way in, or it would be unreachable here.
        $orphans = $existing->keys()->diff($phases->pluck('value'))->map(fn (string $phase) => [
            'value' => $phase,
            'label' => (CycleSubphase::labelFor($phase) ?? $phase).' (قدیمی)',
            'id' => $existing[$phase],
        ]);

        $phases = $phases->concat($orphans);

        return view('admin.phase-contents.index', compact('phases'));
    }

    public function create(Request $request): View
    {
        $phase = $request->query('phase');
        $phase = CycleSubphase::tryFrom((string) $phase)?->value;

        return view('admin.phase-contents.form', [
            'content' => new PhaseContent(['phase' => $phase]),
            'fields' => self::FIELDS,
            'phases' => CycleSubphase::options(),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        PhaseContent::create($this->validated($request));

        return redirect()->route('admin.phase-contents.index')->with('status', 'محتوای فاز ذخیره شد.');
    }

    public function edit(PhaseContent $phaseContent): View
    {
        return view('admin.phase-contents.form', [
            'content' => $phaseContent,
            'fields' => self::FIELDS,
            'phases' => CycleSubphase::options(),
        ]);
    }

    public function update(Request $request, PhaseContent $phaseContent): RedirectResponse
    {
        $phaseContent->update($this->validated($request, $phaseContent));

        return redirect()->route('admin.phase-contents.index')->with('status', 'محتوای فاز به‌روزرسانی شد.');
    }

    public function destroy(PhaseContent $phaseContent): RedirectResponse
    {
        $phaseContent->delete();

        return redirect()->route('admin.phase-contents.index')->with('status', 'محتوای فاز حذف شد.');
    }

    private function validated(Request $request, ?PhaseContent $content = null): array
    {
        $rules = [
            'phase' => ['required', 'string', Rule::in(CycleSubphase::values()), Rule::unique('phase_contents', 'phase')->ignore($content)],
        ];
        foreach (array_keys(self::FIELDS) as $field) {
            $rules[$field] = ['nullable', 'array'];
        }

        return $request->validate($rules);
    }
}
