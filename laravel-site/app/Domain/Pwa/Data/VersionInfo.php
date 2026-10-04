<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Data;

/**
 * `/pwa/version.json` (L8-02) — the two-tier update source of truth read by the `pwa` data-module:
 * `build_id` = deployed build (soft tier when newer than the running one), `min_build_id` = oldest build still
 * allowed (forced tier when the running one is older), `message` = optional admin text for the update UI.
 */
final readonly class VersionInfo
{
    public function __construct(
        public ?string $buildId,
        public ?string $minBuildId,
        public string $message,
    ) {}

    /**
     * @return array{build_id: string|null, min_build_id: string|null, message: string}
     */
    public function toArray(): array
    {
        return [
            'build_id' => $this->buildId,
            'min_build_id' => $this->minBuildId,
            'message' => $this->message,
        ];
    }
}
