<?php

declare(strict_types=1);

namespace App\Http\Controllers\Directory;

use App\Domain\Contact\Support\FormTimer;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\DistrictData;
use App\Domain\Directory\Enums\Weekday;
use App\Domain\Directory\Join\Actions\SubmitJoinRequest;
use App\Domain\Directory\Join\Enums\AgeGroup;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Join\Support\JoinForm;
use App\Domain\Media\Exceptions\InvalidMediaException;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Http\Requests\JoinRequest;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Response;
use Illuminate\Support\Carbon;

/**
 * The business side of the directory (L5-05):
 *
 *  - `/directory/business` (design/html/directory-business.html) — why / how / requirements / terms + the
 *    `directory-business` FAQ group (FaqServiceProvider composer → `$faq` + FAQPage JSON-LD). Indexable, page-cached.
 *  - `/directory/join` (directory-join.html) — ONE long form in four steps (معرفی · مکان و تصاویر · خدمات و قیمت ·
 *    مدارک). Without JS every step is visible; the lazy `stepper` module shows one step at a time. noindex and
 *    `no-store`: every render mints a fresh FormTimer token and carries the visitor's CSRF token / old input.
 *  - POST `/directory/join` — JoinRequest (validation + honeypot/time trap) → SubmitJoinRequest (photos through the
 *    media pipeline, stored pending) → PRG to the done page. Spam gets the same redirect with a made-up code and
 *    stores nothing; `throttle:directory-join` limits posts per IP (DirectoryServiceProvider).
 *  - `/directory/join/done` (directory-join-done.html) — noindex, `no-store`; shows the tracking code flashed by the
 *    post (a direct visit shows the same page without a code).
 */
final class JoinController
{
    public const FLASH = 'directory_join';

    public function __construct(
        private readonly SeoMetaRepository $meta,
        private readonly TaxonomyRepository $taxonomy,
        private readonly SiteNavigation $navigation,
        private readonly FormTimer $timer,
        private readonly ViewFactory $views,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function business(SeoManager $seo, SchemaGraph $graph): View
    {
        $this->seo($seo, $graph, StaticPage::DirectoryBusiness, 'business_page');

        return $this->views->make('pages.directory.business');
    }

    public function create(SeoManager $seo, SchemaGraph $graph): Response
    {
        $this->seo($seo, $graph, StaticPage::DirectoryJoin, 'join');
        $seo->noindex();

        return $this->noStore($this->views->make('pages.directory.join', [
            'categories' => array_map(static fn (CategoryData $c): array => ['id' => $c->id, 'name' => $c->name], $this->taxonomy->categories()),
            'cities' => array_map(static fn (CityData $c): array => [
                'id' => $c->id,
                'name' => $c->name,
                'districts' => array_map(static fn (DistrictData $d): array => ['id' => $d->id, 'name' => $d->name], $c->districts),
            ], $this->taxonomy->cities()),
            'amenities' => array_map(static fn (AmenityData $a): array => ['id' => $a->id, 'name' => $a->name], $this->taxonomy->amenities()),
            'ageGroups' => AgeGroup::options(),
            'bookingModes' => array_map(static fn (BookingMode $m): array => [
                'value' => $m->value, 'title' => $m->label(), 'text' => $m->description(),
            ], BookingMode::cases()),
            'weekdays' => array_map(static fn (Weekday $d): array => ['key' => $d->value, 'label' => $d->label()], Weekday::cases()),
            'defaultHours' => JoinForm::defaultHours(),
            'photoLimits' => [
                'max' => JoinForm::PHOTOS_MAX,
                'recommended' => JoinForm::PHOTOS_RECOMMENDED,
                'bytes' => JoinForm::photoMaxBytes(),
                'total' => JoinForm::photosTotalBytes(),
                'mimes' => JoinForm::PHOTO_MIMES,
            ],
            'formToken' => $this->timer->issue(),
        ])->render());
    }

    public function store(JoinRequest $request, SubmitJoinRequest $submit): RedirectResponse
    {
        $name = $request->string('name')->squish()->limit(JoinForm::NAME_MAX, '')->toString();

        if ($request->isSpam()) {
            return $this->redirectToDone((string) random_int(100000, 999999), $name);
        }

        try {
            $joinRequest = $submit->handle($request->toData());
        } catch (InvalidMediaException $e) {
            return redirect()->to(route('directory.join').'#join-form')
                ->withInput($request->except(['photos', JoinRequest::TIMER, '_token']))
                ->withErrors(['photos' => __('directory.join.validation.photo_rejected', ['reason' => $e->getMessage()])]);
        }

        return $this->redirectToDone($joinRequest->code, $joinRequest->name);
    }

    public function done(SeoManager $seo, SchemaGraph $graph): Response
    {
        $this->seo($seo, $graph, StaticPage::DirectoryJoinDone, 'done');
        $seo->noindex();

        $flash = session(self::FLASH);
        $flash = is_array($flash) ? $flash : [];
        $at = isset($flash['at']) && is_int($flash['at']) ? Carbon::createFromTimestamp($flash['at'], 'Asia/Tehran') : null;

        return $this->noStore($this->views->make('pages.directory.join-done', [
            'code' => isset($flash['code']) && is_string($flash['code']) ? $flash['code'] : null,
            'placeName' => isset($flash['name']) && is_string($flash['name']) && $flash['name'] !== '' ? $flash['name'] : null,
            'submittedAt' => $at,
        ])->render());
    }

    /** PRG target of every post (real or spam): the code, place name and time travel in the flash. */
    private function redirectToDone(string $code, string $name): RedirectResponse
    {
        return redirect()->route('directory.join.done')->with(self::FLASH, ['code' => $code, 'name' => $name, 'at' => now()->getTimestamp()]);
    }

    private function seo(SeoManager $seo, SchemaGraph $graph, StaticPage $page, string $key): void
    {
        $meta = $this->meta->forRoute($page->routeName());
        if (($meta->title ?? '') === '') {
            $seo->rawTitle(self::text("directory.{$key}.seo.title"));
        }
        if (($meta->description ?? '') === '') {
            $seo->description(self::text("directory.{$key}.seo.description"));
        }
        $graph->pageName($page->label())->breadcrumbs(...$this->navigation->breadcrumbs($page));
    }

    private function noStore(string $content): Response
    {
        return new Response($content, 200, ['Content-Type' => 'text/html; charset=UTF-8', 'Cache-Control' => 'no-store, private']);
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
