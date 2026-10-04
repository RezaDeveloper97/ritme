<?php

declare(strict_types=1);

namespace App\Filament\Components\Seo;

use App\Domain\Seo\Analysis\AnalysisInput;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Analysis\Queries\FindDuplicateSeoMeta;
use App\Domain\Seo\Analysis\SeoAnalysis;
use App\Domain\Seo\Analysis\SeoAnalyzer;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Support\DescriptionText;
use App\Domain\Seo\Support\Robots;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SiteSettings;
use App\Filament\Forms\Components\MediaPicker;
use Closure;
use Filament\Forms\Components\RichEditor\RichContentRenderer;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Schemas\Components\Component;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Group;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Components\Utilities\Get;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\View as ViewFactory;
use Throwable;

/**
 * The reusable SEO tab for every content resource whose model uses `App\Domain\Seo\Concerns\HasSeo` (posts,
 * categories, tags, authors now; products, places, static pages later). It edits the model's `seo_meta` row through
 * the `seoMeta` morph relationship (saved by Filament after the record; SeoMetaObserver bumps `seo` + `sitemap` +
 * `pages`). Every field is optional — empty inherits (settings defaults → page fallbacks), exactly like SeoManager.
 *
 * Fields: meta title / description with live character + pixel-width counters, focus keyword, Google SERP preview
 * (desktop + mobile), OG title / description / image with a share-card preview, canonical override, robots toggles
 * (index / follow), sitemap include + priority.
 *
 *     Tab::make('سئو')->schema([
 *         SeoFields::make()
 *             ->titleFrom('title')                 // parent form field (or Closure(Get $get, ?Model $record): ?string)
 *             ->descriptionFrom('excerpt')
 *             ->imageFrom('cover_media_id')        // media id used for og:image when no OG image is picked
 *             ->urlUsing(fn (Get $get, ?Model $record): string => url('/blog/'.($get('slug') ?: 'slug'))),
 *     ]),
 *
 * Inside the closures `$get('field')` reads the parent form (the SEO fields themselves live under `seoMeta.*`).
 *
 * Records without a model (static pages, L7-01) use `SeoFields::standalone()`: same fields under `seoMeta.*`, no
 * relationship — the page reads `seoMeta` from its form state, runs `dehydrateRobots()` and saves through an action.
 * `->fallbackTitleIsComplete()` marks a fallback title that already carries the brand (no title template).
 *
 * Content analysis (L7-02): the "تحلیل سئو" section runs `App\Domain\Seo\Analysis\SeoAnalyzer` on every debounced
 * change (title, description, focus keyword …) and when the tab scrolls into view, so body edits made in another tab
 * are picked up. The body is the parent `body` / `content` / `description` field (rich-editor JSON or HTML or plain
 * text), or `->contentFrom()`; the page kind (post / product / place / page / archive) comes from the resource model
 * or `->analysisType()`; `->cornerstone()` applies the stricter cornerstone thresholds. Standalone pages are analysed
 * without a body (their copy lives in the template).
 */
final class SeoFields extends Group
{
    public const VIEW_NAMESPACE = 'ritme-admin-seo';

    public const SITEMAP_PRIORITIES = ['1.0', '0.9', '0.8', '0.7', '0.6', '0.5', '0.4', '0.3', '0.2', '0.1'];

    private string|Closure|null $titleSource = null;

    private string|Closure|null $descriptionSource = null;

    private string|Closure|null $imageSource = null;

    private ?Closure $urlResolver = null;

    private bool|Closure $fallbackTitleIsComplete = false;

    private bool $standalone = false;

    /** Parent form fields tried, in order, as the analysed body when no `contentFrom()` is given. */
    public const CONTENT_FIELDS = ['body', 'content', 'description'];

    private string|Closure|null $contentSource = null;

    private ContentType|Closure|null $analysisType = null;

    private bool|Closure $cornerstone = false;

