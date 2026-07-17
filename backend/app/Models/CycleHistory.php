<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class CycleHistory extends Model
{
    protected $fillable = [
        'user_id',
        'period_start_date',
        'period_end_date',
        'cycle_length',
        'bleeding_length',
        'is_confirmed',
        'is_estimated',
        'source',
        'data_quality_flags',
    ];

    protected function casts(): array
    {
        return [
            'period_start_date' => 'date',
            'period_end_date' => 'date',
            'is_confirmed' => 'boolean',
            'is_estimated' => 'boolean',
            'data_quality_flags' => 'array',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
