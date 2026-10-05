{{--
    Booked page (L5-04), design/html/directory-booked.html — the status page of a booking REQUEST, found only by its
    unguessable code; noindex, never cached. Controller: App\Http\Controllers\Directory\BookingController@show.
      $booking   BookingData  code, status, place/service snapshot, preferred day + window, child age, masked mobile
      $title     string       h1 by status («درخواست رزروت ثبت شد» …)
      $lead      string       what happens next (the place confirms the time by phone; nothing is paid here)
      $rows      list<array{icon, title, text, price?, unit?}>  summary rows under the service
      $place     ?array       url, phone, directions (geo:), cover (media id), illustration, area, rules, cancellation
                              — null when the place is no longer published
      $appLinks  AppLinksSettings
    Design differences on purpose: no "confirmed + paid" claims (request flow, no online payment), no calendar file
    or online cancel (the time is not fixed yet; changes go through the place), the child's name is not collected.
    Copy: lang/fa/directory.php `booked`.
--}}
@extends('layouts.app')

@php($t = 'directory.booked.')

@section('content')
    <section class="flex items-start justify-center gap-12 px-30 pt-16 pb-24 max-lg:flex-wrap max-lg:px-5 max-lg:pt-[35.2px] max-lg:pb-[52.8px]">
        <div class="flex w-155 flex-col gap-6 max-sm:box-border max-sm:w-full max-sm:max-w-full">
            <x-ui.success-hero :icon="$booking->status->value === 'cancelled' ? 'x' : 'check'" align="start" :title="$title">
                {{ $lead }} {{ __($t.'code') }}: <bdi dir="ltr" class="font-extrabold whitespace-nowrap text-ink">{{ $booking->code }}</bdi>
            </x-ui.success-hero>

            <div class="rounded-6xl border border-line bg-surface px-7 py-6">
                <div class="flex items-center gap-4 pb-4">
                    <div class="relative size-21 shrink-0 overflow-hidden rounded-3xl bg-lavender">
                        @if ($place !== null && $place['cover'] !== null)
                            <x-picture :media="$place['cover']" :alt="$booking->placeName" sizes="84px" class="size-full object-cover"/>
                        @else
                            <x-illustration :name="$place['illustration'] ?? 'place-cover-playhouse'" width="100%" height="100%" class="block size-full"/>
                        @endif
                    </div>
                    <div>
                        <b class="text-2xl">{{ $booking->serviceName ?? __($t.'service_none') }}</b>
                        <div class="text-base font-semibold text-muted">{{ implode(' · ', array_filter([$booking->placeName, $place['area'] ?? null])) }}</div>
                    </div>
                </div>
                @foreach ($rows as $row)
                    <div class="flex items-center gap-3.5 border-t border-line py-3.5">
                        <span aria-hidden="true" class="flex size-11 shrink-0 items-center justify-center rounded-full bg-primary/13"><x-icon :name="$row['icon']" class="size-5.5 text-primary"/></span>
                        <div>
                            @isset($row['price'])
                                <b class="text-[15.5px]"><x-ui.price :amount="$row['price']" size="inline"/>@if ($row['unit']) <span class="font-semibold text-muted">· {{ $row['unit'] }}</span>@endif</b>
                            @else
                                <b class="text-[15.5px]">{{ $row['title'] }}</b>
                            @endisset
                            <div class="text-sm-plus font-semibold text-muted">{{ $row['text'] }}</div>
                        </div>
                    </div>
                @endforeach
            </div>

            <div class="flex gap-3 max-lg:flex-wrap">
                @if ($place !== null)
                    <x-ui.button :href="$place['directions'] ?? $place['url'].'#address'" variant="outline" size="xl" icon="navigate" icon-class="size-4.5 text-ink" class="text-[15.5px]">{{ __($t.'directions') }}</x-ui.button>
                    @if ($place['phone'])
                        <x-ui.button :href="'tel:'.$place['phone']" variant="outline" size="xl" icon="phone" icon-class="size-4.5 text-ink" class="text-[15.5px]">{{ __($t.'call') }}</x-ui.button>
                    @endif
                    <x-ui.button :href="$place['url']" variant="outline" size="xl" icon="store" icon-class="size-4.5 text-ink" class="text-[15.5px]">{{ __($t.'place') }}</x-ui.button>
                @else
                    <x-ui.button :href="route('directory.index')" variant="outline" size="xl" icon="store" icon-class="size-4.5 text-ink" class="text-[15.5px]">{{ __($t.'directory') }}</x-ui.button>
                @endif
            </div>
            <span class="text-base font-semibold text-muted">{{ $place['cancellation'] ?? __($t.'change') }}</span>
        </div>

        <div class="flex w-110 flex-col gap-5 max-sm:w-full max-sm:max-w-full">
            <x-ui.app-cta variant="aside" icon="bell" :title="__($t.'app.title')" :lead="__($t.'app.lead')" :links="$appLinks"/>
            @if ($place !== null && $place['rules'] !== [])
                <section aria-labelledby="booked-rules" class="flex flex-col gap-2.5 rounded-5xl border border-line bg-surface p-6">
                    <h2 id="booked-rules" class="m-0 text-lg font-extrabold">{{ __($t.'rules') }}</h2>
                    <ul class="m-0 flex list-none flex-col gap-1 p-0">
                        @foreach ($place['rules'] as $rule)
                            <li class="flex items-start gap-2.5 text-lg leading-relaxed font-semibold text-ink"><x-icon name="check" class="mt-1.5 size-4.5 shrink-0 text-stage-postpartum"/>{{ $rule }}</li>
                        @endforeach
                    </ul>
                </section>
            @endif
        </div>
    </section>
@endsection
