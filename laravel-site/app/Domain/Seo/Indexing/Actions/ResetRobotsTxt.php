<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Actions;

use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;

/** Back to the built-in production rules (DefaultRobotsRules): clears `seo.robots_txt`. */
final class ResetRobotsTxt
{
    public function __construct(private readonly UpdateSettings $update) {}

    public function handle(): void
    {
        $this->update->handle(SettingGroup::Seo, ['robots_txt' => null]);
    }
}
