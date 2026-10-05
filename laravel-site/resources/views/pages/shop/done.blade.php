{{--
    Order page (L6-05, design/html/shop-done.html): /shop/order/{code} — noindex, no-store, never page-cached.
    Data (Shop\CheckoutController::order): $order (OrderData — already masked), $owner (this session placed the order →
    recipient rows shown), $title, $timeline (x-ui.timeline items), $media (cover MediaData by id), $appLinks.
    Audit corrections: single seller → one «فروشگاه ریتمی» card (+ a details card in the second column); cash on
    delivery → never «پرداخت شد» / «رسید پرداخت»: the amount is due on delivery; no street address on the page.
--}}
@extends('layouts.app')

@php
    $t = 'shop.order.';
    $tints = ['bg-stage-pregnancy/15', 'bg-primary/13', 'bg-stage-pregnancy/13', 'bg-stage-postpartum/13'];
    $cancelled = $order->status === \App\Domain\Shop\Ordering\Enums\OrderStatus::Cancelled;
    $fee = match (true) {
        $order->shippingFee === null => __($t.'shipping_later'),
        $order->shippingFee->isZero() => __($t.'shipping_free'),
        default => $order->shippingFee->formatShort(),
    };
    $row = 'flex justify-between gap-3 border-t border-line py-3 text-[14.5px] font-bold max-lg:flex-wrap';
@endphp

@section('content')
    <section class="flex flex-col items-center gap-5 px-30 pt-14 pb-24 text-center max-lg:px-5 max-lg:pt-[30.8px] max-lg:pb-[52.8px]">
        <x-ui.success-hero :icon="$cancelled ? 'x' : 'check'" :title="$title">{{ __($cancelled ? $t.'lead_cancelled' : $t.'lead', ['code' => $order->code]) }}</x-ui.success-hero>

        <div class="flex w-250 max-w-full items-stretch gap-5 text-start max-lg:flex-wrap">
            <section aria-labelledby="order-seller" class="flex-1 rounded-5xl border border-line bg-surface px-7 pt-6 pb-2 max-lg:basis-75 max-sm:basis-full max-sm:px-5.5">
                <div class="flex items-center justify-between gap-3">
                    <h2 id="order-seller" class="m-0 flex items-center gap-2 text-lg font-extrabold"><x-icon name="store" class="size-[19px] text-primary"/>{{ __($t.'seller') }}</h2>
                    <span class="inline-flex h-7 items-center rounded-full border border-primary/40 px-2.5 text-[11.5px] font-bold whitespace-nowrap text-ink">{{ $order->status->label() }}</span>
                </div>
                <ul aria-label="{{ __($t.'items') }}" class="m-0 my-4 flex list-none flex-wrap gap-2.5 p-0">
                    @foreach ($order->items as $item)
                        @php($cover = $item->coverMediaId === null ? null : ($media[$item->coverMediaId] ?? null))
                        <li class="relative size-18 shrink-0 overflow-hidden rounded-xl {{ $tints[$loop->index % count($tints)] }}">
                            @if ($cover)
                                <x-picture :media="$cover" :alt="$item->title" sizes="72px" class="size-full object-cover"/>
                            @elseif ($item->illustration)
                                <x-illustration :name="$item->illustration" class="size-full"/>
                            @endif
                            <span class="sr-only">{{ $item->title }}{{ $item->variantLabel ? ' · '.$item->variantLabel : '' }} · {{ __($t.'quantity', ['count' => fa_digits($item->quantity)]) }} · {{ $item->lineTotal->formatShort() }}</span>
                        </li>
                    @endforeach
                </ul>
                <x-ui.timeline :items="$timeline" :label="__($t.'status')"/>
            </section>

            <section aria-labelledby="order-details" class="flex-1 rounded-5xl border border-line bg-surface px-7 pt-6 pb-4 max-lg:basis-75 max-sm:basis-full max-sm:px-5.5">
                <h2 id="order-details" class="m-0 flex items-center gap-2 text-lg font-extrabold"><x-icon name="package" class="size-[19px] text-primary"/>{{ __($t.'details') }}</h2>
                <dl class="m-0 mt-2.5">
                    <div class="flex justify-between gap-3 py-3 text-[14.5px] font-bold max-lg:flex-wrap">
                        <dt class="text-muted">{{ __($t.'items_total', ['count' => fa_digits($order->itemsCount)]) }}</dt>
                        <dd class="m-0">{{ $order->subtotal->formatShort() }}</dd>
                    </div>
                    <div class="{{ $row }}">
                        <dt class="text-muted">{{ __($t.'shipping') }}</dt>
                        <dd class="m-0">{{ $fee }}</dd>
                    </div>
                    <div class="{{ $row }} text-lg font-extrabold">
                        <dt class="text-ink">{{ __($t.'payable') }}</dt>
                        <dd class="m-0">{{ $order->total->format() }}</dd>
                    </div>
                    <div class="{{ $row }}">
                        <dt class="text-muted">{{ __($t.'payment') }}</dt>
                        <dd class="m-0">{{ $order->paymentMethod->label() }}</dd>
                    </div>
                    <div class="{{ $row }}">
                        <dt class="text-muted">{{ __($t.'delivery') }}</dt>
                        <dd class="m-0">{{ jdate($order->deliveryDate, 'l j F') }} · {{ $order->deliveryWindow->hours() }}</dd>
                    </div>
                    @if ($owner)
                        <div class="{{ $row }}">
                            <dt class="text-muted">{{ __($t.'recipient') }}</dt>
                            <dd class="m-0">{{ $order->recipientName }} · <span dir="ltr">{{ $order->maskedMobile }}</span></dd>
                        </div>
                        <div class="{{ $row }}">
                            <dt class="text-muted">{{ __($t.'destination') }}</dt>
                            <dd class="m-0">{{ $order->place }}</dd>
                        </div>
                    @endif
                </dl>
                @if ($order->discreetPackaging)
                    <p class="m-0 mt-1 flex items-center gap-2 text-sm font-semibold text-muted"><x-icon name="lock" class="size-4 shrink-0 text-stage-teen"/>{{ __($t.'discreet') }}</p>
                @endif
                @if ($order->isDemo)
                    <p class="m-0 mt-2 flex items-center gap-2 text-sm font-semibold text-muted"><x-icon name="info" class="size-4 shrink-0 text-primary"/>{{ __($t.'demo_note') }}</p>
                @endif
            </section>
        </div>

        <div class="flex flex-wrap justify-center gap-3">
            <x-ui.button :href="route('shop.index')" size="xl" icon="store">{{ __($t.'continue') }}</x-ui.button>
            <x-ui.button :href="route('contact')" variant="outline" size="xl">{{ __($t.'support') }}</x-ui.button>
        </div>
        <span class="text-base font-semibold text-muted">{{ __($t.'returns') }} <a href="{{ route('contact') }}" class="font-extrabold">{{ __($t.'returns_link') }}</a></span>

        <x-ui.app-cta variant="strip" :title="__($t.'app.title')" :lead="__($t.'app.lead')" :links="$appLinks" class="mt-5 w-250 max-w-full"/>
    </section>
@endsection
