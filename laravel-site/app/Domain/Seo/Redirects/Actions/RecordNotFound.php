<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Enums\AgentClass;
use App\Domain\Seo\Redirects\Support\HitBuffer;
use App\Domain\Seo\Redirects\Support\RedirectPath;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Notes one 404 in the cache buffer (no database write; FlushRedirectStats aggregates it into `not_found_logs`).
 *
 * Ignored: crawlers / scripts (unless `seo.not_found.log_bots` is true), static assets and media, admin / Livewire
 * paths, well-known scanner probes (wp-admin, xmlrpc, dotfiles …) and absurdly long paths. Kept: the decoded path,
 * the referer without its query string (no personal data travels in it) and the agent class. No IP, no full UA.
 */
final class RecordNotFound
{
    public const BUCKET = 'not-found';

    private const ASSET_EXTENSIONS = [
        'js', 'mjs', 'css', 'map', 'json', 'png', 'jpg', 'jpeg', 'gif', 'webp', 'avif', 'svg', 'ico', 'bmp', 'tif', 'tiff',
        'woff', 'woff2', 'ttf', 'otf', 'eot', 'mp4', 'webm', 'mp3', 'ogg', 'wav', 'pdf', 'zip', 'gz', 'rar', 'webmanifest',
        'php', 'asp', 'aspx', 'jsp', 'cgi', 'env', 'ini', 'log', 'bak', 'sql', 'yml', 'yaml', 'xml', 'txt',
    ];

    private const IGNORED_PREFIXES = [
        '/build/', '/media/', '/storage/', '/vendor/', '/livewire', '/filament', '/css/filament', '/js/filament',
        '/fonts/', '/icons/', '/.', '/wp-admin', '/wp-login', '/wp-includes', '/wp-json', '/wp-content/plugins',
        '/wp-content/themes', '/xmlrpc', '/cgi-bin', '/phpmyadmin', '/sw.js', '/apple-touch-icon', '/favicon',
    ];

    public function __construct(private readonly HitBuffer $buffer, private readonly Config $config) {}

    public function handle(string $path, ?string $referer, ?string $userAgent): bool
    {
        $agent = AgentClass::fromUserAgent($userAgent);
        if ($agent === AgentClass::Bot && ! (bool) $this->config->get('seo.not_found.log_bots', false)) {
            return false;
        }

        $path = RedirectPath::normalize($path);
        if ($this->ignored($path)) {
            return false;
        }

        return $this->buffer->hit(self::BUCKET, RedirectPath::hash($path), [
            'path' => $path,
            'referer' => $this->referer($referer),
            'agent' => $agent->value,
        ]);
    }

    private function ignored(string $path): bool
    {
        if ($path === '/' || mb_strlen($path) > RedirectPath::MAX_LENGTH) {
            return true;
        }

        $lower = mb_strtolower($path, 'UTF-8');
        $admin = '/'.trim((string) $this->config->get('filament.admin.path', 'admin'), '/');
        if ($admin !== '/' && ($lower === $admin || str_starts_with($lower, $admin.'/'))) {
            return true;
        }

        foreach (self::IGNORED_PREFIXES as $prefix) {
            if (str_starts_with($lower, $prefix)) {
                return true;
            }
        }

        $extension = pathinfo($lower, PATHINFO_EXTENSION);

        return $extension !== '' && in_array($extension, self::ASSET_EXTENSIONS, true);
    }

    private function referer(?string $referer): ?string
    {
        $parts = parse_url(trim((string) $referer));
        if (! is_array($parts) || ! in_array(strtolower((string) ($parts['scheme'] ?? '')), ['http', 'https'], true) || ! isset($parts['host'])) {
            return null;
        }

        $url = strtolower($parts['scheme'] ?? 'https').'://'.strtolower($parts['host']).(isset($parts['port']) ? ':'.$parts['port'] : '').($parts['path'] ?? '/');

        return mb_substr(rawurldecode($url), 0, 500);
    }
}
