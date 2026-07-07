<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * One editable smart-message content entry, keyed by (group, item_key, locale).
 * @see App\Services\MessageSystem\Support\MessageContentRepository
 */
class MessageContent extends Model
{
    protected $fillable = [
        'group',
        'item_key',
        'locale',
        'label',
        'payload',
        'is_active',
        'is_approved',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'payload' => 'array',
            'is_active' => 'boolean',
            'is_approved' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    /** Rows safe to serve to end users: active and approved. */
    public function scopeLive(Builder $query): Builder
    {
        return $query->where('is_active', true)->where('is_approved', true);
    }
}
