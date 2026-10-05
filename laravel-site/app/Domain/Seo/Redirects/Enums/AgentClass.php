<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Enums;

/**
 * The only thing the 404 monitor keeps from a user agent: was it a crawler / script or a browser.
 */
enum AgentClass: string
{
    case Bot = 'bot';
    case Human = 'human';

    private const BOT_PATTERN = '~bot|crawl|spider|slurp|scrap|curl|wget|python|httpclient|http-client|okhttp|go-http|java/|libwww|perl|ruby|php/|node-fetch|axios|headless|phantom|lighthouse|pingdom|uptime|monitor|preview|facebookexternalhit|whatsapp|telegram|embedly|archiver|feed~i';

    public static function fromUserAgent(?string $userAgent): self
    {
        $userAgent = trim((string) $userAgent);

        return $userAgent === '' || preg_match(self::BOT_PATTERN, $userAgent) === 1 ? self::Bot : self::Human;
    }

    public function label(): string
    {
        return $this === self::Bot ? 'ربات' : 'کاربر';
    }
}
