<?php

declare(strict_types=1);

namespace App\Http\Controllers\Directory;

use App\Domain\Directory\Booking\Actions\CreateBookingRequest;
use App\Domain\Directory\Booking\Data\BookingData;
use App\Domain\Directory\Booking\Data\BookingSubmission;
use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Booking\Queries\FindBookingByCode;
use App\Domain\Directory\Booking\Support\BookingCode;
use App\Domain\Directory\Booking\Support\ChildAge;
use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Http\Requests\BookingRequest as BookingForm;
use Carbon\CarbonImmutable;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Response;

/**
 * Booking requests of the directory (L5-04):
 *
 *  - POST `/directory/place/{slug}/book` — the form in the place page's `#book` panel. BookingRequest (validation,
 *    honeypot + FormTimer, Persian messages, `booking` error bag) → CreateBookingRequest (one transaction, queued
 *    notifications after commit) → PRG to the booked page. A spam post gets the same redirect to a made-up code whose
 *    page (rebuilt once from the flash) looks like a real one; nothing is stored. `throttle:directory-booking` limits
 *    posts per IP and per mobile.
 *  - GET `/directory/booked/{code}` (design/html/directory-booked.html) — the request's status page, found only by its
 *    unguessable code (BookingCode). noindex + `no-store`; the mobile is masked and the note is not shown. A REQUEST,
 *    not a confirmed slot: the copy says the place confirms the time by phone, and nothing is paid here.
 */
final class BookingController
{
    public const SPAM_FLASH = 'directory_booking_echo';

    public function __construct(
        private readonly PlaceRepository $places,
        private readonly FindBookingByCode $find,
        private readonly SettingsRepository $settings,
        private readonly DirectoryUrls $urls,
        private readonly Config $config,
        private readonly ViewFactory $views,
    ) {}

    public function store(BookingForm $request, CreateBookingRequest $create): RedirectResponse
    {
        $place = $request->place();

        if ($request->isSpam()) {
            $code = BookingCode::generate();
            $submission = $request->toSpamSubmission();

            return redirect()->route('directory.booked', [$code])->with(self::SPAM_FLASH, [
                'code' => $code,
                'slug' => $place->slug,
                'service' => $submission->serviceId,
                'date' => $submission->date->format('Y-m-d'),
                'window' => $submission->window->value,
                'name' => $submission->parentName,
                'mobile' => $submission->mobile,
                'age' => $submission->childAgeMonths,
            ]);
        }

        $booking = $create->handle($place, $request->toSubmission());

        return redirect()->route('directory.booked', [$booking->code]);
    }

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the controller. */
    public function show(string $code, SeoManager $seo, SchemaGraph $graph): Response
    {
        $booking = $this->find->handle($code) ?? $this->echoed($code) ?? abort(404);
        $place = $booking->placeSlug === null ? null : $this->places->findPublishedBySlug($booking->placeSlug);

        $title = self::text('directory.booked.title.'.$booking->status->value);
        $seo->rawTitle(self::text('directory.booked.seo.title'))->description(self::text('directory.booked.seo.description'))->noindex();
        $graph->pageName($title)->breadcrumbs(...$this->trail($place));

        return new Response($this->views->make('pages.directory.booked', [
            'booking' => $booking,
            'title' => $title,
            'lead' => $this->lead($booking),
            'rows' => $this->rows($booking),
            'place' => $place === null ? null : [
                'url' => $this->urls->place($place->slug),
                'phone' => $place->phones[0] ?? null,
                'directions' => $place->mapLinks()?->geo,
                'cover' => $place->coverMediaId,
                'illustration' => ShowPlaceController::coverIllustration($place->category->slug),
                'area' => $place->district->name ?? $place->city->name,
                'rules' => $place->ruleLines(),
                'cancellation' => trim((string) $place->cancellationPolicy) === '' ? null : trim((string) $place->cancellationPolicy),
            ],
            'appLinks' => $this->settings->all()->appLinks,
            'navRoute' => 'directory.index',
            'appCta' => true,
        ])->render(), 200, ['Content-Type' => 'text/html; charset=UTF-8', 'Cache-Control' => 'no-store, private']);
    }

    /** The page of a spam post: only right after its redirect (flash), never stored, never again. */
    private function echoed(string $code): ?BookingData
    {
        $echo = session(self::SPAM_FLASH);
        if (! is_array($echo) || ($echo['code'] ?? null) !== BookingCode::normalize($code) || ! is_string($echo['slug'] ?? null)) {
            return null;
        }
        $place = $this->places->findPublishedBySlug($echo['slug']);
        if ($place === null) {
            return null;
        }

        return BookingData::fromSubmission($echo['code'], $place, new BookingSubmission(
            serviceId: is_int($echo['service'] ?? null) ? $echo['service'] : null,
            date: CarbonImmutable::parse(is_string($echo['date'] ?? null) ? $echo['date'] : 'tomorrow', 'Asia/Tehran'),
            window: TimeWindow::tryFrom(is_string($echo['window'] ?? null) ? $echo['window'] : '') ?? TimeWindow::Any,
            parentName: is_string($echo['name'] ?? null) ? $echo['name'] : '',
            mobile: is_string($echo['mobile'] ?? null) ? $echo['mobile'] : '',
            childAgeMonths: is_int($echo['age'] ?? null) ? $echo['age'] : null,
        ));
    }

    private function lead(BookingData $booking): string
    {
        $key = $booking->status === BookingStatus::New ? 'lead.new' : 'lead.'.$booking->status->value;

        return (string) __('directory.booked.'.$key, ['place' => $booking->placeName]);
    }

    /**
     * Summary rows under the service: preferred day + window, the parent (child age + masked number), the announced price.
     *
     * @return list<array{icon: string, title: string, text: string, price?: int, unit?: string|null}>
     */
    private function rows(BookingData $booking): array
    {
        $t = 'directory.booked.rows.';
        $rows = [[
            'icon' => 'calendar',
            'title' => jdate($booking->preferredDate, 'l j F Y'),
            'text' => $booking->timeWindow->describe().' · '.self::text($t.($booking->status === BookingStatus::Confirmed ? 'confirmed_time' : 'preferred')),
        ]];
        $rows[] = [
            'icon' => 'person',
            'title' => $booking->parentName !== '' ? $booking->parentName : self::text($t.'parent'),
            'text' => implode(' · ', array_filter([
                $booking->childAgeMonths === null ? null : (string) __($t.'child', ['age' => ChildAge::label($booking->childAgeMonths)]),
                (string) __($t.'mobile', ['mobile' => $booking->maskedMobile]),
            ])),
        ];
        if ($booking->servicePrice !== null) {
            $rows[] = [
                'icon' => 'card',
                'price' => $booking->servicePrice,
                'unit' => $booking->servicePriceUnit,
                'title' => '',
                'text' => self::text($t.'pay'),
            ];
        }

        return $rows;
    }

    /**
     * Home → directory → place (while published) → this page.
     *
     * @return list<BreadcrumbItem>
     */
    private function trail(?PlaceData $place): array
    {
        return array_values(array_filter([
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem((string) __('directory.name'), $this->urls->absolute(route('directory.index'))),
            $place === null ? null : new BreadcrumbItem($place->name, $this->urls->place($place->slug)),
            new BreadcrumbItem(self::text('directory.booked.breadcrumb')),
        ]));
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
