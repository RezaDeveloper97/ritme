<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Models;

use App\Domain\Seo\Audit\IssueCodes;
use App\Domain\Seo\Audit\Severity;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * One finding of a run, with the admin page that fixes it.
 *
 * @property int $id
 * @property int $run_id
 * @property int|null $page_id
 * @property string $path
 * @property Severity $severity
 * @property string $code
 * @property string $message
 * @property string|null $fix_url
 */
final class AuditRunIssue extends Model
{
    public $timestamps = false;

    protected $table = 'seo_audit_issues';

    protected $fillable = ['run_id', 'page_id', 'path', 'severity', 'code', 'message', 'fix_url'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'run_id' => 'integer',
            'page_id' => 'integer',
            'severity' => Severity::class,
        ];
    }

    public function label(): string
    {
        return IssueCodes::label($this->code);
    }

    /**
     * @return BelongsTo<AuditRun, $this>
     */
    public function run(): BelongsTo
    {
        return $this->belongsTo(AuditRun::class, 'run_id');
    }

    /**
     * @return BelongsTo<AuditRunPage, $this>
     */
    public function page(): BelongsTo
    {
        return $this->belongsTo(AuditRunPage::class, 'page_id');
    }
}
