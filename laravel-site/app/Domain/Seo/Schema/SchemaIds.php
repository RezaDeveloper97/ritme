<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema;

/**
 * Stable `@id`s. Site-wide nodes hang off the site root (`https://ritme.ir/#organization`), page nodes off the page's
 * canonical URL (`https://ritme.ir/cycle#webpage`). Never change these: search engines merge entities by `@id`.
 */
final class SchemaIds
{
    public static function organization(string $siteUrl): string
    {
        return self::root($siteUrl).'#organization';
    }

    public static function website(string $siteUrl): string
    {
        return self::root($siteUrl).'#website';
    }

    public static function logo(string $siteUrl): string
    {
        return self::root($siteUrl).'#logo';
    }

    public static function app(string $siteUrl): string
    {
        return self::root($siteUrl).'#app';
    }

    public static function person(string $siteUrl, string $key): string
    {
        $slug = trim((string) preg_replace('/[^\p{L}\p{N}]+/u', '-', mb_strtolower($key)), '-');

        return self::root($siteUrl).'#/person/'.rawurlencode($slug !== '' ? $slug : substr(sha1($key), 0, 12));
    }

    public static function webPage(string $url): string
    {
        return $url.'#webpage';
    }

    public static function breadcrumb(string $url): string
    {
        return $url.'#breadcrumb';
    }

    public static function primaryImage(string $url): string
    {
        return $url.'#primaryimage';
    }

    public static function article(string $url): string
    {
        return $url.'#article';
    }

    public static function faq(string $url): string
    {
        return $url.'#faq';
    }

    public static function product(string $url): string
    {
        return $url.'#product';
    }

    public static function place(string $url): string
    {
        return $url.'#place';
    }

    public static function itemList(string $url): string
    {
        return $url.'#itemlist';
    }

    /**
     * The site root with exactly one trailing slash.
     */
    public static function root(string $siteUrl): string
    {
        return rtrim($siteUrl, '/').'/';
    }
}
