{{--
    Join form (L5-05), design/html/directory-join.html. Controller: App\Http\Controllers\Directory\JoinController@create.
      $categories, $amenities  list<array{id, name}>          $cities  list<array{id, name, districts: list<array{id, name}>}>
      $ageGroups    array<value, label> (AgeGroup)             $bookingModes  list<array{value, title, text}>
      $weekdays     list<array{key, label}> (Saturday first)   $defaultHours  array<key, array{open, opens, closes}>
      $photoLimits  array{max, recommended, bytes, total, mimes}  $formToken  FormTimer time-trap token
    ONE long multipart form in four steps. Without JS every step is visible and «ارسال درخواست» posts everything; the
    lazy `stepper` module (resources/js/modules/stepper.js) shows one step at a time, checks the current step before
    «ذخیره و ادامه», checks photo count/size/type on selection and mirrors the name into the preview card. The server
    validates everything (App\Http\Requests\JoinRequest); after a failed post the first step with an error opens.
    No map widget (no external requests): address + optional latitude/longitude. Copy: lang/fa/directory.php `join`.
--}}
@extends('layouts.app')

@php
    $t = 'directory.join.';
    $total = \App\Domain\Directory\Join\Support\JoinForm::STEPS;
    $stepOf = \App\Http\Requests\JoinRequest::STEP_OF;
    $start = 1;
    if ($errors->any()) {
        $start = $total;
        foreach (array_keys($errors->getMessages()) as $key) {
            $start = min($start, $stepOf[explode('.', (string) $key)[0]] ?? $total);
        }
    }
    $stepLabels = __($t.'steps');
    $navSteps = static fn (int $current): array => array_map(
        static fn (string $label, int $i): array => [
            'label' => $label,
            'hint' => $i + 1 < $current ? __($t.'hint_done') : ($i + 1 === $current ? __($t.'hint_current') : ''),
        ],
        $stepLabels,
        array_keys($stepLabels),
    );
    $oldHours = old('hours');
    $selectedAges = array_map('strval', (array) old('ages', []));
    $selectedAmenities = array_map('strval', (array) old('amenities', []));
    $selectedCity = (string) old('city', count($cities) === 1 ? (string) $cities[0]['id'] : '');
    $mb = static fn (int $bytes): string => fa_digits(rtrim(rtrim(number_format($bytes / 1048576, 1, '.', ''), '0'), '.'));
    $chip = 'flex h-10 cursor-pointer items-center rounded-full border-[1.5px] border-line bg-transparent px-3.5 text-[12.5px] font-bold text-muted transition-colors hover:border-primary has-checked:border-primary has-checked:bg-primary/13 has-checked:text-ink has-focus-visible:outline-2 has-focus-visible:outline-primary';
    $subhead = 'm-0 font-sans text-3xl leading-[inherit] font-extrabold';
    $stepHead = 'm-0 font-display text-[34px] leading-heading font-normal text-ink';
@endphp

