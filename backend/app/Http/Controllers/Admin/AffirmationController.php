<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CyclePhase;
use App\Http\Controllers\Controller;
use App\Models\Affirmation;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class AffirmationController extends Controller
{
    public function index(): View
    {
        $affirmations = Affirmation::orderBy('sort_order')->orderByDesc('id')->paginate(20);

        return view('admin.affirmations.index', compact('affirmations'));
    }

    public function create(): View
    {
        return view('admin.affirmations.form', [
            'affirmation' => new Affirmation(['is_active' => true]),
            'phases' => CyclePhase::options(),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        Affirmation::create($this->validated($request));

        return redirect()->route('admin.affirmations.index')->with('status', 'تأکید ایجاد شد.');
    }

    public function edit(Affirmation $affirmation): View
    {
        return view('admin.affirmations.form', [
            'affirmation' => $affirmation,
            'phases' => CyclePhase::options(),
        ]);
    }

    public function update(Request $request, Affirmation $affirmation): RedirectResponse
    {
        $affirmation->update($this->validated($request));

        return redirect()->route('admin.affirmations.index')->with('status', 'تأکید به‌روزرسانی شد.');
    }

    public function destroy(Affirmation $affirmation): RedirectResponse
    {
        $affirmation->delete();

        return back()->with('status', 'تأکید حذف شد.');
    }

    public function toggle(Affirmation $affirmation): RedirectResponse
    {
        $affirmation->update(['is_active' => ! $affirmation->is_active]);

        return back()->with('status', 'وضعیت تغییر کرد.');
    }

    private function validated(Request $request): array
    {
        $data = $request->validate([
            'text' => ['required', 'array'],
            'text.fa' => ['required', 'string'],
            'text.en' => ['nullable', 'string'],
            'cycle_phase' => ['nullable', Rule::in(CyclePhase::values())],
            'sort_order' => ['nullable', 'integer'],
        ]);

        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;

        return $data;
    }
}
