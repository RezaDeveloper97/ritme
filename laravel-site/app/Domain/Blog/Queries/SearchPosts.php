<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Models\Post;
use App\Support\Text\PersianDigits;
use Illuminate\Database\Eloquent\Builder;
use InvalidArgumentException;

/**
 * Published posts whose title, excerpt or body contain every token (site search, L4-04). Portable `LIKE` (SQLite +
 * MySQL/MariaDB, no FULLTEXT index needed): the columns are normalised in SQL with a LOWER + REPLACE chain built from
 * the caller's character map, so the tokens must be normalised with the same map. A token with digits also matches
 * its Persian-digit form. Title matches first, then newest.
 */
final class SearchPosts
{
    /** LIKE escape character; `\` is not portable (MySQL string literals treat it as an escape). */
    private const ESCAPE = '!';

    /**
     * @param  list<string>  $tokens  normalised, non-empty
     * @param  array<string, string>  $characterMap  from => to, constant (inlined into SQL)
     */
    public function __construct(
        private readonly array $tokens,
        private readonly array $characterMap,
        private readonly int $limit = 50,
    ) {
        foreach ($characterMap as $from => $to) {
            if (str_contains($from.$to, "'") || str_contains($from.$to, '\\')) {
                throw new InvalidArgumentException('Search character map may not contain quotes or backslashes.');
            }
        }
    }

    /**
     * @return list<PostCardData>
     */
    public function get(): array
    {
        if ($this->tokens === []) {
            return [];
        }

        $model = new Post;
        $title = $this->normalized($model->qualifyColumn('title'));
        $columns = [$title, $this->normalized($model->qualifyColumn('excerpt')), $this->normalized($model->qualifyColumn('body'))];

        $query = Post::query()->published();
        foreach ($this->tokens as $token) {
            $patterns = $this->patterns($token);
            $query->where(static function (Builder $q) use ($columns, $patterns): void {
                foreach ($columns as $column) {
                    foreach ($patterns as $pattern) {
                        $q->orWhereRaw("{$column} LIKE ? ESCAPE '".self::ESCAPE."'", [$pattern]);
                    }
                }
            });
        }

        $titleCases = [];
        $titleBindings = [];
        foreach ($this->tokens as $token) {
            foreach ($this->patterns($token) as $pattern) {
                $titleCases[] = "{$title} LIKE ? ESCAPE '".self::ESCAPE."'";
                $titleBindings[] = $pattern;
            }
        }

        return array_values($query
            ->with('category')
            ->orderByRaw('CASE WHEN '.implode(' OR ', $titleCases).' THEN 0 ELSE 1 END', $titleBindings)
            ->orderByDesc('published_at')
            ->orderByDesc('id')
            ->limit(max(1, min(200, $this->limit)))
            ->get()
            ->map(PostCardData::fromModel(...))
            ->all());
    }

    private function normalized(string $column): string
    {
        $sql = "LOWER(COALESCE({$column}, ''))";
        foreach ($this->characterMap as $from => $to) {
            $sql = "REPLACE({$sql}, '{$from}', '{$to}')";
        }

        return $sql;
    }

    /**
     * @return list<string>
     */
    private function patterns(string $token): array
    {
        $variants = array_values(array_unique([$token, PersianDigits::toPersian($token)]));

        return array_map(static fn (string $variant): string => '%'.strtr($variant, [
            self::ESCAPE => self::ESCAPE.self::ESCAPE, '%' => self::ESCAPE.'%', '_' => self::ESCAPE.'_',
        ]).'%', $variants);
    }
}
