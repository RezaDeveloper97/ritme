<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

/**
 * Persian names of the audit finding codes (admin report, widgets, CLI legend). Unknown codes show as-is.
 */
final class IssueCodes
{
    public const LABELS = [
        'render' => 'خطای بارگذاری صفحه',
        'title.missing' => 'بدون عنوان',
        'title.multiple' => 'چند عنوان',
        'title.length' => 'طول عنوان',
        'title.duplicate' => 'عنوان تکراری',
        'description.missing' => 'بدون توضیح متا',
        'description.length' => 'طول توضیح متا',
        'description.duplicate' => 'توضیح متای تکراری',
        'h1.count' => 'تعداد h1',
        'canonical.missing' => 'بدون canonical',
        'canonical.multiple' => 'چند canonical',
        'canonical.relative' => 'canonical ناقص',
        'canonical.host' => 'canonical روی دامنه دیگر',
        'canonical.self' => 'canonical به صفحه دیگر',
        'canonical.target' => 'canonical به صفحه خراب',
        'og.missing' => 'Open Graph ناقص',
        'og.image' => 'بدون تصویر اشتراک',
        'og.image-missing' => 'تصویر اشتراک پیدا نشد',
        'og.image-external' => 'تصویر اشتراک بیرونی',
        'img.attributes' => 'تصویر بدون alt/ابعاد',
        'img.lcp-lazy' => 'تصویر اصلی lazy است',
        'img.not-lazy' => 'تصویرهای پایین صفحه lazy نیستند',
        'link.hash' => 'پیوند خالی (#)',
        'link.broken' => 'پیوند داخلی خراب',
        'link.fragment' => 'لنگر (#) پیدا نشد',
        'link.redirect' => 'پیوند به ریدایرکت',
        'redirect.chain' => 'زنجیره ریدایرکت',
        'redirect.loop' => 'حلقه ریدایرکت',
        'jsonld.missing' => 'بدون داده ساختاریافته',
        'jsonld.parse' => 'JSON-LD نامعتبر',
        'jsonld.required' => 'JSON-LD ناقص',
        'sitemap.noindex' => 'noindex در نقشه سایت',
        'sitemap.status' => 'نشانی خراب در نقشه سایت',
        'sitemap.redirect' => 'ریدایرکت در نقشه سایت',
        'sitemap.canonical' => 'نشانی غیرکانونی در نقشه سایت',
        'sitemap.missing' => 'خارج از نقشه سایت',
        'page.orphan' => 'صفحه یتیم (بی‌پیوند)',
        'content.thin' => 'محتوای کم',
        'content.score' => 'امتیاز تحلیل محتوا پایین',
        'weight.html' => 'حجم HTML زیاد',
        'weight.requests' => 'درخواست‌های زیاد',
        'external.request' => 'درخواست بیرونی',
        'perf.slow' => 'پاسخ کند',
    ];

    public static function label(string $code): string
    {
        return self::LABELS[$code] ?? $code;
    }

    /**
     * @return array<string, string> code => label, for filters
     */
    public static function options(): array
    {
        return self::LABELS;
    }
}
