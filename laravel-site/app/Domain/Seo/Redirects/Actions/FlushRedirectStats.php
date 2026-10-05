<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Enums\AgentClass;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Support\HitBuffer;
use Illuminate\Support\Carbon;

/**
 * Moves the buffered redirect hits and 404s into the database (scheduler, every five minutes): one increment per
 * redirect / path, through the query builder (no observer, so stats never invalidate the cached redirect map).
 * New 404 paths are only inserted while the table holds fewer than MAX_ROWS rows.
 */
final class FlushRedirectStats
{
    public const MAX_ROWS = 5000;

    public function __construct(private readonly HitBuffer $buffer) {}

    /**
     * @return array{redirects: int, not_found: int}
     */
    public function handle(): array
    {
        $now = Carbon::now();

        $redirects = 0;
        foreach ($this->buffer->drain(RecordRedirectHit::BUCKET) as $id => $entry) {
            $redirects += Redirect::query()->whereKey((int) $id)->toBase()
                ->increment('hits', $entry['hits'], ['last_hit_at' => $now]) > 0 ? 1 : 0;
        }

        $notFound = 0;
        $rows = null;
        foreach ($this->buffer->drain(RecordNotFound::BUCKET) as $hash => $entry) {
            $meta = $entry['meta'];
            $path = (string) ($meta['path'] ?? '');
            if ($path === '') {
                continue;
            }

            $extra = ['last_seen_at' => $now];
            if (($meta['referer'] ?? null) !== null) {
                $extra['referer'] = $meta['referer'];
            }

            if (NotFoundLog::query()->where('path_hash', $hash)->toBase()->increment('hits', $entry['hits'], $extra) > 0) {
                $notFound++;

                continue;
            }

            $rows ??= NotFoundLog::query()->count();
            if ($rows >= self::MAX_ROWS) {
                continue;
            }

            $inserted = NotFoundLog::query()->insertOrIgnore([
                'path' => $path,
                'path_hash' => $hash,
                'referer' => $meta['referer'] ?? null,
                'agent' => (AgentClass::tryFrom((string) ($meta['agent'] ?? '')) ?? AgentClass::Human)->value,
                'hits' => $entry['hits'],
                'first_seen_at' => $now,
                'last_seen_at' => $now,
            ]);
            $rows += $inserted;
            $notFound += $inserted;
        }

        return ['redirects' => $redirects, 'not_found' => $notFound];
    }
}
