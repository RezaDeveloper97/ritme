<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\RecommendationTrigger;
use App\Enums\RecommendationType;
use App\Http\Controllers\Controller;
use App\Models\Recommendation;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

/**
 * CRUD for the home page's "توصیه‌های امروز" card. Each row is a piece of advice
 * tied to a cycle phase (and optionally to narrower sub-phases and to a logged
 * symptom); the cycle engine reads them through
 * {@see App\Services\HealthEngine\RecommendationRepository}.
 */
class RecommendationController extends Controller
{
    public function index(Request $request): View
    {
        $phase = $request->query('phase');
        $type = $request->query('type');

        $recommendations = Recommendation::query()
            ->when($phase === 'general', fn ($q) => $q->whereNull('cycle_phase'))
            ->when($phase && $phase !== 'general', fn ($q) => $q->where('cycle_phase', $phase))
            ->when($type, fn ($q) => $q->where('type', $type))
            ->orderByRaw('cycle_phase IS NULL')
            ->orderBy('cycle_phase')
            ->orderBy('sort_order')
            ->orderBy('id')
            ->paginate(30)
            ->withQueryString();

        return view('admin.recommendations.index', [
            'recommendations' => $recommendations,
            'phases' => CyclePhase::options(),
            'types' => RecommendationType::options(),
            'phase' => $phase,
            'type' => $type,
        ]);
    }

    public function create(): View
    {
        return view('admin.recommendations.form', $this->formData(new Recommendation([
            'is_active' => true,
            'type' => RecommendationType::GENERAL->value,
        ])));
    }

    public function store(Request $request): RedirectResponse
    {
        Recommendation::create($this->validated($request));

        return redirect()->route('admin.recommendations.index')->with('status', 'توصیه ایجاد شد.');
    }

    public function edit(Recommendation $recommendation): View
    {
        return view('admin.recommendations.form', $this->formData($recommendation));
    }

    public function update(Request $request, Recommendation $recommendation): RedirectResponse
    {
        $recommendation->update($this->validated($request));

        return redirect()->route('admin.recommendations.index')->with('status', 'توصیه به‌روزرسانی شد.');
    }

    public function destroy(Recommendation $recommendation): RedirectResponse
    {
        $recommendation->delete();

        return back()->with('status', 'توصیه حذف شد.');
    }

    public function toggle(Recommendation $recommendation): RedirectResponse
    {
        $recommendation->update(['is_active' => ! $recommendation->is_active]);

        return back()->with('status', 'وضعیت توصیه تغییر کرد.');
    }

    /** Shared create/edit view data. */
    private function formData(Recommendation $recommendation): array
    {
        return [
            'recommendation' => $recommendation,
            'phases' => CyclePhase::options(),
            // Only the sub-phases a calculated day can actually report — see
            // CyclePhase::subphases(). Offering the rest would let an admin save
            // a recommendation that could never be shown.
            'subphases' => array_map(
                fn (CycleSubphase $case): array => ['value' => $case->value, 'label' => $case->label('fa')],
                CyclePhase::allSubphases(),
            ),
            // sub-phase key => the phase it sits in, so the picker can offer only
            // the sub-phases the selected phase actually reaches.
            'subphasePhases' => $this->subphasePhaseMap(),
            'types' => RecommendationType::options(),
            'triggers' => RecommendationTrigger::options(),
        ];
    }

    /**
     * @return array<string, string>
     */
    private function subphasePhaseMap(): array
    {
        $map = [];

        foreach (CyclePhase::cases() as $phase) {
            foreach ($phase->subphases() as $subphase) {
                $map[$subphase->value] = $phase->value;
            }
        }

        return $map;
    }

    private function validated(Request $request): array
    {
        $data = $request->validate([
            'type' => ['required', Rule::in(RecommendationType::values())],
            'title' => ['nullable', 'array'],
            'title.fa' => ['nullable', 'string', 'max:255'],
            'title.en' => ['nullable', 'string', 'max:255'],
            'text' => ['required', 'array'],
            'text.fa' => ['required', 'string', 'max:2000'],
            'text.en' => ['nullable', 'string', 'max:2000'],
            'cycle_phase' => ['nullable', Rule::in(CyclePhase::values())],
            'cycle_subphases' => ['nullable', 'array'],
            // Restricted to the sub-phases the chosen phase can actually reach:
            // a luteal sub-phase on a menstruation row would silently never fire.
            'cycle_subphases.*' => [Rule::in(CyclePhase::subphaseValuesFor($request->input('cycle_phase') ?: null))],
            'symptom_trigger' => ['nullable', Rule::in(RecommendationTrigger::values())],
            'sort_order' => ['nullable', 'integer'],
        ]);

        // An all-blank title means "use the category label" — store null rather
        // than an object of empty strings the resolver would have to defend against.
        $title = array_filter($data['title'] ?? [], fn ($v) => filled($v));
        $data['title'] = $title ?: null;

        $data['cycle_subphases'] = array_values($data['cycle_subphases'] ?? []) ?: null;
        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;

        return $data;
    }
}
