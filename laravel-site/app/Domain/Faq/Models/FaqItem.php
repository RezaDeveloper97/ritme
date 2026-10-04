<?php

declare(strict_types=1);

namespace App\Domain\Faq\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * One question with its answer (sanitised rich text, see FaqObserver).
 *
 * @property int $id
 * @property int $faq_group_id
 * @property string $question
 * @property string $answer
 * @property bool $is_published
 * @property int $sort_order
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class FaqItem extends Model
{
    protected $table = 'faq_items';

    protected $fillable = ['faq_group_id', 'question', 'answer', 'is_published', 'sort_order'];

    protected $attributes = ['is_published' => true, 'sort_order' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'faq_group_id' => 'integer',
            'is_published' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    /**
     * @return BelongsTo<FaqGroup, $this>
     */
    public function group(): BelongsTo
    {
        return $this->belongsTo(FaqGroup::class, 'faq_group_id');
    }

    /**
     * @param  Builder<self>  $query
     */
    public function scopePublished(Builder $query): void
    {
        $query->where('is_published', true);
    }
}
