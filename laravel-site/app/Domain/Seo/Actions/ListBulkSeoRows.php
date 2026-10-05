<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Seo\Analysis\SeoAnalysis;
use App\Domain\Seo\Analysis\TextWidth;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Support\Robots;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SeoDefaults;
use BackedEnum;
use Illuminate\Support\Collection;

/**
 * Every row of the bulk SEO editor (L7-06): posts, products, places, blog + shop categories and the indexable static
 * pages, each with its own overrides (title / description / robots), the effective title + description Google sees,
 * the stored analyser score and the list checks — missing override, too long (chars or SERP pixels), duplicate
 * (effective title / description shared with another indexable row; a static page is matched by its route name, so it
 * never collides with itself), noindex, «نیاز به کار».
 *
 * Queries: one per model type (narrow columns, body head only) + one for all seo_meta rows; static pages come from
 * ListStaticPageSeo (cached repository).
 */
final class ListBulkSeoRows
{
    private const BODY_HEAD = 1200;

    public function __construct(
        private readonly ResolveSeoTarget $targets,
        private readonly ListStaticPageSeo $staticPages,
        private readonly SettingsRepository $settings,
    ) {}

    /**
     * @return list<array<string, mixed>>
     */
    public function handle(): array
    {
        $seo = $this->settings->all()->seo;
        $meta = SeoMeta::query()->get(['id', 'seoable_type', 'seoable_id', 'route_name', 'title', 'description', 'robots', 'score', 'score_checked_at', 'cornerstone']);
        $byModel = $meta->filter(static fn (SeoMeta $m): bool => $m->seoable_type !== null)
            ->keyBy(static fn (SeoMeta $m): string => ResolveSeoTarget::key((string) $m->seoable_type, (int) $m->seoable_id));
        $byRoute = $meta->filter(static fn (SeoMeta $m): bool => $m->route_name !== null)->keyBy('route_name');

        $rows = [...$this->modelRows($byModel, $seo), ...$this->pageRows($byRoute)];

        return $this->withChecks($rows);
    }

    /**
     * @param  Collection<string, SeoMeta>  $byModel
     * @return list<array<string, mixed>>
     */
    private function modelRows(Collection $byModel, SeoDefaults $seo): array
    {
        $rows = [];
        foreach (ResolveSeoTarget::TYPES as $type => $config) {
            $columns = ['id', 'slug', $config['name'], $config['excerpt']];
            $published = match ($type) {
                'blog_post', 'directory_place' => 'status',
                'shop_product' => 'is_published',
                'shop_category' => 'is_active',
                default => null,
            };
            if ($published !== null) {
                $columns[] = $published;
            }

            $query = $config['model']::query()->select(array_values(array_unique($columns)))->orderBy('id');
            if ($config['body'] !== null) {
                // Only the start of the body is needed (description fallback); never load whole articles here.
                $query->selectRaw('substr('.$config['body'].', 1, '.self::BODY_HEAD.') as body_head');
            }

            foreach ($query->get() as $model) {
                $key = ResolveSeoTarget::key($type, (int) $model->getKey());
                $row = $byModel->get($key);
                $name = trim((string) $model->getAttribute($config['name']));
                $title = trim((string) $row?->title);
                $state = $published !== null ? $model->getAttribute($published) : true;

                $rows[] = $this->row($key, $type, $config['label'], $name, $this->targets->url($type, (string) $model->getAttribute('slug')), $row, [
                    'effective_title' => $seo->title($title !== '' ? $title : $name),
                    'effective_description' => ResolveSeoTarget::description(
                        $row?->description,
                        (string) $model->getAttribute($config['excerpt']),
                        $config['body'] !== null ? (string) $model->getAttribute('body_head') : null,
                        $seo,
                    ),
                    'published' => $state === true || ($state instanceof BackedEnum && $state->value === 'published'),
                ]);
            }
        }

        return $rows;
    }

    /**
     * @param  Collection<string, SeoMeta>  $byRoute
     * @return list<array<string, mixed>>
     */
    private function pageRows(Collection $byRoute): array
    {
        $rows = [];
        foreach ($this->staticPages->handle() as $page) {
            $rows[] = $this->row(
                ResolveSeoTarget::pageKey($page->page),
                ResolveSeoTarget::PAGE,
                'صفحه ثابت',
                $page->page->label(),
                $page->url,
                $byRoute->get($page->page->routeName()),
                ['effective_title' => $page->title, 'effective_description' => $page->description, 'published' => true],
            );
        }

        return $rows;
    }

    /**
     * @param  array{effective_title: string, effective_description: string, published: bool}  $effective
     * @return array<string, mixed>
     */
    private function row(string $key, string $type, string $label, string $name, string $url, ?SeoMeta $meta, array $effective): array
    {
        $title = trim($effective['effective_title']);
        $description = trim($effective['effective_description']);

        return [
            'key' => $key,
            'type' => $type,
            'type_label' => $label,
            'name' => $name,
            'url' => $url,
            'published' => $effective['published'],
            'meta_id' => $meta?->id,
            'title' => (string) $meta?->title,
            'description' => (string) $meta?->description,
            'robots' => (string) $meta?->robots,
            'indexable' => Robots::parse($meta?->robots)->index,
            'effective_title' => $title,
            'effective_description' => $description,
            'title_chars' => TextWidth::chars($title),
            'title_px' => TextWidth::pixels($title, TextWidth::TITLE_FONT_PX),
            'description_chars' => TextWidth::chars($description),
            'description_px' => TextWidth::pixels($description, TextWidth::DESCRIPTION_FONT_PX),
            'score' => $meta?->score,
            'score_checked_at' => $meta?->score_checked_at?->toIso8601String(),
            'cornerstone' => (bool) $meta?->cornerstone,
        ];
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     * @return list<array<string, mixed>>
     */
    private function withChecks(array $rows): array
    {
        $normalize = static fn (mixed $text): string => mb_strtolower(trim((string) preg_replace('/\s+/u', ' ', (string) $text)));
        $titles = [];
        $descriptions = [];
        foreach ($rows as $row) {
            if ($row['indexable'] === true) {
                $titles[$normalize($row['effective_title'])][] = $row['key'];
                $descriptions[$normalize($row['effective_description'])][] = $row['key'];
            }
        }

        foreach ($rows as $i => $row) {
            $t = $normalize($row['effective_title']);
            $d = $normalize($row['effective_description']);
            $duplicateTitle = $row['indexable'] === true && $t !== '' && count($titles[$t] ?? []) > 1;
            $duplicateDescription = $row['indexable'] === true && $d !== '' && count($descriptions[$d] ?? []) > 1;
            $tooLong = $row['title_chars'] > TextWidth::TITLE_CHARS[1] || $row['title_px'] > TextWidth::TITLE_MAX_PX
                || $row['description_chars'] > TextWidth::DESCRIPTION_CHARS[1] || $row['description_px'] > TextWidth::DESCRIPTION_MAX_PX;
            $missing = $row['title'] === '' || $row['description'] === '';
            $score = $row['score'];

            $rows[$i] += [
                'duplicate_title' => $duplicateTitle,
                'duplicate_description' => $duplicateDescription,
                'too_long' => $tooLong,
                'missing' => $missing,
                'needs_work' => $score === null || $score < SeoAnalysis::NEEDS_WORK_SCORE || $duplicateTitle || $duplicateDescription || $tooLong,
            ];
        }

        return $rows;
    }
}
