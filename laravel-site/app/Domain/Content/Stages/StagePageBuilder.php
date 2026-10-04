<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Content\Stages\Data\StageCardData;
use App\Domain\Content\Stages\Data\StageFeatureData;
use App\Domain\Content\Stages\Data\StageFloatCardData;
use App\Domain\Content\Stages\Data\StageHeroData;
use App\Domain\Content\Stages\Data\StagePageData;
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

    public function __construct(
        private readonly Translator $translator,
        private readonly SiteNavigation $navigation,
        private readonly StageNavigation $stageNavigation,
        private readonly PostRepository $posts,
        private readonly SettingsRepository $settings,
        private readonly UrlGenerator $url,
        private readonly FaqRepository $faqs,
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
            tools: array_map(fn (CardSpec $card): StageCardData => $this->card($group, 'tools', $card), $definition->tools()),
            help: array_map(fn (CardSpec $card): StageCardData => $this->card($group, 'help', $card, $common('help.cta')), $definition->help()),
            emergencyText: $this->text($group, 'emergency'),
            emergencyNumber: $settings->general->emergencyNumber,
            readings: $this->readings($definition->stage()),
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
        );
    }

    private function hero(StageDefinition $definition): StageHeroData
    {
        $group = $definition->langGroup();

        return new StageHeroData(
            eyebrow: $this->text($group, 'hero.eyebrow'),
            eyebrowIcon: $definition->eyebrowIcon(),
            highlight: $this->text($group, 'hero.highlight'),
            title: $this->text($group, 'hero.title'),
            lead: $this->text($group, 'hero.lead'),
            downloadLabel: $this->text(self::COMMON, 'hero.download'),
            howLabel: $this->text(self::COMMON, 'hero.how'),
            screen: $definition->heroScreen(),
            screenData: $this->screenCopy($definition->heroScreen()),
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
                    : $this->screenCopy($spec->screen),
                mediaFirst: $index % 2 === 1,
                surface: $index % 2 === 1,
            );
        }

        return $features;
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
     * Copy of a shared mock screen (`stages/common.mock.<screen>`), keys as strings.
     *
     * @return array<string, string>
     */
    private function screenCopy(string $screen): array
    {
        $copy = $this->translator->get(self::COMMON.'.mock.'.str_replace('-', '_', $screen));

        return is_array($copy) ? array_filter($copy, 'is_string') : [];
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
