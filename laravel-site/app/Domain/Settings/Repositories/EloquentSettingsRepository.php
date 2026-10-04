<?php

declare(strict_types=1);

namespace App\Domain\Settings\Repositories;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SettingsGroupData;
use App\Domain\Settings\Data\SiteSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Models\Setting;

final class EloquentSettingsRepository implements SettingsRepository
{
    public function all(): SiteSettings
    {
        $values = [];
        foreach (Setting::query()->get(['group', 'key', 'value']) as $setting) {
            $values[$setting->group][$setting->key] = $setting->value;
        }

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
        foreach (array_keys($values) as $key) {
            $group->assertKey((string) $key);
        }

        // Coerce through the DTO, keep only the keys being written.
        $normalized = array_intersect_key($group->data($values)->toArray(), $values);

        Setting::query()->getConnection()->transaction(static function () use ($group, $normalized): void {
            foreach ($normalized as $key => $value) {
                Setting::query()->updateOrCreate(['group' => $group->value, 'key' => $key], ['value' => $value]);
            }
        });
    }
}
