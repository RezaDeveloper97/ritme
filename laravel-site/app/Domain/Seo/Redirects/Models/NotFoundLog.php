<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Models;

use App\Domain\Seo\Redirects\Enums\AgentClass;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * Aggregated 404s (one row per path, L7-03), filled in batches by FlushRedirectStats. No personal data.
 *
 * @property int $id
 * @property string $path
 * @property string $path_hash
 * @property string|null $referer
 * @property AgentClass $agent
 * @property int $hits
 * @property Carbon|null $first_seen_at
 * @property Carbon|null $last_seen_at
 */
final class NotFoundLog extends Model
{
    public $timestamps = false;

    protected $table = 'not_found_logs';

    protected $fillable = ['path', 'path_hash', 'referer', 'agent', 'hits', 'first_seen_at', 'last_seen_at'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'agent' => AgentClass::class,
            'hits' => 'integer',
            'first_seen_at' => 'datetime',
            'last_seen_at' => 'datetime',
        ];
    }
}