    /**
     * Without the `seoMeta` relationship: the fields live under `$statePath.*` of the parent form and the caller
     * fills (`fillRobots()`) and saves (`dehydrateRobots()` + an action) them itself.
     */
    public static function standalone(string $statePath = 'seoMeta'): static
    {
        $static = app(self::class, ['schema' => []]);
        $static->standalone = true;
        $static->configure();

        return $static->statePath($statePath);
    }

    protected function setUp(): void
    {
        parent::setUp();

        ViewFactory::replaceNamespace(self::VIEW_NAMESPACE, __DIR__.'/views');

        if (! $this->standalone) {
            $this->relationship('seoMeta');
            $this->mutateRelationshipDataBeforeFillUsing(static fn (array $data): array => self::fillRobots($data));
            $this->mutateRelationshipDataBeforeSaveUsing(static fn (array $data): array => self::dehydrateRobots($data));
            $this->mutateRelationshipDataBeforeCreateUsing(static fn (array $data): array => self::dehydrateRobots($data));
        }
        $this->schema(fn (): array => $this->fields());
    }

    /**
     * The fallback title is a complete <title> (already branded): shown and measured verbatim, without the template.
     */
    public function fallbackTitleIsComplete(bool|Closure $condition = true): static
    {
        $this->fallbackTitleIsComplete = $condition;

        return $this;
    }

    public function titleFrom(string|Closure|null $source): static
    {
        $this->titleSource = $source;

        return $this;
    }

    public function descriptionFrom(string|Closure|null $source): static
    {
        $this->descriptionSource = $source;

        return $this;
    }

    public function imageFrom(string|Closure|null $source): static
    {
        $this->imageSource = $source;

        return $this;
    }

    /**
     * The analysed body: a parent field name or a closure returning HTML / rich-editor JSON / plain text.
     */
    public function contentFrom(string|Closure|null $source): static
    {
        $this->contentSource = $source;

        return $this;
    }

    public function analysisType(ContentType|Closure|null $type): static
    {
        $this->analysisType = $type;

        return $this;
    }

    public function cornerstone(bool|Closure $condition = true): static
    {
        $this->cornerstone = $condition;

        return $this;
    }

    /**
     * @param  Closure(Get, ?Model): string  $resolver  absolute URL of the page (shown in the SERP preview)
     */
    public function urlUsing(?Closure $resolver): static
    {
        $this->urlResolver = $resolver;

        return $this;
    }

