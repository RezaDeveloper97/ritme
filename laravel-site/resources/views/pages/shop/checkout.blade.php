{{--
    Checkout (L6-05, design/html/shop-checkout.html): /shop/checkout — noindex, no-store, never page-cached.
    Data (Shop\CheckoutController::show): $cart (CartSummary, live), $state ready|empty|sold_out|cod_limit, $soldOut
    (titles), $error / $notices (flash after a refused order), $slots (DeliverySlot list) + $selectedSlot, $provinces
    (key ⇒ name), $cities (suggestions), $paymentMethod, $codLimit (formatted cap or null), $token (one-time
    idempotency token), $signature (CartSignature of what is shown), $formToken (FormTimer).
    Audit corrections (AUDIT §8): single seller → one delivery group; the design's saved-address card becomes a real
    address form (no accounts); payment renders ONLY cash on delivery (no bank gateway option). Nothing is paid online:
    the button says «ثبت سفارش», the summary says «قابل پرداخت هنگام تحویل».
    No JS needed: slot cards and the discreet-packaging switch are real radios / a checkbox styled with has-checked:.
--}}
@extends('layouts.app')

@php
    $t = 'shop.checkout.';
    $steps = [['label' => __($t.'steps.cart')], ['label' => __($t.'steps.delivery')], ['label' => __($t.'steps.confirm')]];
    $shipping = $cart->shipping;
    $fee = match (true) {
        ! $shipping->isKnown() => __($t.'shipping_later'),
        $shipping->free || $shipping->fee->isZero() => __('shop.cart.shipping_free'),
        default => $shipping->fee->formatShort(),
    };
    $card = 'rounded-5xl border border-line bg-surface p-7 max-sm:p-5.5';
    $badge = 'flex size-8.5 shrink-0 items-center justify-center rounded-full bg-primary text-base font-extrabold text-white';
    $canSubmit = $state === 'ready';
@endphp

