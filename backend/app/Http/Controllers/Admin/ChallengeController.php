<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Challenge;
use Illuminate\Contracts\Pagination\LengthAwarePaginator;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

/**
 * Challenge CRUD for the admin panel.
 *
 * Challenges are targeted by *cycle day* — a range inside 1..{@see
 * Challenge::MAX_CYCLE_DAY} — which is what {@see
 * \App\Services\Challenges\DailyChallengeService} matches against the user's
 * day when it picks «چالش امروز». Leaving the range empty means "any day".
 */
class ChallengeController extends Controller
{
    private const PER_PAGE = 20;

    public function index(Request $request): View
    {
        return view('admin.challenges.index', [
            'challenges' => $this->filtered($request),
            'filters' => [
                'q' => $request->string('q')->toString(),
                'status' => $request->string('status')->toString(),
                'cycle_day' => $request->input('cycle_day'),
            ],
            'maxCycleDay' => Challenge::MAX_CYCLE_DAY,
        ]);
    }

    public function create(): View
    {
        return view('admin.challenges.form', $this->formData(new Challenge(['is_active' => true])));
    }

    public function store(Request $request): RedirectResponse
    {
        Challenge::create($this->validated($request));

        return redirect()->route('admin.challenges.index')->with('status', 'چالش ایجاد شد.');
    }

    public function edit(Challenge $challenge): View
    {
        return view('admin.challenges.form', $this->formData($challenge));
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

    /**
     * The list, narrowed by the filter bar. `cycle_day` answers the question an
     * admin actually asks — "what would a user on day N be offered?" — so it
     * matches untargeted challenges too, exactly like the picker does.
     *
     * @return LengthAwarePaginator<int, Challenge>
     */
    private function filtered(Request $request): LengthAwarePaginator
    {
        $day = $request->filled('cycle_day') ? (int) $request->input('cycle_day') : null;

        return Challenge::query()
            ->when($request->filled('q'), function (Builder $query) use ($request) {
                $term = '%'.$request->string('q')->toString().'%';
                $query->where(fn (Builder $q) => $q
                    ->where('title', 'like', $term)
                    ->orWhere('description', 'like', $term)
                    ->orWhere('category', 'like', $term));
            })
            ->when($request->input('status') === 'active', fn (Builder $q) => $q->where('is_active', true))
            ->when($request->input('status') === 'inactive', fn (Builder $q) => $q->where('is_active', false))
            ->when(
                $day !== null && $day >= 1 && $day <= Challenge::MAX_CYCLE_DAY,
                fn (Builder $q) => $q->forCycleDay($day),
            )
            ->orderBy('sort_order')
            ->orderByDesc('id')
            ->paginate(self::PER_PAGE)
            ->withQueryString();
    }

    /**
     * @return array<string, mixed>
     */
    private function formData(Challenge $challenge): array
    {
        return [
            'challenge' => $challenge,
            'maxCycleDay' => Challenge::MAX_CYCLE_DAY,
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function validated(Request $request): array
    {
        $data = $request->validate([
            'title' => ['required', 'array'],
            'title.fa' => ['required', 'string'],
            'title.en' => ['nullable', 'string'],
            'description' => ['nullable', 'array'],
            'cycle_day_from' => ['nullable', 'integer', 'min:1', 'max:'.Challenge::MAX_CYCLE_DAY],
            // A range that ends before it starts would silently match nothing.
            'cycle_day_to' => ['nullable', 'integer', 'min:1', 'max:'.Challenge::MAX_CYCLE_DAY, 'gte:cycle_day_from'],
            'category' => ['nullable', 'string', 'max:255'],
            'sort_order' => ['nullable', 'integer'],
        ], [
            'cycle_day_to.gte' => 'روز پایان نمی‌تواند کوچک‌تر از روز شروع باشد.',
        ]);

        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;
        $data['cycle_day_from'] = $data['cycle_day_from'] ?? null;
        $data['cycle_day_to'] = $data['cycle_day_to'] ?? null;

        return $data;
    }
}
