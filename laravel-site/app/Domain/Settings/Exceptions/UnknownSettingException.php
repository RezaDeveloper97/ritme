<?php

declare(strict_types=1);

namespace App\Domain\Settings\Exceptions;

use App\Domain\Settings\Enums\SettingGroup;
use InvalidArgumentException;

final class UnknownSettingException extends InvalidArgumentException
{
    public static function group(string $group): self
    {
        return new self("Unknown settings group [{$group}].");
    }

    public static function key(SettingGroup $group, string $key): self
    {
        return new self("Unknown setting [{$group->value}.{$key}].");
    }
}
