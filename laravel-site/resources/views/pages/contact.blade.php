{{--
    Contact page (L3-10), design/html/contact.html. Data from App\Http\Controllers\ContactController:
      $contact          ContactSettings — channel texts (support/partnership email, phone, hours, response time, address)
      $supportEmail, $partnershipEmail  ?string — the same emails when they are real addresses (→ mailto links)
      $phoneHref        ?string — dialable phone number (→ tel: link), null for empty/placeholder settings
      $emergencyNumber  string  — general settings (default 115)
      $topics           array<string, string> — ContactTopic value => label (first is pre-selected)
      $formToken        string  — FormTimer time-trap token
      $faq              ?FaqGroupData — the `contact` FAQ group (FaqServiceProvider composer; also its FAQPage JSON-LD)
    The form is a plain POST (no JS): errors and the «sent» message come back through the session after a redirect to
    #contact-form. Topic chips are real radios styled as the design's chips (CSS `has-checked:`).
    Copy: lang/fa/contact.php.
--}}
@extends('layouts.app')

@php
    $topicKeys = array_keys($topics);
    $selectedTopic = old('topic', $topicKeys[0] ?? null);
    $sent = session(\App\Http\Controllers\ContactController::FLASH) === 'sent';
    $channels = array_values(array_filter([
        [
            'icon' => 'message', 'tile' => 'bg-primary/10', 'color' => 'text-primary',
            'title' => __('contact.channels.app.title'),
            'value' => __('contact.channels.app.path'),
            'note' => $contact->responseTime !== null ? __('contact.channels.app.response', ['time' => $contact->responseTime]) : null,
        ],
        $contact->supportEmail !== null ? [
            'icon' => 'mail', 'tile' => 'bg-stage-postpartum/10', 'color' => 'text-stage-postpartum',
            'title' => __('contact.channels.email.title'),
            'value' => $contact->supportEmail, 'href' => $supportEmail !== null ? 'mailto:'.$supportEmail : null,
            'note' => $contact->partnershipEmail, 'notePrefix' => __('contact.channels.email.partnership'),
            'noteHref' => $partnershipEmail !== null ? 'mailto:'.$partnershipEmail : null,
        ] : null,
        $contact->phone !== null ? [
            'icon' => 'phone', 'tile' => 'bg-primary/10', 'color' => 'text-primary',
            'title' => __('contact.channels.phone.title'),
            'value' => $contact->phone, 'href' => $phoneHref !== null ? 'tel:'.$phoneHref : null,
            'note' => $contact->workingHours,
        ] : null,
        $contact->address !== null ? [
            'icon' => 'map-pin', 'tile' => 'bg-stage-cycle/10', 'color' => 'text-stage-cycle',
            'title' => __('contact.channels.address.title'),
            'value' => $contact->address,
            'note' => $contact->addressNote,
        ] : null,
    ]));
@endphp

