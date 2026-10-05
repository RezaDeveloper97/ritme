<?php

declare(strict_types=1);

namespace App\Http\Controllers\Shop;

use App\Domain\Contact\Support\FormTimer;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Shop\Cart\Actions\ResolveCart;
use App\Domain\Shop\Cart\Data\CartItemData;
use App\Domain\Shop\Cart\Data\CartNotice;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Domain\Shop\Ordering\Actions\PlaceOrder;
use App\Domain\Shop\Ordering\Data\OrderData;
use App\Domain\Shop\Ordering\Data\OrderItemData;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Exceptions\CheckoutException;
use App\Domain\Shop\Ordering\Queries\FindOrder;
use App\Domain\Shop\Ordering\Support\CartSignature;
use App\Domain\Shop\Ordering\Support\CheckoutSession;
use App\Domain\Shop\Ordering\Support\DeliverySlots;
use App\Domain\Shop\Ordering\Support\IranProvinces;
use App\Domain\Shop\Payment\Contracts\PaymentGateway;
use App\Http\Requests\CheckoutRequest;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Response;
use Illuminate\Routing\Controllers\HasMiddleware;
use Illuminate\Routing\Controllers\Middleware;

/**
 * Checkout + order page of the shop (L6-05), cash on delivery only (PaymentGateway contract, CashOnDeliveryGateway):
 *
 *  - GET `/shop/checkout` (design/html/shop-checkout.html) — recipient + address form (province list and city
 *    suggestions are local data), preferred delivery slot, «بسته‌بندی ساده», payment = COD only (AUDIT §8: no bank
 *    gateway option is rendered), live summary. Carries a one-time token (idempotency), the CartSignature of what is
 *    shown, a FormTimer token + honeypot. noindex, `no-store`, never page-cached.
 *  - POST `/shop/checkout` — CheckoutRequest → PlaceOrder (one transaction, live re-price + AdjustStock) → PRG 303 to
 *    the order page. Refusals (cart changed, sold out, COD cap …) come back to the form with Persian messages and the
 *    typed input kept. A repeated submit of the same form lands on the first order. `throttle:shop-checkout`.
 *  - GET `/shop/order/{code}` (design/html/shop-done.html) — found only by the unguessable code; noindex + `no-store`.
 *    Shows status, items and the amount due ON DELIVERY (never "paid"); the recipient rows (name, masked mobile,
 *    province/city — never the street address) only for the session that placed the order.
 */
final class CheckoutController implements HasMiddleware
{
    /** L9-04: order pages are found by an unguessable code only; this caps code guessing per IP (per route, ThrottlePerRoute). */
    public const LOOKUPS_PER_MINUTE = 20;

    /**
     * @return list<Middleware>
     */
    public static function middleware(): array
    {
        return [new Middleware('throttle:'.self::LOOKUPS_PER_MINUTE.',1', only: ['order'])];
    }

    public const ERROR_FLASH = 'checkout_error';

    public const NOTICES_FLASH = 'checkout_notices';

