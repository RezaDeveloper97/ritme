{{--
    <x-shop.cart-summary :cart="$cart"/>
    «خلاصه سفارش» card of the cart page (L6-04): item total (available lines), shipping estimate (known fee, «رایگان»
    or «محاسبه در مرحله بعد»), payable amount and the checkout link (disabled look + no link when nothing is buyable).
--}}
@props(['cart'])
@php
    /** @var \App\Domain\Shop\Cart\Data\CartSummary $cart */
    $shipping = $cart->shipping;
    $fee = match (true) {
        $cart->count === 0 || ! $shipping->isKnown() => __('shop.cart.shipping_later'),
        $shipping->free || $shipping->fee->isZero() => __('shop.cart.shipping_free'),
        default => $shipping->fee->formatShort(),
    };
@endphp
<div {{ $attributes->class('rounded-5xl border border-line bg-surface p-7') }}>
    <h2 id="cart-summary-title" class="m-0 text-2xl font-bold">{{ __('shop.cart.summary') }}</h2>
    <dl class="m-0 mt-2.5">
        <div class="flex justify-between py-3 text-[14.5px] font-bold max-lg:flex-wrap">
            <dt class="text-muted">{{ __('shop.cart.items_total', ['count' => fa_digits($cart->count)]) }}</dt>
            <dd class="m-0">{{ $cart->subtotal->formatShort() }}</dd>
        </div>
        <div class="flex justify-between border-t border-line py-3 text-[14.5px] font-bold max-lg:flex-wrap">
            <dt class="text-muted">{{ __('shop.cart.shipping') }}</dt>
            <dd class="m-0">{{ $fee }}</dd>
        </div>
        <div class="flex justify-between border-t border-line py-3 text-lg font-extrabold max-lg:flex-wrap">
            <dt class="text-ink">{{ __('shop.cart.payable') }}</dt>
            <dd class="m-0">{{ $cart->total->formatShort() }}</dd>
        </div>
    </dl>
    @if ($cart->canCheckout())
        <a href="{{ route('shop.checkout') }}" class="mt-3.5 flex h-14 items-center justify-center rounded-full bg-primary text-lg font-extrabold text-white shadow-primary hover:text-white">{{ __('shop.cart.checkout') }}</a>
    @else
        <span aria-disabled="true" class="mt-3.5 flex h-14 cursor-not-allowed items-center justify-center rounded-full bg-primary text-lg font-extrabold text-white opacity-50">{{ __('shop.cart.checkout') }}</span>
    @endif
</div>