    /**
     * @return list<Component>
     */
    private function fields(): array
    {
        return [
            Section::make('نتیجه جست‌وجو')
                ->description('خالی بماند، از عنوان و خلاصه همین صفحه ساخته می‌شود.')
                ->schema([
                    TextInput::make('title')
                        ->label('عنوان سئو (title)')
                        ->maxLength(255)
                        ->live(debounce: 500)
                        ->placeholder(fn (): ?string => $this->fallbackTitle())
                        ->hint(fn (?string $state): string => SerpMeasure::hint($this->fullTitle($state), SerpMeasure::TITLE_FONT_PX, SerpMeasure::TITLE_MAX_PX))
                        ->hintColor(fn (?string $state): string => SerpMeasure::status($this->fullTitle($state), SerpMeasure::TITLE_CHARS, SerpMeasure::TITLE_FONT_PX, SerpMeasure::TITLE_MAX_PX))
                        ->helperText('شمارش با پسوند نام سایت انجام می‌شود؛ پیشنهاد: ۳۰ تا ۶۰ نویسه.'),
                    Textarea::make('description')
                        ->label('توضیح متا (description)')
                        ->rows(3)
                        ->maxLength(500)
                        ->live(debounce: 500)
                        ->placeholder(fn (): ?string => $this->fallbackDescription())
                        ->hint(fn (?string $state): string => SerpMeasure::hint($this->fullDescription($state), SerpMeasure::DESCRIPTION_FONT_PX, SerpMeasure::DESCRIPTION_MAX_PX))
                        ->hintColor(fn (?string $state): string => SerpMeasure::status($this->fullDescription($state), SerpMeasure::DESCRIPTION_CHARS, SerpMeasure::DESCRIPTION_FONT_PX, SerpMeasure::DESCRIPTION_MAX_PX))
                        ->helperText('پیشنهاد: ۷۰ تا ۱۶۰ نویسه، بدون وعده و اغراق.'),
                    TextInput::make('focus_keyword')
                        ->label('کلیدواژه کانونی')
                        ->maxLength(191)
                        ->live(debounce: 500)
                        ->helperText('عبارتی که این صفحه برای آن نوشته شده است.'),
                    SerpPreview::make()
                        ->viewData(fn (Get $get): array => ['serp' => $this->serpPreview($get)]),
                ]),
            Section::make('تحلیل سئو')
                ->description('بررسی زنده عنوان، توضیح، کلیدواژه، ساختار و خوانایی متن؛ هنگام تایپ به‌روز می‌شود.')
                ->collapsible()
                ->schema([
                    SeoAnalysisPanel::make()
                        ->viewData(fn (Get $get): array => ['analysis' => $this->analysis($get)]),
                ]),
            Section::make('اشتراک‌گذاری در شبکه‌های اجتماعی')
                ->description('Open Graph و کارت توییتر؛ خالی بماند، از عنوان و توضیح سئو استفاده می‌شود.')
                ->collapsible()
                ->schema([
                    TextInput::make('og_title')->label('عنوان اشتراک‌گذاری')->maxLength(255)->live(debounce: 500),
                    Textarea::make('og_description')->label('توضیح اشتراک‌گذاری')->rows(2)->maxLength(500)->live(debounce: 500),
                    MediaPicker::make('og_media_id')
                        ->label('تصویر اشتراک‌گذاری')
                        ->helperText('برش ۱۲۰۰×۶۳۰ خودکار ساخته می‌شود. خالی بماند، تصویر شاخص یا تصویر پیش‌فرض سایت استفاده می‌شود.')
                        ->live(),
                    OgPreview::make()
                        ->viewData(fn (Get $get): array => ['og' => $this->ogPreview($get)]),
                ]),
            Section::make('نمایه‌سازی و نقشه سایت')
                ->collapsible()
                ->collapsed()
                ->schema([
                    TextInput::make('canonical_url')
                        ->label('نشانی کنونیکال (اختیاری)')
                        ->url()
                        ->maxLength(2048)
                        ->live(onBlur: true)
                        ->helperText('فقط وقتی نسخه اصلی این محتوا نشانی دیگری دارد. خالی = نشانی همین صفحه.'),
                    Grid::make(2)->schema([
                        Toggle::make('robots_index')
                            ->label('در نتایج جست‌وجو نمایش داده شود (index)')
                            ->default(true)
                            ->live(),
                        Toggle::make('robots_follow')
                            ->label('پیوندهای صفحه دنبال شوند (follow)')
                            ->default(true),
                        Toggle::make('sitemap_include')
                            ->label('در نقشه سایت باشد')
                            ->default(true),
                        Select::make('sitemap_priority')
                            ->label('اولویت در نقشه سایت')
                            ->options(array_combine(self::SITEMAP_PRIORITIES, array_map(SerpMeasure::digits(...), self::SITEMAP_PRIORITIES)))
                            ->placeholder('پیش‌فرض'),
                    ]),
                ]),
        ];
    }

