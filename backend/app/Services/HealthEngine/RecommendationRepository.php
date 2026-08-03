<?php

namespace App\Services\HealthEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\RecommendationTrigger;
use App\Models\DailyHealthLog;
use App\Models\Recommendation;
use Illuminate\Support\Collection;

/**
 * Resolves the admin-managed daily recommendations for one day of the cycle.
 *
 * Mirrors {@see App\Services\MessageSystem\Support\MessageContentRepository}:
 * content lives in the database and the calling engine keeps a code fallback,
 * so an unseeded install still shows something. Registered as a singleton
 * (AppServiceProvider) so the one query it runs is shared by every engine built
 * during a request — the home page alone constructs two.
 *
 * The whole active set is loaded once and matched in PHP. It is tens of rows,
 * the sub-phase list already had to be matched in PHP (JSON containment differs
 * between MySQL and the SQLite used in tests), and the alternative — a query per
 * day keyed on that day's symptoms — cost 27 queries on a single month view,
 * because the symptom set changes almost daily and defeats any cache.
 */
class RecommendationRepository
{
    /** @var Collection<int, Recommendation>|null */
    private ?Collection $rows = null;

    private ?bool $hasContent = null;

    /** @var array<string, array<int, array<string, mixed>>> */
    private array $cache = [];

    /**
     * Whether any recommendation has been defined at all. False only on an
     * install that has never been seeded, which is the engine's cue to use its
     * built-in copy.
     *
     * Deliberately counts inactive rows too: an admin who switched every
     * recommendation off means the card to be empty, and reviving the built-in
     * copy behind their back would make the panel look broken. (Deleting every
     * row rather than deactivating it does fall back — the panel steers admins
     * to the toggle for exactly this reason.)
     */
    public function hasContent(): bool
    {
        // The loaded set only holds active rows, so an all-inactive table still
        // needs the explicit probe — but a populated one answers for free.
        return $this->hasContent ??= $this->rows()->isNotEmpty() || Recommendation::query()->exists();
    }

    /**
     * The tips for a day, in the engine's bilingual `daily_tips` shape.
     *
     * Ordering follows the admin's `sort_order`, with symptom-triggered rows
     * first so the advice that answers what the user just logged is the part
     * that survives the clients' "first four" cut.
     *
     * @return array<int, array<string, mixed>>
     */
    public function forDay(CyclePhase $phase, CycleSubphase $subphase, ?DailyHealthLog $log): array
    {
        $canonicalSubphase = $subphase->canonical()->value;
        $triggers = RecommendationTrigger::activeFor($log);

        $cacheKey = $phase->value.'|'.$canonicalSubphase.'|'.implode(',', $triggers);

        if (! array_key_exists($cacheKey, $this->cache)) {
            [$triggered, $plain] = $this->rows()
                ->filter(fn (Recommendation $row): bool => $row->appliesTo($phase->value, $canonicalSubphase, $triggers))
                ->partition(fn (Recommendation $row): bool => $row->symptom_trigger !== null);

            $this->cache[$cacheKey] = $triggered->concat($plain)
                ->map(fn (Recommendation $row): array => $row->toTip())
                ->values()
                ->all();
        }

        return $this->cache[$cacheKey];
    }

    /**
     * Every live recommendation, in the admin's display order. Loaded once per
     * request; matching happens in PHP from here on.
     *
     * @return Collection<int, Recommendation>
     */
    private function rows(): Collection
    {
        return $this->rows ??= Recommendation::query()
            ->active()
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get();
    }
}
