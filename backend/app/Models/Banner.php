<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Storage;

/**
 * @OA\Schema(
 *     schema="Banner",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="title", type="string", nullable=true, example="جشنواره تخفیف"),
 *     @OA\Property(property="image_url", type="string", example="http://ritmeapp.ir/storage/banners/abc.jpg"),
 *     @OA\Property(property="position", type="string", enum={"home_top","home_middle","home_bottom"}),
 *     @OA\Property(property="link_url", type="string", nullable=true, example="/calendar"),
 *     @OA\Property(property="link_type", type="string", nullable=true, enum={"internal","external"})
 * )
 */
class Banner extends Model
{
    use HasLocalizedContent;

    protected $fillable = [
        'title',
        'image_path',
        'position',
        'link_url',
        'link_type',
        'starts_at',
        'ends_at',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'title' => 'array',
            'starts_at' => 'datetime',
            'ends_at' => 'datetime',
            'is_active' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    /**
     * Public URL of the stored image (served via `storage:link`). Kept as an
     * accessor rather than a stored absolute URL so it stays correct if APP_URL
     * changes between environments.
     */
    protected function imageUrl(): Attribute
    {
        return Attribute::make(
            get: fn (): ?string => $this->image_path
                ? Storage::disk('public')->url($this->image_path)
                : null,
        );
    }

    /**
     * Banners that should be shown right now: active, and within their
     * (optional) start/end window. Ordered for display.
     */
    public function scopeActive(Builder $query): Builder
    {
        $now = now();

        return $query
            ->where('is_active', true)
            ->where(fn (Builder $q) => $q->whereNull('starts_at')->orWhere('starts_at', '<=', $now))
            ->where(fn (Builder $q) => $q->whereNull('ends_at')->orWhere('ends_at', '>=', $now))
            ->orderBy('sort_order')
            ->orderByDesc('id');
    }

    public function scopeForPosition(Builder $query, string $position): Builder
    {
        return $query->where('position', $position);
    }
}
