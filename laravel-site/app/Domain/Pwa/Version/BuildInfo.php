<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Version;

use App\Domain\Settings\Data\PwaSettings;

/**
 * The deployed front-end build (L8-02): `public/build/build-id.json`, written by every `vite build` (vite.config.js)
 * and stamped into the generated `public/sw.js` by tools/build-sw.mjs. Null when no build is present (tests, a fresh
 * checkout) — the version endpoint then reports no build and no client is ever prompted.
 */
final class BuildInfo
{
    private ?string $resolved = null;

    private bool $read = false;

    private readonly string $file;

    /** @param  string|null  $file  defaults to public/build/build-id.json */
    public function __construct(?string $file = null)
    {
        $this->file = $file ?? public_path('build/build-id.json');
    }

    public function buildId(): ?string
    {
        if ($this->read) {
            return $this->resolved;
        }
        $this->read = true;

        $json = is_file($this->file) ? @file_get_contents($this->file) : false;
        $data = is_string($json) ? json_decode($json, true) : null;
        $id = is_array($data) && is_string($data['build_id'] ?? null) ? $data['build_id'] : null;

        return $this->resolved = $id !== null && preg_match(PwaSettings::BUILD_ID_PATTERN, $id) === 1 ? strtolower($id) : null;
    }
}
