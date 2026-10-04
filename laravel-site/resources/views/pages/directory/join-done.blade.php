{{--
    Join done page (L5-05), design/html/directory-join-done.html — noindex, never cached. Controller:
    App\Http\Controllers\Directory\JoinController@done.
      $code         ?string  tracking code flashed by the join post (null on a direct visit)
      $placeName    ?string  the submitted place name
      $submittedAt  ?Carbon  submit time (Tehran)
    The review time is still the design placeholder «[زمان بررسی]» (lang) until it becomes a directory setting.
    Copy: lang/fa/directory.php `done`.
--}}
@extends('layouts.app')

@php
    $t = 'directory.done.';
    $lead = $placeName !== null ? __($t.'lead_named', ['name' => $placeName]) : __($t.'lead');
    if ($code !== null) {
        $lead .= ' '.__($t.'code', ['code' => fa_digits($code)]);
    }
@endphp

@section('content')
    <section class="flex flex-col items-center gap-5.5 px-30 pt-20 pb-27.5 text-center max-lg:px-5 max-lg:pt-11 max-lg:pb-[60.5px]">
        <x-ui.success-hero icon="paper-plane" tone="lavender" :title="__($t.'title')">{{ $lead }}</x-ui.success-hero>

        <div class="box-border w-155 max-w-full rounded-6xl border border-line bg-surface px-8 pt-7 pb-3 text-start max-sm:w-full">
            <x-ui.timeline :label="__($t.'status')" :items="[
                ['title' => __($t.'timeline.received'), 'time' => $submittedAt !== null ? __($t.'today', ['time' => jdate($submittedAt, 'H:i')]) : null, 'state' => 'done'],
                ['title' => __($t.'timeline.review'), 'time' => __($t.'timeline.review_time'), 'state' => 'current'],
                ['title' => __($t.'timeline.call'), 'time' => __($t.'timeline.call_text'), 'state' => 'todo'],
                ['title' => __($t.'timeline.publish'), 'time' => __($t.'timeline.publish_text'), 'state' => 'todo'],
            ]"/>
        </div>

        <div class="flex gap-3 max-lg:flex-wrap max-lg:justify-center">
            <x-ui.button :href="route('directory.index')" size="xl" icon="store" icon-class="size-4.5 text-white" class="box-content border-[1.5px] border-primary">{{ __($t.'directory') }}</x-ui.button>
            <x-ui.button :href="route('directory.business')" variant="outline" size="xl" icon="info" icon-class="size-4.5 text-ink">{{ __($t.'business') }}</x-ui.button>
        </div>
        <span class="text-base font-semibold text-muted">{{ __($t.'question') }} <a href="{{ route('contact') }}#contact-form" class="font-extrabold">{{ __($t.'support') }}</a></span>
    </section>
@endsection
