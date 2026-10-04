<?php

declare(strict_types=1);

namespace App\Domain\Seo;

use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Data\PageContext;
use App\Domain\Seo\Data\SeoHead;
use App\Domain\Seo\Data\SeoImage;
use App\Domain\Seo\Data\SeoMetaData;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Seo\Support\DescriptionText;
use App\Domain\Seo\Support\Robots;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Routing\Router;
use Illuminate\Routing\UrlGenerator;
use Illuminate\Support\Str;

/**
 * Decides every SEO head tag of the current request (request-scoped; see SeoServiceProvider).
 *
 * Resolution, later layers win field by field:
 *   1. defaults from settings (SeoDefaults, site name)
 *   2. seo_meta of the static page (by route name)
 *   3. seo_meta of the model handed to for()
 *   4. controller overrides (title(), description(), canonical(), robots(), image() …)
 *
 * Then the forced rules: noindex outside production, on search / cart / checkout / done pages and on filtered
 * listings (any query parameter that is not `page`, tracking or allow-listed via keepQuery()).
 *
 *     public function show(Post $post, SeoManager $seo) {
 *         $seo->for($post)->title($post->title)->excerpt($post->excerpt)->type('article');
 *         …
 *     }
 */
final class SeoManager
{
    /** Route-name patterns (Str::is) that are never indexable. */
    public const NOINDEX_ROUTES = [
        'search', '*.search',
        'shop.cart', 'shop.checkout', 'shop.checkout.*', 'shop.order', 'shop.done',
        '*.done', 'directory.booked',
    ];

    /** Query parameters that never make a URL a different page (dropped from canonicals, ignored as filters). */
    public const TRACKING_PARAMS = [
        'utm_*', 'gclid', 'gbraid', 'wbraid', 'dclid', 'fbclid', 'msclkid', 'yclid', 'twclid', 'igshid', 'srsltid',
        '_ga', '_gl', 'mc_cid', 'mc_eid', 'ref', 'source',
    ];

    /** Verification setting key => meta name. Unknown keys are used as the meta name verbatim. */
    public const VERIFICATION_METAS = [
        'google' => 'google-site-verification',
        'bing' => 'msvalidate.01',
        'yandex' => 'yandex-verification',
        'pinterest' => 'p:domain_verify',
        'facebook' => 'facebook-domain-verification',
    ];

    public const TWITTER_CARDS = ['summary', 'summary_large_image'];

    /** Named route of the magazine RSS feed; the alternate link appears once it exists (L4-04). */
    public const FEED_ROUTE = 'blog.feed';

    private ?PageContext $context = null;

    private ?SeoMetaData $modelMeta = null;

    private ?string $title = null;

    private bool $rawTitle = false;

    private ?string $description = null;

    private ?string $excerpt = null;

    private ?string $canonical = null;

    private ?Robots $robots = null;

    private bool $filtered = false;

    private ?SeoImage $image = null;

    private ?string $type = null;

    /** @var list<string> */
    private array $keepQuery = [];

    public function __construct(
        private readonly SettingsRepository $settings,
        private readonly SeoMetaRepository $meta,
        private readonly OgImageResolver $images,
        private readonly Router $router,
        private readonly UrlGenerator $url,
        private readonly Config $config,
    ) {}

    /**
     * Uses the model's seo_meta row (HasSeo) as the page layer.
     */
    public function for(Model $model): self
    {
        $key = $model->getKey();
        $this->modelMeta = is_int($key) || is_string($key) ? $this->meta->forModel($model->getMorphClass(), $key) : null;

        return $this;
    }

    /**
     * Page title; the site title template is applied («%s — ریتمی»).
     */
    public function title(string $title): self
    {
        $this->title = $title;
        $this->rawTitle = false;

        return $this;
    }

    /**
     * Complete title, used verbatim (no template).
     */
    public function rawTitle(string $title): self
    {
        $this->title = $title;
        $this->rawTitle = true;

        return $this;
    }

    public function description(string $description): self
    {
        $this->description = $description;

        return $this;
    }

    /**
     * Source for the description when neither an override nor seo_meta provides one (trimmed to ~155 chars).
     */
    public function excerpt(string $excerpt): self
    {
        $this->excerpt = $excerpt;

        return $this;
    }

    public function canonical(string $url): self
    {
        $this->canonical = $url;

        return $this;
    }

    public function robots(string|Robots $robots): self
    {
        $this->robots = is_string($robots) ? Robots::parse($robots) : $robots;

        return $this;
    }

    public function noindex(): self
    {
        $this->robots = ($this->robots ?? Robots::default())->withNoindex();

        return $this;
    }

    /**
     * Marks the page as a filtered listing (noindex,follow) when filters are not plain query parameters.
     */
    public function filtered(bool $filtered = true): self
    {
        $this->filtered = $filtered;

        return $this;
    }

    public function image(SeoImage $image): self
    {
        $this->image = $image;

        return $this;
    }

    /**
     * og:type (website, article, product…).
     */
    public function type(string $type): self
    {
        $this->type = $type;

        return $this;
    }

    /**
     * Query parameters that make a distinct, indexable page (kept in the canonical, not treated as filters).
     */
    public function keepQuery(string ...$params): self
    {
        $this->keepQuery = array_values(array_unique([...$this->keepQuery, ...$params]));

        return $this;
    }

    /**
     * Overrides the page context (defaults to the current request's URL and route name).
     */
    public function context(PageContext $context): self
    {
        $this->context = $context;

        return $this;
    }

