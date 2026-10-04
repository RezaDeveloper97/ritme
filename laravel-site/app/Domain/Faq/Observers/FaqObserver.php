<?php

declare(strict_types=1);

namespace App\Domain\Faq\Observers;

use App\Domain\Faq\Models\FaqGroup;
use App\Domain\Faq\Models\FaqItem;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Html\RichHtmlSanitizer;
use App\Support\Html\TextSlug;
use Illuminate\Database\Eloquent\Model;

/**
 * Groups and items: any change bumps `faq` (+ `pages` via cacheaside.always_bump) — /faq, the home page, stage pages
 * and every other page with an FAQ block re-render on the next request.
 *
 * Before saving: a group slug is normalised (filled from the title when empty); an item answer is sanitised with the
 * rich-text allow-list (no images: FAQ answers are text, links and lists) and a plain-text answer is wrapped in <p>.
 */
final class FaqObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['faq'];
    }

    public function saving(Model $model): void
    {
        if ($model instanceof FaqGroup) {
            $slug = TextSlug::make(trim($model->slug ?? '') !== '' ? (string) $model->slug : $model->title, 64);
            $model->slug = $slug === '' ? 'faq' : $slug;
        }

        if ($model instanceof FaqItem && $model->isDirty('answer')) {
            $model->answer = self::cleanAnswer((string) $model->answer);
        }
    }

    public static function cleanAnswer(string $answer): string
    {
        $answer = trim($answer);
        if ($answer === '') {
            return '';
        }
        if (! str_contains($answer, '<')) {
            $answer = '<p>'.e($answer, false).'</p>';
        }

        return (new RichHtmlSanitizer([], []))->sanitize($answer);
    }
}
