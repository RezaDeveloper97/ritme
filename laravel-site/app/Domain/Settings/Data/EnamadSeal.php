<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

/**
 * The enamad trust seal, read from whatever the admin pasted (the official HTML snippet, the seal link or a plain
 * `id`/`Code` pair) into the legal settings. Rendered as enamad's own image linking to the verification page.
 *
 * The image is the one deliberate exception to "no external requests on public pages": enamad only accepts its
 * seal when the image is loaded from trustseal.enamad.ir with the site as referrer. HOST is allowed in the CSP
 * img-src (SecurityHeaders) and nothing else from that host is loaded.
 */
final readonly class EnamadSeal
{
    public const HOST = 'https://trustseal.enamad.ir';

    /** Native size of logo.aspx. */
    public const WIDTH = 125;

    public const HEIGHT = 136;

    private function __construct(
        public string $id,
        public string $code,
    ) {}

    public static function fromSettings(LegalSettings $legal): ?self
    {
        foreach ([$legal->enamadHtml, $legal->enamadCode] as $source) {
            $seal = self::parse((string) $source);
            if ($seal !== null) {
                return $seal;
            }
        }

        return null;
    }

    public static function parse(string $text): ?self
    {
        $text = html_entity_decode($text, ENT_QUOTES | ENT_HTML5, 'UTF-8');
        $id = preg_match('/[?&]id=(\d{1,12})\b/i', $text, $m) === 1 ? $m[1] : null;
        $code = preg_match('/[?&]Code=([A-Za-z0-9]{8,64})\b/', $text, $m) === 1 ? $m[1] : null;

        return $id !== null && $code !== null ? new self($id, $code) : null;
    }

    public function linkUrl(): string
    {
        return self::HOST.'/?'.http_build_query(['id' => $this->id, 'Code' => $this->code]);
    }

    public function imageUrl(): string
    {
        return self::HOST.'/logo.aspx?'.http_build_query(['id' => $this->id, 'Code' => $this->code]);
    }
}