    /**
     * seo_meta.robots → the two toggles (extras such as max-image-preview stay in the stored string).
     *
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    public static function fillRobots(array $data): array
    {
        $robots = Robots::parse(is_string($data['robots'] ?? null) ? $data['robots'] : null);
        $data['robots_index'] = $robots->index;
        $data['robots_follow'] = $robots->follow;
        $data['sitemap_priority'] = is_numeric($data['sitemap_priority'] ?? null) ? number_format((float) $data['sitemap_priority'], 1, '.', '') : null;

        return $data;
    }

    /**
     * The toggles → seo_meta.robots: index,follow is the default and stored as null (inherit).
     *
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    public static function dehydrateRobots(array $data): array
    {
        $index = (bool) ($data['robots_index'] ?? true);
        $follow = (bool) ($data['robots_follow'] ?? true);
        unset($data['robots_index'], $data['robots_follow']);

        $data['robots'] = $index && $follow ? null : (string) new Robots($index, $follow);
        $data['sitemap_priority'] = is_numeric($data['sitemap_priority'] ?? null) ? (float) $data['sitemap_priority'] : null;
        $data['sitemap_include'] = (bool) ($data['sitemap_include'] ?? true);

        return $data;
    }

    private function fallbackTitle(): ?string
    {
        $title = $this->source($this->titleSource);

        return $title === '' ? null : $title;
    }

    private function fallbackDescription(): ?string
    {
        $text = $this->source($this->descriptionSource);

        return $text === '' ? null : DescriptionText::fromExcerpt($text);
    }

    /**
     * The <title> Google sees: page title (SEO title or fallback) through the site's title template.
     */
    private function fullTitle(?string $seoTitle): string
    {
        $title = trim((string) $seoTitle);
        if ($title === '' && $this->evaluate($this->fallbackTitleIsComplete) === true && $this->fallbackTitle() !== null) {
            return (string) $this->fallbackTitle();
        }

        return $this->settings()?->seo->title($title !== '' ? $title : $this->fallbackTitle()) ?? $title;
    }

    private function fullDescription(?string $description): string
    {
        $description = trim((string) $description);

        return $description !== '' ? $description : ($this->fallbackDescription() ?? $this->settings()?->seo->defaultDescription ?? '');
    }

