<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * A typed settings group. fromArray() coerces stored JSON values (missing/invalid => default);
 * toArray() returns every key of the group (snake_case, as stored).
 */
interface SettingsGroupData
{
    public static function group(): SettingGroup;

    /**
     * @param  array<string, mixed>  $values
     */
    public static function fromArray(array $values): static;

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array;
}
