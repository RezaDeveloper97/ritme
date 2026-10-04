<?php

declare(strict_types=1);

namespace Tests\Unit\Seo\Schema;

use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Data\SeoHead;
use App\Domain\Seo\Data\SeoImage;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SettingsGroupData;
use App\Domain\Settings\Data\SiteSettings;
use App\Domain\Settings\Enums\SettingGroup;
use Illuminate\Config\Repository as Config;
use LogicException;

/**
 * In-memory settings + resolver so the builders can be tested without the container or a database.
 */
final class SchemaFixtures
{
    public const SITE = 'https://ritme.test';

    /**
     * @param  array<string, array<string, mixed>>  $overrides  group => key => value
     */
    public static function settings(array $overrides = []): SiteSettings
    {
        return SiteSettings::fromArray(array_replace_recursive([
            'general' => ['site_name' => 'ریتمی', 'alternate_name' => 'Ritme'],
            'contact' => ['support_email' => 'support@ritme.test', 'phone' => '021-0000'],
            'social' => ['instagram' => 'https://instagram.com/ritme', 'telegram' => null, 'linkedin' => 'https://linkedin.com/company/ritme'],
            'app_links' => ['bazaar' => 'https://cafebazaar.ir/app/ir.ritmeapp.ritme', 'web_app' => 'https://web.ritme.test'],
            'seo' => ['title_template' => '%s — ریتمی', 'default_description' => 'همراه سلامت زنان.'],
            'organization' => [
                'legal_name' => 'ریتمی', 'logo_media_id' => 7, 'founding_date' => '2025',
                'same_as' => ['https://instagram.com/ritme', 'https://aparat.com/ritme'],
                'contact_point' => ['contact_type' => 'customer support', 'telephone' => null, 'email' => null],
            ],
            'pwa' => ['description' => 'همراه سلامت زنان، از اولین پریود تا یائسگی'],
        ], $overrides));
    }

    public static function logo(): SeoImage
    {
        return new SeoImage(self::SITE.'/media/logo.png', 512, 512, 'ریتمی', 'image/png');
    }

    /**
     * @param  array<string, mixed>  $schemaOverrides
     */
    public static function head(string $canonical = self::SITE.'/cycle', string $title = 'پیگیری چرخه — ریتمی', array $schemaOverrides = []): SeoHead
    {
        return new SeoHead(
            title: $title,
            description: 'چرخه‌ات را بشناس.',
            canonical: $canonical,
            robots: 'index,follow',
            indexable: true,
            openGraph: [],
            twitter: [],
            verification: [],
            feeds: [],
            image: new SeoImage($canonical.'.jpg', 1200, 630, 'چرخه'),
            schemaOverrides: $schemaOverrides,
        );
    }

    public static function pageGraph(SchemaGraph $graph, ?SiteSettings $settings = null): PageGraph
    {
        $settings ??= self::settings();

        $repository = new class($settings) implements SettingsRepository
        {
            public function __construct(private readonly SiteSettings $settings) {}

            public function all(): SiteSettings
            {
                return $this->settings;
            }

            public function group(SettingGroup $group): SettingsGroupData
            {
                return $this->settings->group($group);
            }

            public function get(SettingGroup $group, string $key): mixed
            {
                return $this->settings->get($group, $key);
            }

            public function put(SettingGroup $group, array $values): void
            {
                throw new LogicException('read-only fixture');
            }
        };

        $images = new class implements OgImageResolver
        {
            public function resolve(int $mediaId, string $alt): ?SeoImage
            {
                return $mediaId === 7 ? SchemaFixtures::logo() : null;
            }
        };

        return new PageGraph($graph, $repository, $images, new Config(['app' => ['url' => self::SITE]]));
    }

    /**
     * Every node of a rendered graph by `@id`.
     *
     * @return array<string, array<string, mixed>>
     */
    public static function byId(SchemaGraph $graph): array
    {
        $decoded = json_decode($graph->toJson(), true, 512, JSON_THROW_ON_ERROR);
        $nodes = [];
        foreach ($decoded['@graph'] as $node) {
            $nodes[$node['@id']] = $node;
        }

        return $nodes;
    }
}
