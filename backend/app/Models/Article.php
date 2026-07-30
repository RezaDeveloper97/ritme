<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Storage;

/**
 * @OA\Schema(
 *     schema="Article",
 *     type="object",
 *
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="slug", type="string", example="period-blood-clots"),
 *     @OA\Property(property="title", type="object", @OA\Property(property="fa", type="string"), @OA\Property(property="en", type="string")),
 *     @OA\Property(property="excerpt", type="object", nullable=true),
 *     @OA\Property(property="body", type="object", nullable=true),
 *     @OA\Property(property="cycle_phases", type="array", nullable=true, description="CycleSubphase keys the article is tagged with (legacy rows may hold a CyclePhase key); null = general, shown in every phase",
 *
 *         @OA\Items(type="string", example="high_fertility")
 *     ),
 *
 *     @OA\Property(property="category", type="string", nullable=true),
 *     @OA\Property(property="read_time_minutes", type="integer", nullable=true, example=4),
 *     @OA\Property(property="image_url", type="string", nullable=true),
 *     @OA\Property(property="is_published", type="boolean", example=true)
 * )
 */
class Article extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'slug',
        'title',
        'excerpt',
        'body',
        'cycle_phases',
        'category',
        'read_time_minutes',
        'image_url',
        'image_path',
        'is_published',
        'published_at',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'title' => 'array',
            'excerpt' => 'array',
            'body' => 'array',
            'cycle_phases' => 'array',
            'read_time_minutes' => 'integer',
            'is_published' => 'boolean',
            'published_at' => 'datetime',
            'sort_order' => 'integer',
        ];
    }

    /**
     * Public URL of the cover image. An uploaded file (image_path, stored on
     * the "public" disk) wins over the manually entered external URL, and is
     * resolved through the disk so the host follows APP_URL.
     */
    protected function imageUrl(): Attribute
    {
        return Attribute::make(
            get: fn (?string $value): ?string => $this->image_path
                ? Storage::disk('public')->url($this->image_path)
                : $value,
        );
    }

    public function scopePublished(Builder $query): Builder
    {
        return $query->where('is_published', true);
    }

    /**
     * Articles that apply to the given phase(s), phase-specific first then
     * general (untagged) ones.
     *
     * An article carries a LIST of phases, and several keys may be passed in,
     * so a row matches when the two sets intersect. Both granularities are
     * accepted because a row may be tagged with a fine-grained sub-phase
     * (CycleSubphase — what the admin panel offers) or a legacy main phase
     * (CyclePhase), and either should surface for a reader in that part of the
     * cycle.
     *
     * @param  string|array<int, string|null>|null  $phase
     */
    public function scopeForPhase(Builder $query, string|array|null $phase): Builder
    {
        $phases = array_values(array_unique(array_filter((array) $phase)));

        return $query->where(function (Builder $q) use ($phases) {
            $q->whereNull('cycle_phases');
            foreach ($phases as $candidate) {
                $q->orWhereJsonContains('cycle_phases', $candidate);
            }
        })->orderByRaw('cycle_phases is null')->orderBy('sort_order');
    }
}
