<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CyclePhase;
use App\Http\Controllers\Controller;
use App\Models\Challenge;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class ChallengeController extends Controller
{
    private const DIFFICULTIES = ['easy', 'medium', 'hard'];

    public function index(): View
    {
        $challenges = Challenge::orderBy('sort_order')->orderByDesc('id')->paginate(20);

        return view('admin.challenges.index', compact('challenges'));
    }

    public function create(): View
    {
        return view('admin.challenges.form', [
            'challenge' => new Challenge(['is_active' => true]),
            'phases' => CyclePhase::options(),
            'difficulties' => self::DIFFICULTIES,
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        Challenge::create($this->validated($request));

        return redirect()->route('admin.challenges.index')->with('status', 'چالش ایجاد شد.');
    }

    public function edit(Challenge $challenge): View
    {
        return view('admin.challenges.form', [
            'challenge' => $challenge,
            'phases' => CyclePhase::options(),
            'difficulties' => self::DIFFICULTIES,
        ]);
    }

    public function update(Request $request, Challenge $challenge): RedirectResponse
    {
        $challenge->update($this->validated($request));

        return redirect()->route('admin.challenges.index')->with('status', 'چالش به‌روزرسانی شد.');
    }

    public function destroy(Challenge $challenge): RedirectResponse
    {
        $challenge->delete();

        return back()->with('status', 'چالش حذف شد.');
    }

    public function toggle(Challenge $challenge): RedirectResponse
    {
        $challenge->update(['is_active' => ! $challenge->is_active]);

        return back()->with('status', 'وضعیت تغییر کرد.');
    }

    private function validated(Request $request): array
    {
        $data = $request->validate([
            'title' => ['required', 'array'],
            'title.fa' => ['required', 'string'],
            'title.en' => ['nullable', 'string'],
            'description' => ['nullable', 'array'],
            'cycle_phase' => ['nullable', Rule::in(CyclePhase::values())],
            'category' => ['nullable', 'string', 'max:255'],
            'difficulty' => ['nullable', Rule::in(self::DIFFICULTIES)],
            'sort_order' => ['nullable', 'integer'],
        ]);

        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;

        return $data;
    }
}
