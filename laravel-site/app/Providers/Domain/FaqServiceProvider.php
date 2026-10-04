<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\Models\FaqGroup;
use App\Domain\Faq\Models\FaqItem;
use App\Domain\Faq\Observers\FaqObserver;
use App\Domain\Faq\PageFaq;
use App\Domain\Faq\Repositories\CachedFaqRepository;
use App\Domain\Faq\Repositories\EloquentFaqRepository;
use App\Providers\DomainServiceProvider;
use Illuminate\Contracts\View\View;
use Illuminate\Support\Facades\View as ViewFacade;

final class FaqServiceProvider extends DomainServiceProvider
{
    /**
     * Page views whose controller belongs to another task get their contextual FAQ group from a composer: the view
     * receives `$faq` (FaqGroupData, or null when the group is missing/empty) and the group's questions are added to
     * the page's FAQPage JSON-LD (PageFaq). A view listed here MUST render it, e.g.
     * `<x-faq :faq="$faq" eyebrow="…" title="…"/>` (`variant="grid"` on plus) — JSON-LD is only for visible Q&As.
     * Pages with their own controller (/faq, stage pages) call PageFaq / FaqRepository directly instead.
     *
     * @var array<string, string> view name => FAQ group slug
     */
    public const PAGE_GROUPS = [
        'pages.home' => 'home',
        'pages.plus' => 'plus',
        'pages.contact' => 'contact',
        'pages.directory.business' => 'directory-business',
    ];

    protected array $repositories = [
        FaqRepository::class => [EloquentFaqRepository::class, CachedFaqRepository::class],
    ];

    protected array $observers = [
        FaqGroup::class => FaqObserver::class,
        FaqItem::class => FaqObserver::class,
    ];

    public function boot(): void
    {
        parent::boot();

        foreach (self::PAGE_GROUPS as $view => $slug) {
            ViewFacade::composer($view, function (View $view) use ($slug): void {
                $view->with('faq', $this->app->make(PageFaq::class)->group($slug));
            });
        }
    }
}