@section('content')
    <section aria-labelledby="join-title" class="flex flex-col gap-7 px-30 pt-10 pb-24 max-lg:px-5 max-lg:pt-6 max-lg:pb-[52.8px]">
        <div class="flex items-center justify-between max-lg:flex-wrap">
            <h1 id="join-title" class="m-0 font-display text-d-xl leading-display font-normal text-ink">{{ __($t.'title') }}</h1>
            <a href="{{ route('directory.business') }}" class="text-base font-extrabold">{{ __($t.'exit') }}</a>
        </div>

        <form id="join-form" method="post" action="{{ route('directory.join.store') }}" enctype="multipart/form-data" novalidate
              data-module="stepper" data-start="{{ $start }}" aria-labelledby="join-title"
              class="flex scroll-mt-6 items-start gap-8 max-lg:flex-wrap">
            @csrf

            <div class="flex w-70 shrink-0 flex-col max-sm:w-full">
                @for ($n = 1; $n <= $total; $n++)
                    <div data-stepper-nav="{{ $n }}" @if ($n !== $start) hidden @endif>
                        <x-ui.stepper :steps="$navSteps($n)" :current="$n" :label="__($t.'steps_label')"/>
                    </div>
                @endfor
                <p class="m-0 mt-5 rounded-3xl bg-lavender p-4 text-sm-plus leading-relaxed font-semibold text-muted">{{ __($t.'note') }}</p>
            </div>

            <div class="flex flex-1 flex-col gap-5.5 rounded-6xl border border-line bg-surface p-9 max-lg:basis-93.5 max-sm:basis-full max-sm:p-5.5">
                @if ($errors->any())
                    <div role="alert" class="rounded-[22px] border border-danger-line bg-danger-soft p-4.5 text-md leading-relaxed font-semibold text-ink">
                        <b class="block text-base">{{ __($t.'errors_title') }}</b>
                        @if ($errors->has('form_token'))<span class="block">{{ $errors->first('form_token') }}</span>@endif
                        <span class="block text-sm text-muted">{{ __($t.'photos_again') }}</span>
                    </div>
                @endif

                {{-- Step 1 — معرفی --}}
                <div id="join-step-1" data-step="1" role="group" aria-labelledby="join-step-1-title" class="flex scroll-mt-6 flex-col gap-5.5">
                    <div class="flex flex-col gap-1.5">
                        <span class="text-sm font-extrabold text-primary">{{ __($t.'step_of', ['n' => fa_digits(1), 'total' => fa_digits($total)]) }}</span>
                        <h2 id="join-step-1-title" class="{{ $stepHead }}">{{ $stepLabels[0] }}</h2>
                    </div>
                    <x-ui.form.field for="join-name" :label="__($t.'intro.name')" :error="$errors->first('name')" required>
                        <x-ui.form.input id="join-name" name="name" required minlength="2" maxlength="{{ \App\Domain\Directory\Join\Support\JoinForm::NAME_MAX }}" autocomplete="organization"
                                         data-preview-name :placeholder="__($t.'intro.name_placeholder')" :value="old('name')" :invalid="$errors->has('name')" described/>
                    </x-ui.form.field>
                    <div class="grid grid-cols-3 gap-3.5 max-lg:grid-cols-1">
                        <x-ui.form.field for="join-category" :label="__($t.'intro.category')" :error="$errors->first('category')" required>
                            <x-ui.form.select id="join-category" name="category" required :options="array_column($categories, 'name', 'id')" :selected="old('category')"
                                              :placeholder="__($t.'intro.category_placeholder')" :invalid="$errors->has('category')" aria-describedby="join-category-error"/>
                        </x-ui.form.field>
                        <x-ui.form.field for="join-city" :label="__($t.'intro.city')" :error="$errors->first('city')" required>
                            <x-ui.form.select id="join-city" name="city" required data-city-select :options="array_column($cities, 'name', 'id')" :selected="$selectedCity"
                                              :placeholder="__($t.'intro.city_placeholder')" :invalid="$errors->has('city')" aria-describedby="join-city-error"/>
                        </x-ui.form.field>
                        <x-ui.form.field for="join-district" :label="__($t.'intro.district')" :error="$errors->first('district')">
                            <span class="relative block">
                                <select id="join-district" name="district" data-district-select @if ($errors->has('district')) aria-invalid="true" @endif aria-describedby="join-district-error"
                                        @class(['min-h-14 w-full appearance-none rounded-2xl border-[1.5px] bg-surface ps-4.5 pe-11 text-md font-semibold text-ink', $errors->has('district') ? 'border-danger' : 'border-line focus:border-primary'])>
                                    <option value="">{{ __($t.'intro.district_placeholder') }}</option>
                                    @foreach ($cities as $city)
                                        @if ($city['districts'] !== [])
                                            <optgroup label="{{ $city['name'] }}" data-city="{{ $city['id'] }}">
                                                @foreach ($city['districts'] as $district)
                                                    <option value="{{ $district['id'] }}" @selected((string) old('district') === (string) $district['id'])>{{ $district['name'] }}</option>
                                                @endforeach
                                            </optgroup>
                                        @endif
                                    @endforeach
                                </select>
                                <x-icon name="chevron-left" class="pointer-events-none absolute end-4 top-1/2 size-4 -translate-y-1/2 -rotate-90 text-muted"/>
                            </span>
                        </x-ui.form.field>
                    </div>
                    <div>
                        <h3 class="{{ $subhead }}">{{ __($t.'intro.contact') }}</h3>
                        <p class="m-0 mt-1 text-sm-plus font-semibold text-muted">{{ __($t.'intro.contact_hint') }}</p>
                    </div>
                    <div class="grid grid-cols-3 gap-3.5 max-lg:grid-cols-1">
                        <x-ui.form.field for="join-contact-name" :label="__($t.'intro.contact_name')" :error="$errors->first('contact_name')" required>
                            <x-ui.form.input id="join-contact-name" name="contact_name" required minlength="2" maxlength="100" autocomplete="name"
                                             :value="old('contact_name')" :invalid="$errors->has('contact_name')" described/>
                        </x-ui.form.field>
                        <x-ui.form.field for="join-mobile" :label="__($t.'intro.mobile')" :error="$errors->first('mobile')" required>
                            <x-ui.form.input id="join-mobile" name="mobile" type="tel" required maxlength="30" autocomplete="tel" inputmode="tel" dir="ltr" placeholder="09…"
                                             pattern="[0-9۰-۹+\s\-]{10,16}" :value="old('mobile')" :invalid="$errors->has('mobile')" described class="text-end"/>
                        </x-ui.form.field>
                        <x-ui.form.field for="join-email" :label="__($t.'intro.email')" :error="$errors->first('email')">
                            <x-ui.form.input id="join-email" name="email" type="email" maxlength="191" autocomplete="email" dir="ltr"
                                             :value="old('email')" :invalid="$errors->has('email')" described class="text-end"/>
                        </x-ui.form.field>
                    </div>
                </div>

                {{-- Step 2 — مکان و تصاویر --}}
                <div id="join-step-2" data-step="2" role="group" aria-labelledby="join-step-2-title" class="flex scroll-mt-6 flex-col gap-5.5 border-t border-line pt-5.5">
                    <div class="flex flex-col gap-1.5">
                        <span class="text-sm font-extrabold text-primary">{{ __($t.'step_of', ['n' => fa_digits(2), 'total' => fa_digits($total)]) }}</span>
                        <h2 id="join-step-2-title" class="{{ $stepHead }}">{{ $stepLabels[1] }}</h2>
                    </div>
                    <div>
                        <h3 class="{{ $subhead }}">{{ __($t.'location.title') }}</h3>
                        <p class="m-0 mt-1 text-sm-plus font-semibold text-muted">{{ __($t.'location.hint') }}</p>
                    </div>
                    <div class="grid grid-cols-[2fr_1fr] gap-3.5 max-sm:grid-cols-1">
                        <x-ui.form.field for="join-address" :label="__($t.'location.address')" :error="$errors->first('address')" required>
                            <x-ui.form.input id="join-address" name="address" required minlength="10" maxlength="500" autocomplete="street-address"
                                             :placeholder="__($t.'location.address_placeholder')" :value="old('address')" :invalid="$errors->has('address')" described/>
                        </x-ui.form.field>
                        <x-ui.form.field for="join-phone" :label="__($t.'location.phone')" :error="$errors->first('phone')">
                            <x-ui.form.input id="join-phone" name="phone" type="tel" maxlength="20" inputmode="tel" dir="ltr" placeholder="021 …"
                                             :value="old('phone')" :invalid="$errors->has('phone')" described class="text-end"/>
                        </x-ui.form.field>
                    </div>
                    <div class="grid grid-cols-2 gap-3.5 max-sm:grid-cols-1">
                        <x-ui.form.field for="join-latitude" :label="__($t.'location.latitude')" :error="$errors->first('latitude')">
                            <x-ui.form.input id="join-latitude" name="latitude" inputmode="decimal" maxlength="20" dir="ltr" placeholder="35.7"
                                             :value="old('latitude')" :invalid="$errors->has('latitude')" described class="text-end"/>
                        </x-ui.form.field>
                        <x-ui.form.field for="join-longitude" :label="__($t.'location.longitude')" :error="$errors->first('longitude')">
                            <x-ui.form.input id="join-longitude" name="longitude" inputmode="decimal" maxlength="20" dir="ltr" placeholder="51.4"
                                             :value="old('longitude')" :invalid="$errors->has('longitude')" described class="text-end"/>
                        </x-ui.form.field>
                    </div>

                    <div>
                        <h3 class="{{ $subhead }}">{{ __($t.'photos.title') }}</h3>
                        <p class="m-0 mt-1 text-sm-plus font-semibold text-muted">{{ __($t.'photos.hint', ['recommended' => fa_digits($photoLimits['recommended'])]) }}</p>
                    </div>
                    <div class="flex flex-col gap-2">
                        <label for="join-photos" @class([
                            'flex h-30 cursor-pointer flex-col items-center justify-center gap-1.5 rounded-2xl border-[1.5px] border-dashed bg-lavender text-sm font-extrabold text-ink has-focus-visible:outline-2 has-focus-visible:outline-primary',
                            $errors->has('photos') || $errors->has('photos.*') ? 'border-danger' : 'border-primary',
                        ])>
                            <x-icon name="upload" class="size-6 text-primary"/>{{ __($t.'photos.pick') }}
                            <span class="text-xs font-semibold text-muted">{{ __($t.'photos.limits', ['max' => fa_digits($photoLimits['max']), 'mb' => $mb($photoLimits['bytes'])]) }}</span>
                            <input id="join-photos" type="file" name="photos[]" multiple accept="{{ implode(',', $photoLimits['mimes']) }}" class="sr-only"
                                   aria-describedby="join-photos-status join-photos-error"
                                   data-photos data-max="{{ $photoLimits['max'] }}" data-bytes="{{ $photoLimits['bytes'] }}" data-total="{{ $photoLimits['total'] }}"
                                   data-types="{{ implode(',', $photoLimits['mimes']) }}"
                                   data-msg-selected="{{ __($t.'photos.selected') }}" data-msg-too-many="{{ __($t.'photos.too_many', ['max' => fa_digits($photoLimits['max'])]) }}"
                                   data-msg-too-big="{{ __($t.'photos.too_big', ['mb' => $mb($photoLimits['bytes'])]) }}" data-msg-total="{{ __($t.'photos.total_big') }}"
                                   data-msg-type="{{ __($t.'photos.bad_type') }}">
                        </label>
                        <p id="join-photos-status" data-photos-status aria-live="polite" class="m-0 text-sm font-semibold text-muted empty:hidden"></p>
                        @if ($errors->has('photos') || $errors->has('photos.*'))
                            <span id="join-photos-error" role="alert" class="text-sm font-bold text-danger">{{ $errors->first('photos') ?: $errors->first('photos.*') }}</span>
                        @endif
                    </div>

                    <x-ui.form.field for="join-about" :label="__($t.'about')" :error="$errors->first('about')">
                        <x-ui.form.textarea id="join-about" name="about" rows="3" maxlength="{{ \App\Domain\Directory\Join\Support\JoinForm::ABOUT_MAX }}" class="min-h-30"
                                            :placeholder="__($t.'about_placeholder', ['max' => fa_digits(\App\Domain\Directory\Join\Support\JoinForm::ABOUT_MAX)])"
                                            :invalid="$errors->has('about')" described>{{ old('about') }}</x-ui.form.textarea>
                    </x-ui.form.field>

                    <fieldset class="m-0 flex min-w-0 flex-col gap-5.5 border-0 p-0">
                        <legend class="contents"><h3 class="{{ $subhead }}">{{ __($t.'ages') }}</h3></legend>
                        <div class="flex flex-wrap gap-2">
                            @foreach ($ageGroups as $value => $label)
                                <label class="{{ $chip }}"><input type="checkbox" name="ages[]" value="{{ $value }}" @checked(in_array((string) $value, $selectedAges, true)) class="sr-only">{{ $label }}</label>
                            @endforeach
                        </div>
                        @if ($errors->has('ages') || $errors->has('ages.*'))<span role="alert" class="text-sm font-bold text-danger">{{ $errors->first('ages') ?: $errors->first('ages.*') }}</span>@endif
                    </fieldset>

                    @if ($amenities !== [])
                        <fieldset class="m-0 flex min-w-0 flex-col gap-5.5 border-0 p-0">
                            <legend class="contents"><h3 class="{{ $subhead }}">{{ __($t.'amenities') }}</h3></legend>
                            <div class="flex flex-wrap gap-2">
                                @foreach ($amenities as $amenity)
                                    <label class="{{ $chip }}"><input type="checkbox" name="amenities[]" value="{{ $amenity['id'] }}" @checked(in_array((string) $amenity['id'], $selectedAmenities, true)) class="sr-only">{{ $amenity['name'] }}</label>
                                @endforeach
                            </div>
                            @if ($errors->has('amenities') || $errors->has('amenities.*'))<span role="alert" class="text-sm font-bold text-danger">{{ $errors->first('amenities') ?: $errors->first('amenities.*') }}</span>@endif
                        </fieldset>
                    @endif

                    <fieldset class="m-0 flex min-w-0 flex-col gap-5.5 border-0 p-0">
                        <legend class="contents"><h3 class="{{ $subhead }}">{{ __($t.'hours.title') }}</h3></legend>
                        <div>
                            @foreach ($weekdays as $day)
                                @php
                                    $key = $day['key'];
                                    $row = $oldHours === null
                                        ? $defaultHours[$key]
                                        : ['open' => ! empty($oldHours[$key]['open']), 'opens' => (string) ($oldHours[$key]['opens'] ?? ''), 'closes' => (string) ($oldHours[$key]['closes'] ?? '')];
                                @endphp
                                <div @class(['group flex items-center gap-3.5 py-2.5 max-sm:flex-wrap', 'border-t border-line' => ! $loop->first])>
                                    <b class="w-22.5 text-[14.5px]">{{ $day['label'] }}</b>
                                    <label class="relative flex h-8 w-13 shrink-0 cursor-pointer rounded-full bg-line transition-colors has-checked:bg-primary has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary">
                                        <input type="checkbox" role="switch" name="hours[{{ $key }}][open]" value="1" @checked($row['open']) class="peer sr-only">
                                        <span class="sr-only">{{ __($t.'hours.open') }}: {{ $day['label'] }}</span>
                                        <span aria-hidden="true" class="absolute start-1 top-1 size-6 rounded-full bg-surface peer-checked:start-6"></span>
                                    </label>
                                    <span class="hidden grow items-center gap-2.5 text-[14.5px] font-bold text-ink group-has-checked:flex">
                                        <input type="time" name="hours[{{ $key }}][opens]" value="{{ $row['opens'] }}" aria-label="{{ $day['label'] }} — {{ __($t.'hours.opens') }}"
                                               class="h-10.5 rounded-lg border-[1.5px] border-line bg-surface px-3.5 text-[14.5px] font-bold text-ink focus:border-primary">
                                        {{ __($t.'hours.to') }}
                                        <input type="time" name="hours[{{ $key }}][closes]" value="{{ $row['closes'] }}" aria-label="{{ $day['label'] }} — {{ __($t.'hours.closes') }}"
                                               class="h-10.5 rounded-lg border-[1.5px] border-line bg-surface px-3.5 text-[14.5px] font-bold text-ink focus:border-primary">
                                    </span>
                                    <span class="grow text-[14.5px] font-bold text-muted group-has-checked:hidden">{{ __($t.'hours.closed') }}</span>
                                </div>
                            @endforeach
                        </div>
                        @if ($errors->has('hours') || $errors->has('hours.*'))<span role="alert" class="text-sm font-bold text-danger">{{ __($t.'validation.hours') }}</span>@endif
                    </fieldset>
                </div>

                {{-- Step 3 — خدمات و قیمت --}}
                <div id="join-step-3" data-step="3" role="group" aria-labelledby="join-step-3-title" class="flex scroll-mt-6 flex-col gap-5.5 border-t border-line pt-5.5">
                    <div class="flex flex-col gap-1.5">
                        <span class="text-sm font-extrabold text-primary">{{ __($t.'step_of', ['n' => fa_digits(3), 'total' => fa_digits($total)]) }}</span>
                        <h2 id="join-step-3-title" class="{{ $stepHead }}">{{ $stepLabels[2] }}</h2>
                        <p class="m-0 text-md leading-relaxed font-medium text-muted">{{ __($t.'services.lead') }}</p>
                    </div>
                    <x-ui.form.field for="join-services" :label="__($t.'services.label')" :error="$errors->first('services')">
                        <x-ui.form.textarea id="join-services" name="services" rows="4" maxlength="{{ \App\Domain\Directory\Join\Support\JoinForm::SERVICES_MAX }}"
                                            :placeholder="__($t.'services.placeholder')" :invalid="$errors->has('services')" described>{{ old('services') }}</x-ui.form.textarea>
                    </x-ui.form.field>
                    <x-ui.form.field as="fieldset" :label="__($t.'services.booking')" :error="$errors->first('booking_mode')" required>
                        <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                            @foreach ($bookingModes as $mode)
                                <x-ui.form.radio-card name="booking_mode" :value="$mode['value']" :title="$mode['title']" :text="$mode['text']"
                                                      :icon="$mode['value'] === 'phone' ? 'phone' : 'calendar'" :checked="old('booking_mode', 'online') === $mode['value']"/>
                            @endforeach
                        </div>
                    </x-ui.form.field>
                </div>

                {{-- Step 4 — مدارک --}}
                <div id="join-step-4" data-step="4" role="group" aria-labelledby="join-step-4-title" class="flex scroll-mt-6 flex-col gap-5.5 border-t border-line pt-5.5">
                    <div class="flex flex-col gap-1.5">
                        <span class="text-sm font-extrabold text-primary">{{ __($t.'step_of', ['n' => fa_digits(4), 'total' => fa_digits($total)]) }}</span>
                        <h2 id="join-step-4-title" class="{{ $stepHead }}">{{ $stepLabels[3] }}</h2>
                        <p class="m-0 text-md leading-relaxed font-medium text-muted">{{ __($t.'documents.lead') }}</p>
                    </div>
                    <div class="flex flex-col gap-2">
                        <x-ui.form.checkbox name="license" value="1" required :checked="(bool) old('license')" aria-describedby="join-license-error">{{ __($t.'documents.license') }}</x-ui.form.checkbox>
                        @if ($errors->has('license'))<span id="join-license-error" role="alert" class="text-sm font-bold text-danger">{{ $errors->first('license') }}</span>@endif
                    </div>
                    <div class="flex flex-col gap-2">
                        <x-ui.form.checkbox name="terms" value="1" required :checked="(bool) old('terms')" aria-describedby="join-terms-error">{{ __($t.'documents.terms') }}<a href="{{ route('terms') }}" class="font-extrabold">{{ __($t.'documents.terms_link') }}</a>{{ __($t.'documents.terms_after') }}</x-ui.form.checkbox>
                        @if ($errors->has('terms'))<span id="join-terms-error" role="alert" class="text-sm font-bold text-danger">{{ $errors->first('terms') }}</span>@endif
                    </div>
                    <p class="m-0 flex items-start gap-2.5 text-base leading-relaxed font-semibold text-muted"><x-icon name="info" class="size-4.5 shrink-0 text-primary"/>{{ __($t.'documents.review') }}</p>

                    <div aria-hidden="true" class="sr-only">
                        <label for="join-website">{{ __($t.'honeypot') }}</label>
                        <input id="join-website" type="text" name="{{ \App\Http\Requests\JoinRequest::HONEYPOT }}" value="" tabindex="-1" autocomplete="off">
                    </div>
                    <input type="hidden" name="{{ \App\Http\Requests\JoinRequest::TIMER }}" value="{{ $formToken }}">
                </div>

                <div class="flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4.5">
                    <x-ui.button :href="route('directory.business')" variant="outline" size="xl" data-stepper-back>{{ __($t.'previous') }}</x-ui.button>
                    <x-ui.button variant="outline" size="xl" data-stepper-prev hidden>{{ __($t.'previous') }}</x-ui.button>
                    <x-ui.button size="xl" icon="arrow-left" icon-class="size-4.5 text-white" class="box-content border-[1.5px] border-primary" data-stepper-next hidden>{{ __($t.'next') }}</x-ui.button>
                    <x-ui.button type="submit" size="xl" icon="arrow-left" icon-class="size-4.5 text-white" class="box-content border-[1.5px] border-primary" data-stepper-submit>{{ __($t.'submit') }}</x-ui.button>
                </div>
            </div>

            <aside aria-label="{{ __($t.'preview.label') }}" class="flex w-90 shrink-0 flex-col gap-3 max-sm:w-full">
                <span aria-hidden="true" class="flex items-center gap-1.5 text-sm-plus font-extrabold text-muted"><x-icon name="eye-off" class="size-4 text-muted"/>{{ __($t.'preview.label') }}</span>
                <div inert data-preview>
                    <x-cards.place href="{{ route('directory.index') }}" as="b" :name="old('name') ?: __($t.'preview.name')" :category="__($t.'preview.category')"
                                   :district="__($t.'preview.district')" :slots="[__($t.'preview.after')]" illustration="place-cover-pool"/>
                </div>
                <p class="m-0 rounded-3xl border border-line bg-surface p-4 text-sm-plus leading-relaxed font-semibold text-muted">{{ __($t.'preview.note') }}</p>
            </aside>
        </form>
    </section>
@endsection
