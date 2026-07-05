<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class UserChallengeCompletion extends Model
{
    protected $fillable = [
        'user_id',
        'challenge_id',
        'completion_date',
        'completed_at',
    ];

    protected function casts(): array
    {
        return [
            'completion_date' => 'date',
            'completed_at' => 'datetime',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function challenge(): BelongsTo
    {
        return $this->belongsTo(Challenge::class);
    }
}