@section('content')
    <section aria-labelledby="checkout-title" class="flex flex-col gap-6 px-30 pt-9 pb-24 max-lg:px-5 max-lg:pb-[52.8px]">
        <div class="flex items-center justify-between gap-4 max-lg:flex-wrap">
            <h1 id="checkout-title" class="m-0 font-display text-d-xl leading-display font-normal text-ink">{{ __($t.'title') }}</h1>
            <x-ui.stepper variant="bar" :steps="$steps" :current="2" :label="__($t.'steps_label')" class="w-105 max-sm:w-full"/>
        </div>

        @if ($error !== null || $notices !== [] || $errors->any())
            <div role="alert" class="rounded-[22px] border border-danger-line bg-danger-soft p-4.5 text-md leading-relaxed font-semibold text-ink">
                @if ($error !== null)<b class="block text-base">{{ $error }}</b>@endif
                @foreach ($notices as $notice)<span class="block">{{ $notice }}</span>@endforeach
                @if ($errors->any() && $error === null)<b class="block text-base">{{ __($t.'errors_title') }}</b>@endif
                @if ($errors->has('form_token'))<span class="block">{{ $errors->first('form_token') }}</span>@endif
            </div>
        @endif

        @if ($state === 'empty' || $state === 'sold_out')
            <div class="flex flex-col items-start gap-4 {{ $card }}">
                <span aria-hidden="true" class="flex size-14 items-center justify-center rounded-full bg-primary/13"><x-icon name="cart" class="size-6 text-primary"/></span>
                @if ($state === 'empty')
                    <h2 class="m-0 text-2xl font-bold">{{ __($t.'empty_title') }}</h2>
                    <p class="m-0 text-base leading-loose font-medium text-muted">{{ __($t.'empty_text') }}</p>
                    <x-ui.button :href="route('shop.index')" size="xl" icon="store">{{ __($t.'empty_cta') }}</x-ui.button>
                @else
                    <p class="m-0 text-base leading-loose font-semibold text-ink">{{ __($t.'sold_out', ['names' => implode('، ', $soldOut)]) }}</p>
                    <x-ui.button :href="route('shop.cart')" size="xl" icon="cart">{{ __($t.'sold_out_cta') }}</x-ui.button>
                @endif
            </div>
        @else
            <form id="checkout-form" method="post" action="{{ route('shop.checkout.store') }}" novalidate class="flex items-start gap-8 max-lg:flex-wrap">
                @csrf
                <input type="hidden" name="checkout_token" value="{{ $token }}">
                <input type="hidden" name="cart_signature" value="{{ $signature }}">
                <input type="hidden" name="{{ \App\Http\Requests\CheckoutRequest::TIMER }}" value="{{ $formToken }}">
                <div aria-hidden="true" class="sr-only">
                    <label for="checkout-website">{{ __($t.'honeypot') }}</label>
                    <input id="checkout-website" type="text" name="{{ \App\Http\Requests\CheckoutRequest::HONEYPOT }}" value="" tabindex="-1" autocomplete="off">
                </div>

                <div class="flex flex-1 flex-col gap-4.5 max-lg:basis-75 max-sm:basis-full">
                    {{-- 1 · آدرس تحویل --}}
                    <fieldset class="m-0 min-w-0 {{ $card }}">
                        <legend class="sr-only">{{ __($t.'address') }}</legend>
                        <div class="flex items-center justify-between max-lg:flex-wrap">
                            <div class="flex items-center gap-3">
                                <span aria-hidden="true" class="{{ $badge }}">{{ fa_digits(1) }}</span>
                                <b class="text-[19px]">{{ __($t.'address') }}</b>
                            </div>
                            <a href="{{ route('shop.cart') }}" class="text-base font-extrabold">{{ __($t.'edit_cart') }}</a>
                        </div>
                        <div class="mt-4.5 grid grid-cols-3 gap-4 max-lg:grid-cols-2 max-sm:grid-cols-1">
                            <x-ui.form.field for="checkout-name" :label="__($t.'name')" :error="$errors->first('name')" required>
                                <x-ui.form.input id="checkout-name" name="name" :value="old('name')" autocomplete="name" maxlength="{{ \App\Http\Requests\CheckoutRequest::NAME_MAX }}" required :invalid="$errors->has('name')" described/>
                            </x-ui.form.field>
                            <x-ui.form.field for="checkout-mobile" :label="__($t.'mobile')" :error="$errors->first('mobile')" required>
                                <x-ui.form.input id="checkout-mobile" name="mobile" type="tel" inputmode="tel" dir="ltr" :value="old('mobile')" autocomplete="tel" placeholder="۰۹۱۲۳۴۵۶۷۸۹" required :invalid="$errors->has('mobile')" described class="text-end"/>
                            </x-ui.form.field>
                            <x-ui.form.field for="checkout-province" :label="__($t.'province')" :error="$errors->first('province')" required>
                                <x-ui.form.select id="checkout-province" name="province" :options="$provinces" :selected="old('province')" :placeholder="__($t.'province_placeholder')" autocomplete="address-level1" required :invalid="$errors->has('province')" aria-describedby="checkout-province-error"/>
                            </x-ui.form.field>
                            <x-ui.form.field for="checkout-city" :label="__($t.'city')" :error="$errors->first('city')" required>
                                <x-ui.form.input id="checkout-city" name="city" list="checkout-cities" :value="old('city')" autocomplete="address-level2" maxlength="{{ \App\Http\Requests\CheckoutRequest::CITY_MAX }}" required :invalid="$errors->has('city')" described/>
                                <datalist id="checkout-cities">
                                    @foreach ($cities as $city)<option value="{{ $city }}"></option>@endforeach
                                </datalist>
                            </x-ui.form.field>
                            <x-ui.form.field for="checkout-postal" :label="__($t.'postal_code')" :error="$errors->first('postal_code')">
                                <x-ui.form.input id="checkout-postal" name="postal_code" inputmode="numeric" dir="ltr" :value="old('postal_code')" :placeholder="__($t.'postal_code_hint')" autocomplete="postal-code" maxlength="14" :invalid="$errors->has('postal_code')" described class="text-end"/>
                            </x-ui.form.field>
                            <x-ui.form.field for="checkout-note" :label="__($t.'note')" :error="$errors->first('note')">
                                <x-ui.form.input id="checkout-note" name="note" :value="old('note')" :placeholder="__($t.'note_placeholder')" maxlength="{{ \App\Http\Requests\CheckoutRequest::NOTE_MAX }}" :invalid="$errors->has('note')" described/>
                            </x-ui.form.field>
                            <x-ui.form.field for="checkout-address" :label="__($t.'street')" :error="$errors->first('address')" required class="col-span-full">
                                <x-ui.form.input id="checkout-address" name="address" :value="old('address')" :placeholder="__($t.'street_placeholder')" autocomplete="street-address" maxlength="{{ \App\Http\Requests\CheckoutRequest::ADDRESS_MAX }}" required :invalid="$errors->has('address')" described/>
                            </x-ui.form.field>
                        </div>
                        <p class="m-0 mt-3 text-sm leading-relaxed font-semibold text-muted">{{ __($t.'mobile_hint') }}</p>
                    </fieldset>

                    {{-- 2 · زمان تحویل (one seller → one group of preferred slots) --}}
                    <fieldset class="m-0 min-w-0 {{ $card }}" aria-describedby="checkout-delivery-hint">
                        <legend class="sr-only">{{ __($t.'delivery') }}</legend>
                        <div class="flex items-center gap-3">
                            <span aria-hidden="true" class="{{ $badge }}">{{ fa_digits(2) }}</span>
                            <b class="text-[19px]">{{ __($t.'delivery') }}</b>
                        </div>
                        <div class="mt-4.5 flex flex-col gap-3">
                            <span class="flex items-center gap-2 text-base font-extrabold text-muted"><x-icon name="store" class="size-[17px] text-primary"/>{{ __($t.'delivery_group', ['seller' => __('shop.cart.seller'), 'count' => fa_digits($cart->count)]) }}</span>
                            <div class="grid grid-cols-4 gap-2.5 max-lg:grid-cols-2">
                                @foreach ($slots as $slot)
                                    <label class="flex cursor-pointer flex-col items-center gap-1 rounded-2xl border-[1.5px] border-line bg-surface p-3.5 text-ink has-checked:border-primary has-checked:bg-lavender has-focus-visible:outline-2 has-focus-visible:outline-primary">
                                        <input type="radio" name="delivery" value="{{ $slot->value() }}" @checked($selectedSlot === $slot->value()) class="sr-only">
                                        <b class="text-md">{{ $slot->dayLabel() }}</b>
                                        <span class="text-sm font-semibold text-muted">{{ $slot->window->hours() }}</span>
                                    </label>
                                @endforeach
                            </div>
                            <span id="checkout-delivery-hint" class="text-sm leading-relaxed font-semibold text-muted">{{ __($t.'delivery_hint') }}</span>
                            @if ($errors->has('delivery'))<span class="text-sm font-bold text-danger" role="alert">{{ $errors->first('delivery') }}</span>@endif
                        </div>
                    </fieldset>

                    {{-- بسته‌بندی ساده: a real checkbox drawn as the design's switch --}}
                    <label class="flex cursor-pointer items-center gap-4 rounded-5xl border border-line bg-surface px-7 py-5.5 max-sm:px-5.5">
                        <span aria-hidden="true" class="flex size-12 shrink-0 items-center justify-center rounded-full bg-primary/13"><x-icon name="package" class="size-6 text-primary"/></span>
                        <span class="grow">
                            <b class="text-xl">{{ __($t.'discreet') }}</b>
                            <span class="block text-base font-semibold text-muted">{{ __($t.'discreet_text') }}</span>
                        </span>
                        <span class="relative h-8 w-13 shrink-0">
                            <input type="checkbox" role="switch" name="discreet" value="1" @checked(old('discreet', old('name') === null ? '1' : null) === '1') class="peer absolute inset-0 m-0 size-full cursor-pointer appearance-none rounded-full bg-line checked:bg-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary">
                            <span aria-hidden="true" class="pointer-events-none absolute start-1 top-1 size-6 rounded-full bg-surface peer-checked:start-6"></span>
                        </span>
                    </label>

                    {{-- 3 · روش پرداخت — cash on delivery only (AUDIT §8) --}}
                    <fieldset class="m-0 min-w-0 {{ $card }}">
                        <legend class="sr-only">{{ __($t.'payment') }}</legend>
                        <div class="flex items-center gap-3">
                            <span aria-hidden="true" class="{{ $badge }}">{{ fa_digits(3) }}</span>
                            <b class="text-[19px]">{{ __($t.'payment') }}</b>
                        </div>
                        <div class="mt-4.5 grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                            <x-ui.form.radio-card name="payment" :value="$paymentMethod->value" icon="credit-card" :title="$paymentMethod->label()"
                                :text="$codLimit === null ? __($t.'cod_text') : __($t.'cod_limit_text', ['amount' => $codLimit])" :checked="true"/>
                        </div>
                    </fieldset>
                </div>

                <aside aria-labelledby="checkout-summary-title" class="flex w-100 shrink-0 flex-col gap-4 max-sm:w-full max-sm:max-w-full">
                    <div class="{{ $card }}">
                        <h2 id="checkout-summary-title" class="m-0 text-2xl font-bold">{{ __($t.'summary') }}</h2>
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
                                <dt class="text-ink">{{ __($t.'payable') }}</dt>
                                <dd class="m-0">{{ $cart->total->format() }}</dd>
                            </div>
                        </dl>
                        @if ($canSubmit)
                            <button type="submit" class="mt-3.5 flex h-14 w-full cursor-pointer items-center justify-center rounded-full border-0 bg-primary text-lg font-extrabold text-white shadow-primary">{{ __($t.'submit') }}</button>
                        @else
                            <p class="m-0 mt-3.5 rounded-2xl bg-danger-soft px-4 py-3 text-sm-plus leading-relaxed font-bold text-danger">{{ __($t.'cod_limit', ['amount' => $codLimit ?? '']) }}</p>
                            <button type="submit" disabled class="mt-3.5 flex h-14 w-full cursor-not-allowed items-center justify-center rounded-full border-0 bg-primary text-lg font-extrabold text-white opacity-50">{{ __($t.'submit') }}</button>
                        @endif
                        <p class="m-0 mt-2.5 text-center text-sm font-semibold text-muted">{{ __($t.'submit_note') }}</p>
                    </div>
                    <p class="m-0 flex gap-2.5 px-2 text-sm-plus leading-relaxed font-semibold text-muted">
                        <x-icon name="lock" class="size-4.5 shrink-0 text-stage-teen"/>{{ __($t.'privacy') }}
                    </p>
                </aside>
            </form>
        @endif
    </section>
@endsection
