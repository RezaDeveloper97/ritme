<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Content\Enums\SiteSection;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Content\SitePhotos;
use App\Domain\Content\SiteSections;
use App\Domain\Content\Stages\Data\StageCardData;
use App\Domain\Content\Stages\Data\StageFeatureData;
use App\Domain\Content\Stages\Data\StageFloatCardData;
use App\Domain\Content\Stages\Data\StageHeroData;
use App\Domain\Content\Stages\Data\StagePageData;
use App\Domain\Content\Stages\Data\StageRedesignData;
use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Seo\Schema\Data\FaqItem;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\Translation\Translator;
use Illuminate\Routing\UrlGenerator;
use LogicException;

/**
 * Turns a StageDefinition (structure) + its lang file (copy) + live data (readings from the cached PostRepository,
 * app links / emergency number from the cached settings) into the StagePageData DTO the template renders.
 * Missing copy is a programming error and throws, so a new stage cannot ship with raw translation keys.
 */
final class StagePageBuilder
{
    public const READINGS = 3;

    private const COMMON = 'stages/common';

    private const HIGHLIGHT = ':highlight';

    public function __construct(
        private readonly Translator $translator,
        private readonly SiteNavigation $navigation,
        private readonly StageNavigation $stageNavigation,
        private readonly PostRepository $posts,
        private readonly SettingsRepository $settings,
        private readonly UrlGenerator $url,
        private readonly FaqRepository $faqs,
        private readonly SiteSections $sections,
        private readonly SitePhotos $photos,
    ) {}

    public function build(StageDefinition $definition): StagePageData
    {
        $group = $definition->langGroup();
        $settings = $this->settings->all();
        $common = fn (string $key): string => $this->text(self::COMMON, $key);

        return new StagePageData(
            stage: $definition->stage(),
            name: $this->text($group, 'name'),
            seoTitle: $this->text($group, 'seo.title'),
            seoDescription: $this->text($group, 'seo.description'),
            navLabel: $common('nav_label'),
            stagesLabel: $common('breadcrumb'),
            nav: $this->stageNavigation->items($definition->stage()),
            hero: $this->hero($definition),
            features: $this->features($definition),
            tools: array_map(fn (CardSpec $card): StageCardData => $this->card($group, 'tools', $card), $this->enabledCards($definition->tools())),
            help: array_map(fn (CardSpec $card): StageCardData => $this->card($group, 'help', $card, $common('help.cta')), $this->enabledCards($definition->help())),
            emergencyText: $this->text($group, 'emergency'),
            emergencyNumber: $settings->general->emergencyNumber,
            readings: $this->sections->enabled(SiteSection::Blog) ? $this->readings($definition->stage()) : [],
            readingsMoreUrl: $this->navigation->url(StaticPage::Blog),
            faqGroup: $definition->faqGroup(),
            faqTitle: $this->text($group, 'faq.title'),
            faq: $this->faq($group, $definition->faqGroup()),
            appCtaTitle: $this->text($group, 'app_cta.title'),
            appCtaLead: $this->optional($group, 'app_cta.lead'),
            appLinks: $settings->appLinks,
            qrUrl: $settings->appLinks->webApp ?? $this->navigation->url(StaticPage::Home, 'download', absolute: true),
            copy: [
                'toolsEyebrow' => $common('tools.eyebrow'),
                'toolsTitle' => $common('tools.title'),
                'helpEyebrow' => $common('help.eyebrow'),
                'helpTitle' => $common('help.title'),
                'readingsEyebrow' => $common('readings.eyebrow'),
                'readingsTitle' => $common('readings.title'),
                'readingsMore' => $common('readings.more'),
                'faqEyebrow' => $common('faq.eyebrow'),
            ],
            redesign: ($spec = $definition->redesign()) === null ? null : $this->redesign($group, $spec),
        );
    }

