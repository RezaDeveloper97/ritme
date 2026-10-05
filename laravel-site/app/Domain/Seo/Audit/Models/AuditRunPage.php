<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * One page audited in a run.
 *
 * @property int $id
 * @property int $run_id
 * @property string $path
 * @property string $path_hash
 * @property string|null $title
 * @property int $status
 * @property string $source
 * @property string|null $route_name
 * @property string|null $content_type
 * @property bool $indexable
 * @property bool $in_sitemap
 * @property int $response_ms
 * @property int $html_bytes
 * @property int $requests_count
 * @property int $inbound_links
 * @property int|null $analysis_score
 * @property int $score
 * @property int $errors_count
 * @property int $warnings_count
 * @property int $notices_count
 * @property string|null $content_hash
 * @property string|null $edit_url
 */
final class AuditRunPage extends Model
{
    public $timestamps = false;

    protected $table = 'seo_audit_pages';

    protected $fillable = [
        'run_id', 'path', 'path_hash', 'title', 'status', 'source', 'route_name', 'content_type', 'indexable', 'in_sitemap',
        'response_ms', 'html_bytes', 'requests_count', 'inbound_links', 'analysis_score', 'score', 'errors_count',
        'warnings_count', 'notices_count', 'content_hash', 'edit_url',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'run_id' => 'integer',
            'status' => 'integer',
            'indexable' => 'boolean',
            'in_sitemap' => 'boolean',
            'response_ms' => 'integer',
            'html_bytes' => 'integer',
            'requests_count' => 'integer',
            'inbound_links' => 'integer',
            'analysis_score' => 'integer',
            'score' => 'integer',
            'errors_count' => 'integer',
            'warnings_count' => 'integer',
            'notices_count' => 'integer',
        ];
    }

    /**
     * @return BelongsTo<AuditRun, $this>
     */
    public function run(): BelongsTo
    {
        return $this->belongsTo(AuditRun::class, 'run_id');
    }
}