@section('content')
    <x-ui.page-intro :eyebrow="__('contact.intro.eyebrow')" :title="__('contact.intro.title')" :lead="__('contact.intro.lead')"/>

    <section aria-label="{{ __('contact.channels.label') }}" class="flex flex-col gap-10 px-30 pt-6 pb-24 max-lg:px-5 max-lg:pb-[52.8px]">
        <div class="flex items-start gap-12 max-lg:flex-wrap">
            <div class="flex-1 max-lg:basis-75 max-sm:basis-full">
                <ul class="m-0 list-none p-0">
                    @foreach ($channels as $channel)
                        <li class="flex items-start gap-4 border-b border-line py-5.5 max-lg:flex-wrap">
                            <span aria-hidden="true" class="flex size-13 shrink-0 items-center justify-center rounded-2xl {{ $channel['tile'] }}"><x-icon :name="$channel['icon']" class="size-6 {{ $channel['color'] }}"/></span>
                            <div>
                                <h2 class="m-0 font-sans text-xl leading-[inherit] font-bold">{{ $channel['title'] }}</h2>
                                <div class="mt-0.5 text-lg font-bold">
                                    @if (! empty($channel['href']))<a href="{{ $channel['href'] }}" dir="ltr" class="text-ink hover:text-primary">{{ $channel['value'] }}</a>@else{{ $channel['value'] }}@endif
                                </div>
                                @if ($channel['note'] !== null)
                                    <div class="mt-1 text-base font-semibold text-muted">{{ $channel['notePrefix'] ?? '' }}@if (! empty($channel['noteHref']))<a href="{{ $channel['noteHref'] }}" dir="ltr" class="text-muted hover:text-primary">{{ $channel['note'] }}</a>@else{{ $channel['note'] }}@endif</div>
                                @endif
                            </div>
                        </li>
                    @endforeach
                </ul>
                <x-ui.alert-emergency class="mt-6" :number="$emergencyNumber">{{ __('contact.emergency', ['number' => fa_digits($emergencyNumber)]) }}</x-ui.alert-emergency>
            </div>

            <form id="contact-form" method="post" action="{{ route('contact.store') }}" aria-labelledby="contact-form-title" novalidate
                  class="box-content flex w-140 shrink-0 scroll-mt-6 flex-col gap-4.5 rounded-6xl border border-line bg-surface p-8 max-sm:box-border max-sm:w-full max-sm:max-w-full">
                @csrf
                <h2 id="contact-form-title" class="m-0 font-display text-d-sm leading-heading font-normal text-ink">{{ __('contact.form.title') }}</h2>

                @if ($sent)
                    <div role="status" class="flex items-start gap-3 rounded-[22px] border border-success/30 bg-success-soft p-4.5">
                        <x-icon name="check" class="mt-1 size-5 shrink-0 text-success"/>
                        <p class="m-0 text-md leading-relaxed font-semibold text-ink"><b class="block text-base">{{ __('contact.form.sent_title') }}</b>{{ __('contact.form.sent') }}</p>
                    </div>
                @elseif ($errors->any())
                    <div role="alert" class="rounded-[22px] border border-danger-line bg-danger-soft p-4.5 text-md leading-relaxed font-semibold text-ink">
                        <b class="block text-base">{{ __('contact.form.errors_title') }}</b>
                        @if ($errors->has('form_token'))<span class="block">{{ $errors->first('form_token') }}</span>@endif
                    </div>
                @endif

                <x-ui.form.field as="fieldset" for="contact-topic" :label="__('contact.form.topic')" :error="$errors->first('topic')" class="gap-0">
                    <div class="flex flex-wrap gap-2">
                        @foreach ($topics as $value => $label)
                            <label class="flex h-10 cursor-pointer items-center rounded-full border-[1.5px] border-line bg-surface px-3.5 text-sm-plus font-bold transition-colors hover:border-primary has-checked:border-primary has-checked:bg-lavender has-focus-visible:outline-2 has-focus-visible:outline-primary">
                                <input type="radio" name="topic" value="{{ $value }}" @checked($selectedTopic === $value) class="sr-only">{{ $label }}
                            </label>
                        @endforeach
                    </div>
                </x-ui.form.field>

                <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                    <x-ui.form.field for="contact-name" :label="__('contact.form.name')" :error="$errors->first('name')">
                        <x-ui.form.input id="contact-name" name="name" autocomplete="name" maxlength="{{ \App\Http\Requests\ContactRequest::NAME_MAX }}" required
                                         :placeholder="__('contact.form.name_placeholder')" :value="old('name')" :invalid="$errors->has('name')" described/>
                    </x-ui.form.field>
                    <x-ui.form.field for="contact-reply" :label="__('contact.form.contact')" :error="$errors->first('contact')">
                        <x-ui.form.input id="contact-reply" name="contact" autocomplete="email" maxlength="191" required dir="auto"
                                         :placeholder="__('contact.form.contact_placeholder')" :value="old('contact')" :invalid="$errors->has('contact')" described/>
                    </x-ui.form.field>
                </div>

                <x-ui.form.field for="contact-message" :label="__('contact.form.message')" :error="$errors->first('message')">
                    <x-ui.form.textarea id="contact-message" name="message" rows="3" required maxlength="{{ \App\Http\Requests\ContactRequest::MESSAGE_MAX }}"
                                        :placeholder="__('contact.form.message_placeholder')" :invalid="$errors->has('message')" described>{{ old('message') }}</x-ui.form.textarea>
                </x-ui.form.field>

                <p id="contact-note" class="m-0 text-sm leading-relaxed font-semibold text-muted">{{ __('contact.form.note') }}</p>

                <div aria-hidden="true" class="sr-only">
                    <label for="contact-website">{{ __('contact.form.honeypot') }}</label>
                    <input id="contact-website" type="text" name="{{ \App\Http\Requests\ContactRequest::HONEYPOT }}" value="" tabindex="-1" autocomplete="off">
                </div>
                <input type="hidden" name="{{ \App\Http\Requests\ContactRequest::TIMER }}" value="{{ $formToken }}">

                <x-ui.button type="submit" size="xl" icon="arrow-left" icon-class="size-4.5 text-white" class="box-content border-[1.5px] border-primary">{{ __('contact.form.submit') }}</x-ui.button>
            </form>
        </div>
    </section>

    <x-faq :faq="$faq" :eyebrow="__('contact.faq.eyebrow')" :title="__('contact.faq.title')" bg="surface">
        <a href="{{ route('faq') }}" class="text-md font-extrabold">{{ __('contact.faq.all') }}</a>
    </x-faq>
@endsection
