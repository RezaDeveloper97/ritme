<?php

declare(strict_types=1);

namespace App\Domain\Directory\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A previous slug of a published place (written by PlaceSlugger when the slug changes).
 *
 * @property int $id
 * @property int $place_id
 * @property string $slug
 * @property Carbon|null $created_at
 */
final class PlaceSlug extends Model
{
    public const UPDATED_AT = null;

    protected $table = 'directory_place_slugs';

    protected $fillable = ['place_id', 'slug'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['place_id' => 'integer'];
    }

    /**
     * @return BelongsTo<Place, $this>
     */
    public function place(): BelongsTo
    {
        return $this->belongsTo(Place::class);
    }
}
