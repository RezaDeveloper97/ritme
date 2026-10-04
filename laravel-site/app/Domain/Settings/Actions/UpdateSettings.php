<?php

declare(strict_types=1);

namespace App\Domain\Settings\Actions;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SettingsGroupData;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Exceptions\UnknownSettingException;

/**
 * Updates keys of one settings group (used by the admin settings pages). Cache invalidation happens in
 * SettingObserver after the write commits.
 */
final class UpdateSettings
{
    public function __construct(private readonly SettingsRepository $settings) {}

    /**
     * @param  array<string, mixed>  $values
     *
     * @throws UnknownSettingException
     */
    public function handle(SettingGroup $group, array $values): SettingsGroupData
    {
        $this->settings->put($group, $values);

        return $this->settings->group($group);
    }
}