    /**
     * @return array<string, mixed>
     */
    private function serpPreview(Get $get): array
    {
        $title = $this->fullTitle($get('title'));
        $description = $this->fullDescription($get('description'));
        $url = $this->pageUrl($get('canonical_url'));
        $parts = parse_url($url) ?: [];
        $path = array_values(array_filter(explode('/', trim(rawurldecode($parts['path'] ?? ''), '/'))));

        return [
            'site' => $this->settings()?->general->siteName ?? 'ریتمی',
            'host' => $parts['host'] ?? '',
            'breadcrumb' => implode(' › ', [($parts['scheme'] ?? 'https').'://'.($parts['host'] ?? ''), ...$path]),
            'desktopTitle' => SerpMeasure::truncate($title, SerpMeasure::TITLE_FONT_PX, SerpMeasure::TITLE_MAX_PX),
            'desktopDescription' => SerpMeasure::truncate($description, SerpMeasure::DESCRIPTION_FONT_PX, SerpMeasure::DESCRIPTION_MAX_PX),
            'mobileTitle' => SerpMeasure::truncate($title, SerpMeasure::TITLE_FONT_PX, SerpMeasure::TITLE_MAX_PX * 2 - 40),
            'mobileDescription' => SerpMeasure::truncate($description, SerpMeasure::DESCRIPTION_FONT_PX, SerpMeasure::MOBILE_DESCRIPTION_MAX_PX),
            'noindex' => $get('robots_index') === false,
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function ogPreview(Get $get): array
    {
        $ogTitle = trim((string) $get('og_title'));
        $ogDescription = trim((string) $get('og_description'));
        $title = $ogTitle !== '' ? $ogTitle : $this->fullTitle($get('title'));
        $description = $ogDescription !== '' ? $ogDescription : $this->fullDescription($get('description'));

        $mediaId = $get('og_media_id');
        if (! is_numeric($mediaId)) {
            $fallback = $this->source($this->imageSource);
            $mediaId = is_numeric($fallback) ? $fallback : $this->settings()?->seo->defaultOgMediaId;
        }

        $image = null;
        if (is_numeric($mediaId)) {
            try {
                $image = app(OgImageResolver::class)->resolve((int) $mediaId, $title);
            } catch (Throwable) {
                $image = null;
            }
        }

        return [
            'site' => $this->settings()?->general->siteName ?? 'ریتمی',
            'host' => (string) (parse_url($this->pageUrl($get('canonical_url')), PHP_URL_HOST) ?? ''),
            'title' => mb_strimwidth($title, 0, 90, '…'),
            'description' => mb_strimwidth($description, 0, 160, '…'),
            'image' => $image?->url,
        ];
    }

    private function analysis(Get $get): ?SeoAnalysis
    {
        try {
            $type = $this->contentType();
            $seoTitle = trim((string) $get('title'));
            $description = trim((string) $get('description'));
            $duplicates = $this->duplicates($seoTitle, $description);
            $url = $this->pageUrl($get('canonical_url'));

            return app(SeoAnalyzer::class)->analyze(new AnalysisInput(
                type: $type,
                title: $this->fullTitle($seoTitle),
                heading: $this->standalone ? '' : ($this->fallbackTitle() ?? ''),
                description: $this->fullDescription($description),
                slug: $this->slug($url),
                focusKeyword: trim((string) $get('focus_keyword')),
                contentHtml: $this->contentHtml($type),
                ownHosts: array_values(array_unique(array_filter([
                    (string) parse_url(url('/'), PHP_URL_HOST),
                    (string) parse_url($url, PHP_URL_HOST),
                ]))),
                cornerstone: $this->evaluate($this->cornerstone) === true,
                duplicateTitle: $duplicates['title'],
                duplicateDescription: $duplicates['description'],
            ));
        } catch (Throwable) {
            return null;
        }
    }

    private function contentType(): ContentType
    {
        $type = $this->evaluate($this->analysisType);
        if ($type instanceof ContentType) {
            return $type;
        }
        if ($this->standalone) {
            return ContentType::Page;
        }

        return match (class_basename((string) $this->getRootContainer()->getModel())) {
            'Post' => ContentType::Post,
            'Product' => ContentType::Product,
            'Place' => ContentType::Place,
            default => ContentType::Archive,
        };
    }

    /**
     * Body as HTML (rich-editor JSON rendered), null when the page has none to analyse.
     */
    private function contentHtml(ContentType $type): ?string
    {
        if ($this->contentSource instanceof Closure) {
            $raw = $this->evaluate($this->contentSource);
        } else {
            $raw = null;
            foreach ($this->contentSource !== null ? [$this->contentSource] : ($this->standalone ? [] : self::CONTENT_FIELDS) as $field) {
                $raw = $this->makeGetUtility()($field);
                if ($raw !== null && $raw !== '' && $raw !== []) {
                    break;
                }
            }
        }

        $html = match (true) {
            is_array($raw) => RichContentRenderer::make($raw)->toUnsafeHtml(), // analysed only, never rendered
            is_scalar($raw) => trim((string) $raw),
            default => '',
        };

        return $html === '' && in_array($type, [ContentType::Page, ContentType::Archive], true) ? null : $html;
    }

    private function slug(string $url): string
    {
        $slug = $this->standalone ? '' : $this->source('slug');
        if ($slug !== '') {
            return $slug;
        }

        $segments = explode('/', trim((string) parse_url($url, PHP_URL_PATH), '/'));

        return (string) end($segments);
    }

    /**
     * @return array{title: bool|null, description: bool|null}
     */
    private function duplicates(string $seoTitle, string $description): array
    {
        if ($this->standalone) {
            return ['title' => null, 'description' => null]; // the page's own row is keyed by route, unknown here
        }

        $existing = $this->getCachedExistingRecord();

        return app(FindDuplicateSeoMeta::class)->handle($seoTitle, $description, $existing instanceof SeoMeta ? $existing->id : null);
    }

    private function pageUrl(mixed $canonical): string
    {
        if (is_string($canonical) && trim($canonical) !== '') {
            return trim($canonical);
        }

        $url = $this->urlResolver === null ? null : $this->evaluate($this->urlResolver);

        return is_string($url) && $url !== '' ? $url : url('/');
    }

    /**
     * A parent-form value: field name, or a closure evaluated on this component (so `Get` reads the parent form).
     */
    private function source(string|Closure|null $source): string
    {
        if ($source === null) {
            return '';
        }

        $value = $source instanceof Closure ? $this->evaluate($source) : $this->makeGetUtility()($source);

        return is_scalar($value) ? trim((string) $value) : '';
    }

    private function settings(): ?SiteSettings
    {
        try {
            return app(SettingsRepository::class)->all();
        } catch (Throwable) {
            return null;
        }
    }
}
