<?php

declare(strict_types=1);

namespace App\Domain\Settings\Contracts;

use App\Domain\Settings\Data\SettingsGroupData;
use App\Domain\Settings\Data\SiteSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Exceptions\UnknownSettingException;

interface SettingsRepository
{
    /**
     * Every group at once (one query; zero when cached).
     */
    public function all(): SiteSettings;

    public function group(SettingGroup $group): SettingsGroupData;

    /**
     * @throws UnknownSettingException
     */
    public function get(SettingGroup $group, string $key): mixed;

    /**
     * Stores the given keys of one group (values are coerced through the group's DTO).
     *
     * @param  array<string, mixed>  $values
     *
     * @throws UnknownSettingException
     */
    public function put(SettingGroup $group, array $values): void;
}