    private function redesign(string $group, RedesignSpec $spec): StageRedesignData
    {
        $common = fn (string $key): string => $this->text(self::COMMON, "redesign.{$key}");
        $feature = fn (string $key): array => [
            'title' => $this->text($group, "redesign.features.items.{$key}.title"),
            'text' => $this->text($group, "redesign.features.items.{$key}.text"),
        ];

        return new StageRedesignData(
            accent: $spec->accent,
            eyebrow: $this->text($group, 'redesign.hero.eyebrow'),
            title: $this->text($group, 'redesign.hero.title'),
            lead: $this->text($group, 'redesign.hero.lead'),
            chips: array_map(fn (string $chip): array => [
                'title' => $this->text($group, "redesign.hero.chips.{$chip}.title"),
                'text' => $this->text($group, "redesign.hero.chips.{$chip}.text"),
            ], ['top', 'bottom']),
            heroPhoto: $this->photos->get($spec->heroPhoto),
            heroAlt: $spec->heroPhoto->alt(),
            featurePhoto: $this->photos->get($spec->featurePhoto),
            featureAlt: $spec->featurePhoto->alt(),
            featuresTitle: $this->text($group, 'redesign.features.title'),
            features: array_map($feature, $spec->featureKeys),
            copy: [
                'download' => $this->text(self::COMMON, 'hero.download'),
                'see' => $common('see'),
                'featuresEyebrow' => $common('features_eyebrow'),
                'toolsEyebrow' => $this->text(self::COMMON, 'tools.eyebrow'),
                'toolsTitle' => $this->text(self::COMMON, 'tools.title'),
                'readingsMore' => $common('readings_more'),
                'faqTitle' => $common('faq_title'),
                'downloadTitle' => $common('download_title'),
                'downloadLead' => $common('download_lead'),
            ],
        );
    }

    private function hero(StageDefinition $definition): StageHeroData
    {
        $group = $definition->langGroup();
        [$titleBefore, $titleAfter] = $this->splitTitle($this->text($group, 'hero.title'));

        return new StageHeroData(
            eyebrow: $this->text($group, 'hero.eyebrow'),
            eyebrowIcon: $definition->eyebrowIcon(),
            titleBefore: $titleBefore,
            highlight: $this->text($group, 'hero.highlight'),
            titleAfter: $titleAfter,
            lead: $this->text($group, 'hero.lead'),
            downloadLabel: $this->text(self::COMMON, 'hero.download'),
            howLabel: $this->text(self::COMMON, 'hero.how'),
            screen: $definition->heroScreen(),
            screenData: $this->screenCopy($group, $definition->heroScreen()),
            floatCards: array_map(fn (FloatCardSpec $card): StageFloatCardData => new StageFloatCardData(
                icon: $card->icon,
                tint: $card->tint,
                title: $this->text($group, "hero.float.{$card->key}.title"),
                text: $this->text($group, "hero.float.{$card->key}.text"),
                position: $card->position,
            ), $definition->floatCards()),
        );
    }

    /**
     * @return list<StageFeatureData>
     */
    private function features(StageDefinition $definition): array
    {
        $group = $definition->langGroup();
        $features = [];
        foreach ($definition->features() as $index => $spec) {
            $prefix = "features.{$spec->key}";
            $title = $this->text($group, "{$prefix}.title");
            $eyebrow = $this->text($group, "{$prefix}.eyebrow");
            $points = $this->list($group, "{$prefix}.points");

            $features[] = new StageFeatureData(
                key: $spec->key,
                anchor: $index === 0 ? 'how' : null,
                eyebrow: $eyebrow,
                title: $title,
                text: $this->text($group, "{$prefix}.text"),
                points: $points,
                partial: $definition->featurePartial($spec),
                screen: $spec->screen,
                screenData: $spec->screen === 'checklist'
                    ? [
                        'eyebrow' => $eyebrow,
                        'title' => $this->optional($group, "{$prefix}.screen_title") ?? $title,
                        'items' => $points,
                        'tint' => $spec->tint,
                        'cta' => $this->text(self::COMMON, 'mock.continue'),
                    ]
                    : $this->screenCopy($group, $spec->screen),
                mediaFirst: $index % 2 === 1,
                surface: $index % 2 === 1,
            );
        }

        return $features;
    }

    /**
     * Cards whose link target is not in a site section switched off in the admin «صفحه‌ها» settings.
     *
     * @param  list<CardSpec>  $cards
     * @return list<CardSpec>
     */
    private function enabledCards(array $cards): array
    {
        return array_values(array_filter($cards, fn (CardSpec $card): bool => $this->sections->routeEnabled($card->route)));
    }

