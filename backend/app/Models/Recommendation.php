<?php

namespace App\Models;

use App\Enums\RecommendationType;
use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * One admin-editable daily recommendation ("توصیه‌های امروز" on the home page).
 *
 * Targeting is layered: `cycle_phase` (null = every phase) narrowed by
 * `cycle_subphases` (empty = every sub-phase of that phase), and optionally
 * gated on a logged symptom via `symptom_trigger`.
 *
 * @see App\Services\HealthEngine\RecommendationRepository
 */
class Recommendation extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'key',
        'type',
        'title',
        'text',
        'cycle_phase',
        'cycle_subphases',
        'symptom_trigger',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'title' => 'array',
            'text' => 'array',
            'cycle_subphases' => 'array',
            'is_active' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    /**
     * Whether this row should be shown on a day in the given phase and
     * sub-phase, for a user reporting `$triggers` today.
     *
     * All three targets widen when left blank: no phase means every phase, no
     * sub-phase list means every sub-phase of the phase, and no trigger means
     * the row is not symptom-gated.
     *
     * Matched in PHP rather than SQL — see {@see App\Services\HealthEngine\RecommendationRepository}
     * for why the whole set is loaded once instead of queried per day.
     *
     * @param  array<int, string>  $triggers  trigger keys today's log satisfies
     */
    public function appliesTo(?string $phase, ?string $subphase, array $triggers): bool
    {
        if ($this->cycle_phase !== null && $this->cycle_phase !== $phase) {
            return false;
        }

        if ($this->symptom_trigger !== null && ! in_array($this->symptom_trigger, $triggers, true)) {
            return false;
        }

        $subphases = $this->cycle_subphases ?: [];

        return $subphases === [] || ($subphase !== null && in_array($subphase, $subphases, true));
    }

    /**
     * The engine's daily-tip shape: bilingual text plus the resolved category
     * title, so downstream localization has everything it needs.
     *
     * @return array{type: string, en: string, fa: string, title: array{fa: string, en: string}}
     */
    public function toTip(): array
    {
        $type = $this->type ?: RecommendationType::GENERAL->value;

        return [
            'type' => $type,
            'fa' => (string) ($this->text['fa'] ?? $this->text['en'] ?? ''),
            'en' => (string) ($this->text['en'] ?? $this->text['fa'] ?? ''),
            'title' => [
                'fa' => (string) ($this->title['fa'] ?? RecommendationType::labelFor($type, 'fa')),
                'en' => (string) ($this->title['en'] ?? RecommendationType::labelFor($type, 'en')),
            ],
        ];
    }
}
