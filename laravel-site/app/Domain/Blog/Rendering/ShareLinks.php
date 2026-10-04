<?php

declare(strict_types=1);

namespace App\Domain\Blog\Rendering;

/**
 * Plain share URLs (no third-party script, no request until the reader clicks): Telegram, WhatsApp, X.
 */
final class ShareLinks
{
    /**
     * @return list<array{key: string, label: string, href: string}>
     */
    public static function for(string $url, string $title): array
    {
        $u = rawurlencode($url);
        $t = rawurlencode($title);

        return [
            ['key' => 'telegram', 'label' => 'تلگرام', 'href' => "https://t.me/share/url?url={$u}&text={$t}"],
            ['key' => 'whatsapp', 'label' => 'واتس‌اپ', 'href' => 'https://wa.me/?text='.rawurlencode($title.' '.$url)],
            ['key' => 'x', 'label' => 'ایکس', 'href' => "https://x.com/intent/post?url={$u}&text={$t}"],
        ];
    }
}
