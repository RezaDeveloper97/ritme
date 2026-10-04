<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts;

use App\Domain\Blog\Models\Post;
use App\Domain\Seo\Models\SeoMeta;
use App\Support\Html\HtmlText;
use BackedEnum;
use DateTimeInterface;
use Filament\Facades\Filament;
use Illuminate\Support\HtmlString;
use Spatie\Activitylog\Models\Activity;

/**
 * Revision trail of a post on top of the activity log (log `blog`): a snapshot of the editable fields (content, tags,
 * SEO) is taken before a save and the changed fields are logged as old → new afterwards. The edit page shows the
 * trail in a modal; the full entries (incl. old body HTML) are in «گزارش فعالیت‌ها».
 */
final class PostRevisions
{
    public const LOG = 'blog';

    public const LABELS = [
        'title' => 'عنوان',
        'slug' => 'نامک',
        'excerpt' => 'خلاصه',
        'body' => 'متن',
        'sources' => 'منابع',
        'status' => 'وضعیت',
        'published_at' => 'زمان انتشار',
        'category_id' => 'دسته',
        'author_id' => 'نویسنده',
        'reviewer_id' => 'بازبین پزشکی',
        'reviewed_at' => 'تاریخ بازبینی',
        'life_stage' => 'مرحله زندگی',
        'is_featured' => 'ویژه',
        'cover_media_id' => 'تصویر شاخص',
        'cover_mobile_media_id' => 'تصویر شاخص موبایل',
        'tags' => 'برچسب‌ها',
        'seo.title' => 'عنوان سئو',
        'seo.description' => 'توضیح متا',
        'seo.focus_keyword' => 'کلیدواژه کانونی',
        'seo.canonical_url' => 'کنونیکال',
        'seo.robots' => 'robots',
        'seo.og_title' => 'عنوان اشتراک‌گذاری',
        'seo.og_description' => 'توضیح اشتراک‌گذاری',
        'seo.og_media_id' => 'تصویر اشتراک‌گذاری',
        'seo.sitemap_include' => 'در نقشه سایت',
        'seo.sitemap_priority' => 'اولویت نقشه سایت',
    ];

    private const LONG = ['body', 'sources'];

    /**
     * @return array<string, string|null>
     */
    public static function snapshot(Post $post): array
    {
        $post->load(['tags', 'seoMeta']);
        $seo = $post->seoMeta ?? new SeoMeta; // no row yet = the defaults
        $values = [];

        foreach (array_keys(self::LABELS) as $key) {
            $values[$key] = match (true) {
                $key === 'tags' => $post->tags->pluck('name')->sort()->implode('، '),
                str_starts_with($key, 'seo.') => self::scalar($seo->getAttribute(substr($key, 4))),
                default => self::scalar($post->getAttribute($key)),
            };
        }

        return $values;
    }

    /**
     * Logs the difference between two snapshots; nothing is logged when nothing changed.
     *
     * @param  array<string, string|null>  $before
     */
    public static function log(Post $post, array $before, string $event = 'updated'): void
    {
        $after = self::snapshot($post);
        $old = [];
        $new = [];
        foreach ($after as $key => $value) {
            if ($event === 'created' ? $value !== null && $value !== '' : ! self::same($key, $before[$key] ?? null, $value)) {
                $old[$key] = $before[$key] ?? null;
                $new[$key] = $value;
            }
        }

        if ($new === []) {
            return;
        }

        activity(self::LOG)
            ->causedBy(Filament::auth()->user())
            ->performedOn($post)
            ->event($event)
            ->withProperties($event === 'created' ? ['attributes' => $new] : ['old' => $old, 'attributes' => $new])
            ->log('post.'.$event);
    }

    /**
     * Plain event (duplicated, published, unpublished …) without a field diff.
     *
     * @param  array<string, mixed>  $properties
     */
    public static function event(Post $post, string $event, array $properties = []): void
    {
        activity(self::LOG)
            ->causedBy(Filament::auth()->user())
            ->performedOn($post)
            ->event($event)
            ->withProperties($properties)
            ->log('post.'.$event);
    }