    public function __construct(
        private readonly FindOrder $find,
        private readonly MediaRepository $media,
        private readonly SettingsRepository $settings,
        private readonly ShopUrls $urls,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    /**
     * SeoManager, SchemaGraph, CheckoutSession and the cart (ResolveCart → CartRepository) are request-scoped and the
     * gateway reads the shop settings: injected per call, not into the (route-cached) controller.
     */
    public function show(Request $request, FormTimer $timer, SeoManager $seo, SchemaGraph $graph, ResolveCart $resolve, CheckoutSession $session, PaymentGateway $gateway): Response
    {
        $cart = $resolve->handle();
        $soldOut = array_values(array_filter($cart->items, static fn (CartItemData $i): bool => ! $i->available));
        $limit = $gateway->limit();

        $seo->rawTitle(self::text('shop.checkout.seo_title'))->description(self::text('shop.checkout.seo_description'))->noindex();
        $graph->pageName(self::text('shop.checkout.title'))->breadcrumbs(...$this->trail(self::text('shop.checkout.title')));

        $slots = DeliverySlots::offered();
        $oldSlot = $request->old('delivery');

        return $this->page('pages.shop.checkout', [
            'cart' => $cart,
            'state' => match (true) {
                ! $cart->canCheckout() => 'empty',
                $soldOut !== [] => 'sold_out',
                ! $gateway->accepts($cart->total) => 'cod_limit',
                default => 'ready',
            },
            'soldOut' => array_map(static fn (CartItemData $i): string => $i->title, $soldOut),
            'error' => $this->flash($request, self::ERROR_FLASH),
            'notices' => array_values(array_filter((array) $request->session()->get(self::NOTICES_FLASH, []), 'is_string')),
            'slots' => $slots,
            'selectedSlot' => is_string($oldSlot) && $oldSlot !== '' ? $oldSlot : $slots[0]->value(),
            'provinces' => IranProvinces::options(),
            'cities' => IranProvinces::cities(),
            'paymentMethod' => $gateway->method(),
            'codLimit' => $limit?->format(),
            'token' => $session->token(),
            'signature' => CartSignature::of($cart),
            'formToken' => $timer->issue(),
            'navRoute' => 'shop.index',
        ]);
    }

    public function store(CheckoutRequest $request, PlaceOrder $place, CheckoutSession $session): RedirectResponse
    {
        if ($request->isSpam()) {
            return redirect()->route('shop.checkout')->with(self::ERROR_FLASH, self::text('shop.checkout.retry'));
        }

        $submission = $request->toSubmission();
        if (! $session->isCurrent($submission->token)) {
            // The form was already used: a double submit lands on the order it placed; anything else is a stale form.
            $existing = $this->find->byToken($submission->token);
            if ($existing !== null && $session->owns($existing->code)) {
                return redirect()->route('shop.order', [$existing->code], 303);
            }

            return redirect()->route('shop.checkout')->withInput($request->except(['checkout_token', CheckoutRequest::TIMER]))
                ->with(self::ERROR_FLASH, self::text('shop.checkout.validation.checkout_token'));
        }

        try {
            $placement = $place->handle($submission);
        } catch (CheckoutException $e) {
            return redirect()->to(route('shop.checkout').'#checkout-form')
                ->withInput($request->except(['checkout_token', 'cart_signature', CheckoutRequest::TIMER]))
                ->with(self::ERROR_FLASH, self::problem($e))
                ->with(self::NOTICES_FLASH, array_map(self::notice(...), $e->notices));
        }

        $session->remember($placement->order->code);

        return $placement->payment->redirectUrl !== null
            ? redirect()->away($placement->payment->redirectUrl, 303)
            : redirect()->route('shop.order', [$placement->order->code], 303);
    }

    public function order(string $code, SeoManager $seo, SchemaGraph $graph, CheckoutSession $session): Response
    {
        $order = $this->find->byCode($code) ?? abort(404);
        $owner = $session->owns($order->code);
        $cancelled = $order->status === OrderStatus::Cancelled;
        $title = self::text($cancelled ? 'shop.order.title_cancelled' : 'shop.order.title');

        $seo->rawTitle(self::text('shop.order.seo_title'))->description(self::text('shop.order.seo_description'))->noindex();
        $graph->pageName($title)->breadcrumbs(...$this->trail(self::text('shop.order.breadcrumb')));

        $ids = array_values(array_unique(array_filter(array_map(static fn (OrderItemData $i): ?int => $i->coverMediaId, $order->items))));

        return $this->page('pages.shop.done', [
            'order' => $order,
            'owner' => $owner,
            'title' => $title,
            'timeline' => $this->timeline($order),
            'media' => $ids === [] ? [] : $this->media->findMany($ids),
            'appLinks' => $this->settings->all()->appLinks,
            'navRoute' => 'shop.index',
            'appCta' => true,
        ]);
    }

    /**
     * Order status as the design's per-seller timeline: placed → confirmed (by phone) → shipped → delivered (with the
     * preferred slot). Cancelled orders show «لغو شد» after «ثبت سفارش».
     *
     * @return list<array{title: string, time?: string|\DateTimeInterface, state: string}>
     */
    private function timeline(OrderData $order): array
    {
        $t = 'shop.order.timeline.';
        if ($order->status === OrderStatus::Cancelled) {
            return [
                ['title' => self::text($t.'placed'), 'time' => $order->placedAt, 'state' => 'done'],
                ['title' => self::text($t.'cancelled'), 'state' => 'done'],
            ];
        }

        $step = $order->status->step(); // pending 1 … delivered 4
        $state = static fn (int $i): string => $i <= $step ? 'done' : ($i === $step + 1 ? 'current' : 'todo');

        return [
            ['title' => self::text($t.'placed'), 'time' => $order->placedAt, 'state' => 'done'],
            ['title' => self::text($step === 1 ? $t.'confirm_pending' : $t.'confirmed'), 'time' => $step === 1 ? self::text($t.'confirm_hint') : '', 'state' => $state(2)],
            ['title' => self::text($t.'shipped'), 'state' => $state(3)],
            ['title' => self::text($t.'delivered'), 'time' => $step === 4 ? '' : jdate($order->deliveryDate, 'l j F').' · '.$order->deliveryWindow->hours(), 'state' => $state(4)],
        ];
    }

    /**
     * @param  view-string  $view
     * @param  array<string, mixed>  $data
     */
    private function page(string $view, array $data): Response
    {
        return new Response($this->views->make($view, $data)->render(), 200, [
            'Content-Type' => 'text/html; charset=UTF-8',
            'Cache-Control' => 'no-store, private',
        ]);
    }

    /**
     * Home → shop → cart → this page.
     *
     * @return list<BreadcrumbItem>
     */
    private function trail(string $label): array
    {
        return [
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem(self::text('shop.name'), $this->urls->absolute(route('shop.index'))),
            new BreadcrumbItem(self::text('shop.cart.title'), $this->urls->absolute(route('shop.cart'))),
            new BreadcrumbItem($label),
        ];
    }

    private function flash(Request $request, string $key): ?string
    {
        $value = $request->hasSession() ? $request->session()->get($key) : null;

        return is_string($value) && $value !== '' ? $value : null;
    }

    private static function problem(CheckoutException $e): string
    {
        $text = __('shop.checkout.problems.'.$e->problem->value, $e->params);

        return is_string($text) ? $text : '';
    }

    private static function notice(CartNotice $notice): string
    {
        $text = __($notice->problem->translationKey(), array_map(static fn (string|int $v): string => is_int($v) ? fa_digits($v) : $v, $notice->params));

        return is_string($text) ? $text : '';
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
