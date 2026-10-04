<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Version;

use App\Domain\Pwa\Data\VersionInfo;
use App\Domain\Settings\Contracts\SettingsRepository;

/**
 * Two-tier update policy (L8-02): the deployed build (BuildInfo) + the admin's minimum build and message
 * (PwaSettings, cached settings repository). A minimum newer than the deployed build is capped to it, so a typo can
 * never lock every visitor behind the blocking screen.
 */
final class AppVersion
{
    public function __construct(
        private readonly BuildInfo $build,
        private readonly SettingsRepository $settings,
    ) {}

    public function current(): VersionInfo
    {
        $pwa = $this->settings->all()->pwa;
        $buildId = $this->build->buildId();
        $min = $pwa->minBuildId;

        if ($min !== null && ($buildId === null || self::stamp($min) > self::stamp($buildId))) {
            $min = $buildId;
        }

        return new VersionInfo($buildId, $min, $pwa->updateMessage);
    }

    public function deployedBuildId(): ?string
    {
        return $this->build->buildId();
    }

    /** The sortable 14-digit time of a build id (the sha suffix is informational). */
    public static function stamp(string $buildId): string
    {
        return substr($buildId, 0, 14);
    }
}
