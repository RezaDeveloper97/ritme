<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class UserProfile extends Model
{
    use HasFactory;

    protected $fillable = [
        'user_id',
        'birthday',
        'weight',
        'height',
        'period_duration',
        'cycle_duration',
        'last_period_start',
        'calculation_status',
        'calculation_started_at',
        'calculation_completed_at',
        'calculation_version',
    ];

    protected function casts(): array
    {
        return [
            'birthday' => 'date',
            'weight' => 'decimal:2',
            'height' => 'integer',
            'period_duration' => 'integer',
            'cycle_duration' => 'integer',
            'last_period_start' => 'date',
            'calculation_started_at' => 'datetime',
            'calculation_completed_at' => 'datetime',
            'calculation_version' => 'integer',
        ];
    }

    /**
     * Get the user that owns the profile
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
