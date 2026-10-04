<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis\Queries;

use App\Domain\Seo\Models\SeoMeta;
use Illuminate\Database\Eloquent\Builder;

/**
 * Whether another seo_meta row already uses this SEO title / meta description (case- and space-insensitive), for the
 * analyser's duplicate checks. The page's own row ($exceptId) is excluded. Only explicit overrides are
 * compared — inherited titles are unique by their content titles. Runs on the debounced admin form, one query.
 */
final class FindDuplicateSeoMeta
{
    /**
     * @return array{title: bool|null, description: bool|null} null = nothing to compare (field empty)
     */
    public function handle(?string $title, ?string $description, ?int $exceptId = null): array
    {
        $title = $this->clean($title);
        $description = $this->clean($description);
        if ($title === '' && $description === '') {
            return ['title' => null, 'description' => null];
        }

        $query = SeoMeta::query()->where(function (Builder $q) use ($title, $description): void {
            if ($title !== '') {
                $q->orWhereRaw('LOWER(TRIM(title)) = ?', [mb_strtolower($title)]);
            }
            if ($description !== '') {
                $q->orWhereRaw('LOWER(TRIM(description)) = ?', [mb_strtolower($description)]);
            }
        });

        if ($exceptId !== null) {
            $query->whereKeyNot($exceptId);
        }

        $titleTaken = false;
        $descriptionTaken = false;
        foreach ($query->limit(20)->get(['title', 'description']) as $row) {
            $titleTaken = $titleTaken || ($title !== '' && mb_strtolower($this->clean($row->title)) === mb_strtolower($title));
            $descriptionTaken = $descriptionTaken || ($description !== '' && mb_strtolower($this->clean($row->description)) === mb_strtolower($description));
        }

        return [
            'title' => $title === '' ? null : $titleTaken,
            'description' => $description === '' ? null : $descriptionTaken,
        ];
    }

    private function clean(?string $text): string
    {
        return trim((string) preg_replace('/\s+/u', ' ', (string) $text));
    }
}