    /**
     * The trail as HTML for the edit page modal (newest first, escaped).
     */
    public static function render(Post $post, int $limit = 30): HtmlString
    {
        $entries = Activity::query()
            ->where('subject_type', $post->getMorphClass())
            ->where('subject_id', $post->getKey())
            ->with('causer')
            ->latest('id')
            ->limit($limit)
            ->get();

        if ($entries->isEmpty()) {
            return new HtmlString('<p class="text-sm text-gray-500 dark:text-gray-400">هنوز تغییری ثبت نشده است.</p>');
        }

        $html = '<ol class="flex flex-col gap-4" data-post-revisions>';
        foreach ($entries as $entry) {
            $causer = $entry->causer?->getAttribute('name');
            $html .= '<li class="rounded-lg p-3 ring-1 ring-gray-950/10 dark:ring-white/10">'
                .'<p class="text-sm font-medium text-gray-950 dark:text-white">'.e(self::eventLabel((string) $entry->event))
                .' <span class="font-normal text-gray-500 dark:text-gray-400">· '.e($entry->created_at?->format('Y-m-d H:i') ?? '').' · '.e(is_string($causer) ? $causer : 'سیستم').'</span></p>';

            $attributes = $entry->properties->get('attributes', []);
            $old = $entry->properties->get('old', []);
            if (is_array($attributes) && $attributes !== []) {
                $html .= '<dl class="mt-2 grid gap-1 text-sm">';
                foreach ($attributes as $key => $value) {
                    $html .= '<div class="flex flex-wrap gap-x-2"><dt class="text-gray-500 dark:text-gray-400">'.e(self::LABELS[$key] ?? (string) $key).':</dt><dd class="text-gray-950 dark:text-white">'
                        .e(self::describe((string) $key, is_array($old) && array_key_exists($key, $old), is_array($old) ? ($old[$key] ?? null) : null, $value)).'</dd></div>';
                }
                $html .= '</dl>';
            }
            $html .= '</li>';
        }

        return new HtmlString($html.'</ol>');
    }

    private static function describe(string $key, bool $hasOld, mixed $old, mixed $new): string
    {
        if (in_array($key, self::LONG, true)) {
            $before = HtmlText::wordCount((string) $old);
            $after = HtmlText::wordCount((string) $new);

            return ! $hasOld ? "{$after} واژه" : "تغییر کرد ({$before} ← {$after} واژه)";
        }

        $show = static fn (mixed $value): string => $value === null || $value === '' ? '—' : mb_strimwidth((string) (is_scalar($value) ? $value : json_encode($value)), 0, 120, '…');

        return $hasOld ? $show($old).' ← '.$show($new) : $show($new);
    }

    /**
     * The editor re-serialises HTML (whitespace next to tags); such a save is not a content change.
     */
    private static function same(string $key, ?string $old, ?string $new): bool
    {
        if (! in_array($key, self::LONG, true) || $old === null || $new === null) {
            return $old === $new;
        }

        $normalize = static fn (string $html): string => trim((string) preg_replace(['/\s+/u', '/\s*(<[^>]+>)\s*/u'], [' ', '$1'], $html));

        return $normalize($old) === $normalize($new);
    }

    private static function eventLabel(string $event): string
    {
        return match ($event) {
            'created' => 'ایجاد',
            'updated' => 'ویرایش',
            'duplicated' => 'نسخه‌برداری',
            'published' => 'انتشار',
            'unpublished' => 'لغو انتشار',
            'deleted' => 'حذف',
            default => $event,
        };
    }

    private static function scalar(mixed $value): ?string
    {
        return match (true) {
            $value === null => null,
            $value instanceof BackedEnum => (string) $value->value,
            $value instanceof DateTimeInterface => $value->format('Y-m-d H:i'),
            is_bool($value) => $value ? '1' : '0',
            is_scalar($value) => (string) $value,
            default => (string) json_encode($value, JSON_UNESCAPED_UNICODE),
        };
    }
}
