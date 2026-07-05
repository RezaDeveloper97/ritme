<?php

namespace App\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @OA\Schema(
 *     schema="Reminder",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="type", type="string", enum={"doctor","medication","appointment","custom"}),
 *     @OA\Property(property="title", type="string", example="دکتر عابدزاده"),
 *     @OA\Property(property="subtitle", type="string", nullable=true, example="متخصص زنان و زایمان"),
 *     @OA\Property(property="notes", type="string", nullable=true),
 *     @OA\Property(property="scheduled_at", type="string", format="date-time", nullable=true),
 *     @OA\Property(property="recurrence", type="string", enum={"none","daily","weekly","monthly"}),
 *     @OA\Property(property="recurrence_time", type="string", nullable=true, example="16:00"),
 *     @OA\Property(property="is_active", type="boolean", example=true)
 * )
 */
class Reminder extends Model
{
    protected $fillable = [
        'user_id',
        'type',
        'title',
        'subtitle',
        'notes',
        'scheduled_at',
        'recurrence',
        'recurrence_time',
        'starts_on',
        'ends_on',
        'is_active',
        'meta',
    ];

    protected function casts(): array
    {
        // Note: recurrence_time is intentionally left uncast — it is read as a
        // raw "H:i:s" string (see MedicationReminderSection). Do not add a
        // time/datetime cast without updating those consumers.
        return [
            'scheduled_at' => 'datetime',
            'starts_on' => 'date',
            'ends_on' => 'date',
            'is_active' => 'boolean',
            'meta' => 'array',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    public function scopeOfType(Builder $query, string $type): Builder
    {
        return $query->where('type', $type);
    }

    /**
     * Reminders relevant to a given day: one-off reminders scheduled that day
     * (or still upcoming) plus active recurring reminders within their window.
     */
    public function scopeRelevantOn(Builder $query, Carbon $date): Builder
    {
        return $query->where(function (Builder $q) use ($date) {
            // Recurring reminders within their optional active window
            $q->where(function (Builder $rec) use ($date) {
                $rec->where('recurrence', '!=', 'none')
                    ->where(function (Builder $w) use ($date) {
                        $w->whereNull('starts_on')->orWhereDate('starts_on', '<=', $date);
                    })
                    ->where(function (Builder $w) use ($date) {
                        $w->whereNull('ends_on')->orWhereDate('ends_on', '>=', $date);
                    });
            })
            // One-off reminders for this day or still in the future
            ->orWhere(function (Builder $once) use ($date) {
                $once->where('recurrence', 'none')
                    ->whereNotNull('scheduled_at')
                    ->where('scheduled_at', '>=', $date->copy()->startOfDay());
            });
        });
    }
}
