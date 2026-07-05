<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class UserTaskCompletion extends Model
{
    protected $fillable = [
        'user_id',
        'task_template_id',
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

    public function taskTemplate(): BelongsTo
    {
        return $this->belongsTo(TaskTemplate::class);
    }
}