    private function card(string $group, string $section, CardSpec $spec, ?string $cta = null): StageCardData
    {
        return new StageCardData(
            href: $this->url->route($spec->route, [], false).($spec->fragment !== '' ? '#'.$spec->fragment : ''),
            icon: $spec->icon,
            color: $spec->color,
            title: $this->text($group, "{$section}.{$spec->key}.title"),
            text: $this->optional($group, "{$section}.{$spec->key}.text"),
            where: $spec->where === null ? null : $this->text(self::COMMON, "tools.{$spec->where}"),
            cta: $cta,
            onSite: $spec->where === CardSpec::ON_SITE,
        );
    }

    /**
     * Posts of the stage first, topped up with the newest posts of the magazine (the design mixes stages) so the
     * block is full while a stage has few articles. Both lists come from the cached repository (`blog` ns).
     *
     * @return list<PostCardData>
     */
    private function readings(LifeStage $stage): array
    {
        $posts = $this->posts->latest(1, self::READINGS, $stage)->items;
        if (count($posts) < self::READINGS) {
            $ids = array_map(static fn (PostCardData $post): int => $post->id, $posts);
            foreach ($this->posts->latest(1, self::READINGS * 2)->items as $post) {
                if (count($posts) >= self::READINGS) {
                    break;
                }
                if (! in_array($post->id, $ids, true)) {
                    $posts[] = $post;
                    $ids[] = $post->id;
                }
            }
        }

        return $posts;
    }

    /**
     * Published items of the stage's FAQ group (`stage-<slug>`, cached `faq` ns, admin-edited); the lang copy only
     * while the group is not seeded. A seeded group with nothing published shows no FAQ.
     *
     * @return list<FaqItem>
     */
    private function faq(string $group, string $faqGroup): array
    {
        $seeded = $this->faqs->group($faqGroup);
        if ($seeded !== null) {
            return $seeded->schemaItems();
        }

        $items = $this->translator->get("{$group}.faq.items");
        if (! is_array($items)) {
            throw new LogicException("Missing translation [{$group}.faq.items].");
        }

        $faq = [];
        foreach ($items as $item) {
            if (is_array($item) && is_string($item['question'] ?? null) && is_string($item['answer'] ?? null)) {
                $faq[] = new FaqItem($item['question'], $item['answer']);
            }
        }

        return $faq;
    }

    /**
     * `hero.title` carries a `:highlight` placeholder where the lilac phrase sits in the design's h1; without one the
     * phrase leads the title. Returns the text before and after the phrase (spacing as written in the lang file).
     *
     * @return array{0: string, 1: string}
     */
    private function splitTitle(string $title): array
    {
        if (! str_contains($title, self::HIGHLIGHT)) {
            return ['', ' '.$title];
        }

        [$before, $after] = explode(self::HIGHLIGHT, $title, 2);

        return [$before, $after];
    }

    /**
     * Copy of a mock screen: the stage's own `<stage>.mock.<screen>` first, then the shared
     * `stages/common.mock.<screen>` (dashes → underscores). Values may be nested lists (stats, rows). The phone
     * partial keys its fragment cache by this array, so lang edits show up without a cache bump.
     *
     * @return array<string, mixed>
     */
    private function screenCopy(string $group, string $screen): array
    {
        $key = 'mock.'.str_replace('-', '_', $screen);
        foreach ([$group, self::COMMON] as $source) {
            $copy = $this->translator->get("{$source}.{$key}");
            if (is_array($copy)) {
                return array_filter($copy, static fn (mixed $value, mixed $name): bool => is_string($name) && (is_string($value) || is_array($value)), ARRAY_FILTER_USE_BOTH);
            }
        }

        return [];
    }

    private function text(string $group, string $key): string
    {
        return $this->optional($group, $key) ?? throw new LogicException("Missing translation [{$group}.{$key}].");
    }

    private function optional(string $group, string $key): ?string
    {
        $value = $this->translator->get("{$group}.{$key}");

        return is_string($value) && $value !== "{$group}.{$key}" && trim($value) !== '' ? $value : null;
    }

    /**
     * @return list<string>
     */
    private function list(string $group, string $key): array
    {
        $value = $this->translator->get("{$group}.{$key}");

        return is_array($value) ? array_values(array_filter($value, 'is_string')) : [];
    }
}