    public function resolve(): SeoHead
    {
        $site = $this->settings->all();
        $defaults = $site->seo;
        $context = $this->context ?? new PageContext($this->url->full(), $this->router->currentRouteName());
        $routeMeta = $context->routeName === null ? null : $this->meta->forRoute($context->routeName);
        $layers = array_values(array_filter([$routeMeta, $this->modelMeta]));
        $pick = static function (string $field) use ($layers): mixed {
            $value = null;
            foreach ($layers as $layer) {
                $value = $layer->{$field} ?? $value;
            }

            return $value;
        };

        $title = match (true) {
            $this->title !== null && $this->rawTitle => trim($this->title),
            $this->title !== null => $defaults->title($this->title),
            default => $defaults->title(is_string($pick('title')) ? $pick('title') : null),
        };

        $description = $this->description
            ?? (is_string($pick('description')) ? $pick('description') : null)
            ?? ($this->excerpt !== null && trim($this->excerpt) !== '' ? DescriptionText::fromExcerpt($this->excerpt) : null)
            ?? $defaults->defaultDescription;
        $description = trim($description);

        $explicitCanonical = $this->canonical ?? (is_string($pick('canonicalUrl')) ? $pick('canonicalUrl') : null);
        $canonical = $explicitCanonical !== null && $this->isForeign($explicitCanonical)
            ? $explicitCanonical // e.g. syndicated content pointing at its source
            : CanonicalUrl::normalize($explicitCanonical ?? $context->url, (string) $this->config->get('app.url'), $this->keepQuery);

        $storedRobots = $pick('robots');
        $robots = $this->robots ?? (is_string($storedRobots) ? Robots::parse($storedRobots) : Robots::default());
        if ($this->forcedNoindex($context)) {
            $robots = $robots->withNoindex();
        }

        $ogTitle = is_string($pick('ogTitle')) ? $pick('ogTitle') : $title;
        $ogDescription = is_string($pick('ogDescription')) ? $pick('ogDescription') : $description;
        $mediaId = $pick('ogMediaId') ?? $defaults->defaultOgMediaId;
        $image = $this->image ?? (is_int($mediaId) ? $this->images->resolve($mediaId, $ogTitle) : null);

        $openGraph = [
            'og:locale' => 'fa_IR',
            'og:site_name' => $site->general->siteName,
            'og:type' => $this->type ?? (is_string($pick('ogType')) ? $pick('ogType') : 'website'),
            'og:url' => $canonical,
            'og:title' => $ogTitle,
            'og:description' => $ogDescription,
        ];
        if ($image !== null) {
            $openGraph += [
                'og:image' => $image->url,
                'og:image:secure_url' => $image->url,
                'og:image:type' => $image->type,
                'og:image:width' => (string) $image->width,
                'og:image:height' => (string) $image->height,
                'og:image:alt' => $image->alt,
            ];
        }

        $card = $pick('twitterCard');
        $twitter = [
            'twitter:card' => in_array($card, self::TWITTER_CARDS, true) ? $card : ($image !== null ? 'summary_large_image' : 'summary'),
        ];
        if ($defaults->twitterHandle !== null) {
            $handle = '@'.ltrim($defaults->twitterHandle, '@');
            $twitter += ['twitter:site' => $handle];
        }
        $twitter += ['twitter:title' => $ogTitle, 'twitter:description' => $ogDescription];
        if ($image !== null) {
            $twitter += ['twitter:image' => $image->url, 'twitter:image:alt' => $image->alt];
        }

        $verification = [];
        foreach ($defaults->verification as $engine => $token) {
            if (trim($token) !== '') {
                $verification[self::VERIFICATION_METAS[$engine] ?? $engine] = trim($token);
            }
        }

        $feeds = $this->router->has(self::FEED_ROUTE)
            ? [['href' => $this->url->route(self::FEED_ROUTE), 'title' => 'مجله '.$site->general->siteName]]
            : [];

        $schemaOverrides = $pick('schemaOverrides');

        return new SeoHead(
            title: $title,
            description: $description,
            canonical: $canonical,
            robots: (string) $robots,
            indexable: $robots->index,
            openGraph: $openGraph,
            twitter: $twitter,
            verification: $verification,
            feeds: $feeds,
            image: $image,
            schemaOverrides: is_array($schemaOverrides) ? $schemaOverrides : [],
        );
    }

    /**
     * An explicit canonical (admin / controller) to another host is kept as given; everything else — including the
     * current request URL, whatever host it arrived on — is normalised onto the configured origin.
     */
    private function isForeign(string $url): bool
    {
        $host = parse_url($url, PHP_URL_HOST);

        return is_string($host) && strcasecmp($host, (string) parse_url((string) $this->config->get('app.url'), PHP_URL_HOST)) !== 0;
    }

    private function forcedNoindex(PageContext $context): bool
    {
        if ($this->config->get('app.env') !== 'production' || $this->filtered) {
            return true;
        }

        if ($context->routeName !== null && Str::is(self::NOINDEX_ROUTES, $context->routeName)) {
            return true;
        }

        parse_str((string) parse_url($context->url, PHP_URL_QUERY), $query);
        foreach (array_keys($query) as $param) {
            $param = (string) $param;
            if ($param !== 'page' && ! in_array($param, $this->keepQuery, true) && ! Str::is(self::TRACKING_PARAMS, $param)) {
                return true; // filtered / sorted / searched listing
            }
        }

        return false;
    }
}
