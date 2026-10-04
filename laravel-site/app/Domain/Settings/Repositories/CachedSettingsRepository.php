<?php

declare(strict_types=1);

namespace App\Domain\Settings\Repositories;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SettingsGroupData;
use App\Domain\Settings\Data\SiteSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * Caches every group as one plain array, forever, in the `settings` namespace (SettingObserver bumps it).
 * Arrays rather than DTOs are cached so a deploy that changes a DTO never unserializes a stale object.
 */
final class CachedSettingsRepository extends CachedRepository implements SettingsRepository
{
    public function __construct(private readonly SettingsRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'settings';
    }

    public function all(): SiteSettings
    {
        /** @var array<string, array<string, mixed>> $values */
        $values = $this->rememberForever('all', fn (): array => $this->inner->all()->toArray());

        return SiteSettings::fromArray($values);
    }

    public function group(SettingGroup $group): SettingsGroupData
    {
        return $this->all()->group($group);
    }

    public function get(SettingGroup $group, string $key): mixed
    {
        $group->assertKey($key);

        return $this->all()->get($group, $key);
    }

    public function put(SettingGroup $group, array $values): void
    {
        $this->inner->put($group, $values);
    }
}
