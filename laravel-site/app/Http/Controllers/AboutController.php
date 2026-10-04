<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /about (design/html/about.html) — pages/about.blade.php: story + numbers, values, red lines, team + scientific
 * council (#review-policy, the E-E-A-T anchor), links, promise banner. All copy and the team list come from
 * lang/fa/about.php (no database: the team is a short content list; photos are media ids). JSON-LD: AboutPage.
 */
final class AboutController
{
    public function __construct(
        private readonly SeoManager $seo,
        private readonly SeoMetaRepository $meta,
        private readonly SchemaGraph $graph,
        private readonly ViewFactory $views,
    ) {}

    public function __invoke(): View
    {
        $meta = $this->meta->forRoute(StaticPage::About->routeName());
        if (($meta->title ?? '') === '') {
            $this->seo->rawTitle(self::text('about.seo.title'));
        }
        if (($meta->description ?? '') === '') {
            $this->seo->description(self::text('about.seo.description'));
        }
        $this->graph->pageType(WebPageType::AboutPage)->pageName(StaticPage::About->label());

        return $this->views->make('pages.about', [
            'stats' => self::rows('about.story.stats', ['value', 'label']),
            'values' => self::rows('about.values.items', ['title', 'text']),
            'redLines' => self::strings('about.red_lines.items'),
            'team' => self::team(),
            'careersUrl' => self::nullableText('about.links.careers.url'),
        ]);
    }

    /**
     * @return list<array{name: string, role: string, media: int|null}>
     */
    private static function team(): array
    {
        $members = __('about.team.members');
        $team = [];
        foreach (is_array($members) ? $members : [] as $member) {
            if (! is_array($member)) {
                continue;
            }
            $team[] = [
                'name' => is_string($member['name'] ?? null) ? $member['name'] : '',
                'role' => is_string($member['role'] ?? null) ? $member['role'] : '',
                'media' => is_int($member['media'] ?? null) ? $member['media'] : null,
            ];
        }

        return $team;
    }

    /**
     * @param  list<string>  $keys
     * @return list<array<string, string>>
     */
    private static function rows(string $key, array $keys): array
    {
        $value = __($key);
        $rows = [];
        foreach (is_array($value) ? $value : [] as $row) {
            if (! is_array($row)) {
                continue;
            }
            $rows[] = array_combine($keys, array_map(static fn (string $k): string => is_string($row[$k] ?? null) ? $row[$k] : '', $keys));
        }

        return $rows;
    }

    /**
     * @return list<string>
     */
    private static function strings(string $key): array
    {
        $value = __($key);

        return array_values(array_filter(is_array($value) ? $value : [], is_string(...)));
    }

    private static function nullableText(string $key): ?string
    {
        $text = __($key);

        return is_string($text) && $text !== '' && $text !== $key ? $text : null;
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
