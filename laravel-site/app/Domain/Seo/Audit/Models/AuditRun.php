<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Models;

use App\Domain\Seo\Audit\Enums\RunStatus;
use App\Domain\Seo\Audit\Enums\RunTrigger;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * One SEO audit run (L7-05).
 *
 * @property int $id
 * @property RunStatus $status
 * @property RunTrigger $trigger
 * @property int|null $triggered_by
 * @property int $max_pages
 * @property int $time_limit
 * @property int $pages_count
 * @property int $errors_count
 * @property int $warnings_count
 * @property int $notices_count
 * @property int|null $score
 * @property bool $truncated
 * @property string|null $truncated_by
 * @property int|null $duration_ms
 * @property string|null $message
 * @property Carbon|null $started_at
 * @property Carbon|null $finished_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class AuditRun extends Model
{
    protected $table = 'seo_audit_runs';

    protected $fillable = [
        'status', 'trigger', 'triggered_by', 'max_pages', 'time_limit', 'pages_count', 'errors_count', 'warnings_count',
        'notices_count', 'score', 'truncated', 'truncated_by', 'duration_ms', 'message', 'started_at', 'finished_at',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'status' => RunStatus::class,
            'trigger' => RunTrigger::class,
            'triggered_by' => 'integer',
            'max_pages' => 'integer',
            'time_limit' => 'integer',
            'pages_count' => 'integer',
            'errors_count' => 'integer',
            'warnings_count' => 'integer',
            'notices_count' => 'integer',
            'score' => 'integer',
            'truncated' => 'boolean',
            'duration_ms' => 'integer',
            'started_at' => 'datetime',
            'finished_at' => 'datetime',
        ];
    }

    /**
     * @return HasMany<AuditRunPage, $this>
     */
    public function pages(): HasMany
    {
        return $this->hasMany(AuditRunPage::class, 'run_id');
    }

    /**
     * @return HasMany<AuditRunIssue, $this>
     */
    public function issues(): HasMany
    {
        return $this->hasMany(AuditRunIssue::class, 'run_id');
    }
}
