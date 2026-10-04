{{--
    <x-directory.booking :price-from="320000" unit="هر جلسه" phone="+982100000000" :form="$booking['form']"/>
    Booking panel of the place page (`#book`, the target of every «انتخاب» link). `form` (ShowPlaceController::bookingForm)
    turns the `data-booking-slot` block into the booking REQUEST form (L5-04): service, preferred day (radio chips of the
    next open days — plain radios, no JS), time window (a preference, not a slot), name, mobile, optional child age and
    note → POST BookingController@store. No fake availability: days come from the opening hours only, and the place
    confirms the time by phone. Works on page-cache HITs: `@csrf` is swapped per visitor by PageCache, the FormTimer
    token is the cache-store time (see App\Http\Requests\BookingRequest), and errors / old input come back as flash
    data in the `booking` bag, which bypasses the page cache. Without `form` the panel shows how to reach the place;
    `:form="false"` marks a phone-only place (BookingMode::Phone): the same panel without the «online booking soon» note.
--}}
@props(['priceFrom' => null, 'unit' => null, 'phone' => null, 'form' => null])
@php
    $t = 'directory.booking.';
    $e = $errors->getBag('booking');
    $failed = $e->any();
@endphp
<aside id="book" aria-labelledby="book-title" {{ $attributes->class('box-content flex w-100 shrink-0 flex-col gap-4.5 rounded-6xl border border-line bg-surface p-7 shadow-booking max-sm:box-border max-sm:w-full max-sm:max-w-full') }}>
    <h2 id="book-title" class="sr-only">{{ __('directory.place.booking.label') }}</h2>
    @if ($priceFrom)
        <x-ui.price :amount="$priceFrom" from :unit="$unit ?? __('directory.place.booking.unit')" size="lg"/>
    @endif
    @if ($form)
        <form method="post" action="{{ $form['action'] }}" data-booking-slot aria-label="{{ __($t.'form_label') }}" class="relative m-0 flex flex-col gap-4.5" novalidate>
            @csrf
            <input type="hidden" name="{{ $form['timer'] }}" value="{{ $form['token'] }}">
            <p class="m-0 text-sm leading-[1.8] font-semibold text-muted">{{ __($t.'intro') }}</p>
            @if ($failed)
                <p role="alert" class="m-0 rounded-2xl bg-danger-soft px-4 py-3 text-sm font-bold text-danger">{{ $e->first($form['timer']) ?: __($t.'errors_title') }}</p>
            @endif

            @if ($form['services'] !== [])
                <x-ui.form.field for="book-service" :label="__($t.'service')" :error="$e->first('service')" required>
                    <x-ui.form.select id="book-service" name="service" :options="collect($form['services'])->pluck('label', 'id')->all()"
                                      :selected="old('service', count($form['services']) === 1 ? $form['services'][0]['id'] : null)" :placeholder="__($t.'service_placeholder')"
                                      :invalid="$e->has('service')" aria-describedby="book-service-error" required class="min-h-13 text-[14.5px] font-bold"/>
                </x-ui.form.field>
            @endif

            <fieldset class="m-0 flex min-w-0 flex-col gap-2 border-0 p-0" @if ($e->has('date')) aria-describedby="book-date-error" @endif>
                <legend class="mb-2 flex w-full justify-between p-0 text-sm font-extrabold text-ink">
                    <span>{{ __($t.'day') }} <span aria-hidden="true" class="text-danger">*</span></span>
                    @if ($form['month'])<span class="text-muted">{{ $form['month'] }}</span>@endif
                </legend>
                <div class="grid grid-cols-5 gap-1.5 max-lg:grid-cols-3 max-sm:grid-cols-2">
                    @foreach ($form['days'] as $day)
                        <label class="flex h-15 cursor-pointer flex-col items-center justify-center gap-0.5 rounded-xl border-[1.5px] border-line bg-surface text-ink has-checked:border-primary has-checked:bg-primary has-checked:text-white has-focus-visible:outline-2 has-focus-visible:outline-primary">
                            <input type="radio" name="date" value="{{ $day['value'] }}" class="sr-only" aria-label="{{ $day['label'] }}" required @checked(old('date') === $day['value'])>
                            <span aria-hidden="true" class="text-2xs font-bold">{{ $day['weekday'] }}</span>
                            <b aria-hidden="true" class="text-lg">{{ $day['day'] }}</b>
                        </label>
                    @endforeach
                </div>
                @if ($e->has('date'))<span id="book-date-error" class="text-sm font-bold text-danger">{{ $e->first('date') }}</span>@endif
            </fieldset>

            <fieldset class="m-0 flex min-w-0 flex-col gap-2 border-0 p-0" @if ($e->has('time_window')) aria-describedby="book-window-error" @endif>
                <legend class="mb-2 p-0 text-sm font-extrabold text-ink">{{ __($t.'window') }} <span aria-hidden="true" class="text-danger">*</span></legend>
                <div class="grid grid-cols-2 gap-1.5">
                    @foreach ($form['windows'] as $window)
                        <label class="flex min-h-11.5 cursor-pointer flex-col items-center justify-center rounded-lg border-[1.5px] border-line bg-surface px-2 py-1.5 text-center text-ink has-checked:border-primary has-checked:bg-lavender has-focus-visible:outline-2 has-focus-visible:outline-primary">
                            <input type="radio" name="time_window" value="{{ $window['value'] }}" class="sr-only" required @checked(old('time_window') === $window['value'])>
                            <span class="text-base font-extrabold">{{ $window['label'] }}</span>
                            @if ($window['hours'])<span class="text-2xs font-bold text-muted">{{ $window['hours'] }}</span>@endif
                        </label>
                    @endforeach
                </div>
                @if ($e->has('time_window'))<span id="book-window-error" class="text-sm font-bold text-danger">{{ $e->first('time_window') }}</span>@endif
            </fieldset>

            <x-ui.form.field for="book-name" :label="__($t.'name')" :error="$e->first('name')" required>
                <x-ui.form.input id="book-name" name="name" :value="old('name')" maxlength="100" autocomplete="name" required :invalid="$e->has('name')" described class="min-h-13"/>
            </x-ui.form.field>

            <x-ui.form.field for="book-mobile" :label="__($t.'mobile')" :hint="__($t.'mobile_hint')" :error="$e->first('mobile')" required>
                <x-ui.form.input id="book-mobile" name="mobile" type="tel" inputmode="tel" dir="ltr" :value="old('mobile')" maxlength="30" autocomplete="tel" placeholder="09xx xxx xxxx"
                                 required :invalid="$e->has('mobile')" described class="min-h-13 text-end"/>
            </x-ui.form.field>

            <x-ui.form.field for="book-age" :label="__($t.'child_age')" :hint="__($t.'child_age_hint')" :error="$e->first('child_age') ?: $e->first('child_age_unit')">
                <span class="flex gap-2">
                    <x-ui.form.input id="book-age" name="child_age" type="text" inputmode="numeric" :value="old('child_age')" maxlength="3" :invalid="$e->has('child_age')" described class="min-h-13"/>
                    <x-ui.form.select name="child_age_unit" :options="['month' => __($t.'months'), 'year' => __($t.'years')]" :selected="old('child_age_unit', 'month')"
                                      aria-label="{{ __($t.'child_age') }}" class="min-h-13 w-28"/>
                </span>
            </x-ui.form.field>

            <x-ui.form.field for="book-note" :label="__($t.'note')" :hint="__($t.'note_hint')" :error="$e->first('note')">
                <x-ui.form.textarea id="book-note" name="note" rows="3" maxlength="500" :invalid="$e->has('note')" described class="min-h-24">{{ old('note') }}</x-ui.form.textarea>
            </x-ui.form.field>

            {{-- Honeypot: off-screen, out of the tab order; a filled value is answered like a real submit and dropped. --}}
            <div aria-hidden="true" class="absolute -start-[9999px] top-0 size-px overflow-hidden">
                <label for="book-{{ $form['honeypot'] }}">{{ __($t.'honeypot') }}</label>
                <input type="text" id="book-{{ $form['honeypot'] }}" name="{{ $form['honeypot'] }}" value="" tabindex="-1" autocomplete="off">
            </div>

            <x-ui.button type="submit" size="2xl" class="w-full justify-center shadow-primary">{{ __($t.'submit') }}</x-ui.button>
            @if ($phone)
                <x-ui.button :href="'tel:'.$phone" variant="outline" size="lg" icon="phone" class="w-full justify-center">{{ __($t.'call') }}</x-ui.button>
            @endif
        </form>
    @else
        <div data-booking-slot class="flex flex-col gap-4.5">
            @if ($form !== false)
                <p class="m-0 rounded-2xl bg-lavender px-4 py-3.5 text-[14.5px] leading-loose font-semibold text-ink">{{ __('directory.place.booking.soon') }}</p>
            @endif
            @if ($phone)
                <x-ui.button :href="'tel:'.$phone" size="2xl" icon="phone" class="w-full justify-center">{{ __('directory.place.booking.call') }}</x-ui.button>
            @endif
            <x-ui.button href="#services" :variant="$phone ? 'outline' : 'primary'" size="2xl" class="w-full justify-center">{{ __('directory.place.booking.services') }}</x-ui.button>
            <x-ui.button href="#address" variant="outline" size="2xl" icon="navigate" class="w-full justify-center">{{ __('directory.place.booking.directions') }}</x-ui.button>
        </div>
    @endif
    <div class="flex flex-col gap-2 text-sm leading-[1.8] font-semibold text-muted">
        @if (($form['cancellation'] ?? null) !== null)
            <span class="flex gap-2"><x-icon name="check" class="mt-1 size-4 shrink-0 text-stage-teen"/>{{ $form['cancellation'] }}</span>
        @endif
        <span class="flex gap-2"><x-icon name="lock" class="mt-1 size-4 shrink-0 text-stage-teen"/>{{ __('directory.place.booking.privacy') }}</span>
    </div>
</aside>
