<?php

declare(strict_types=1);

namespace App\Domain\Faq\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A named set of questions (`home`, `plus`, `stage-cycle`, a /faq category …). Read through FaqRepository.
 *
 * @property int $id
 * @property string $slug
 * @property string $title
 * @property bool $is_listed
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class FaqGroup extends Model
{
    protected $table = 'faq_groups';

    protected $fillable = ['slug', 'title', 'is_listed', 'sort_order'];

    protected $attributes = ['is_listed' => false, 'sort_order' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'is_listed' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    /**
     * @return HasMany<FaqItem, $this>
     */
    public function items(): HasMany
    {
        return $this->hasMany(FaqItem::class, 'faq_group_id');
    }
}
