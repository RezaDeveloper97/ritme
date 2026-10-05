{{--
    Newsletter confirmation (L9-04b, F19): the mailed link shows this button; only the POST confirms, so mail scanners
    and link prefetchers that follow the link change nothing. $action = POST URL with the token (CSRF-protected; this
    page is never page-cached, so the token is always the visitor's session token).
--}}
@extends('layouts.app', ['appCta' => false])

@section('content')
    <x-ui.section pad="lg">
        <x-ui.success-hero icon="mail" tone="lavender" :title="__('blog.newsletter.confirm.title')">
            {{ __('blog.newsletter.confirm.text') }}
            <x-slot:actions>
                <form action="{{ $action }}" method="post">
                    @csrf
                    <button type="submit" class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-primary bg-primary px-6.5 text-[15.5px] font-extrabold text-white transition-colors hover:bg-primary-hover">{{ __('blog.newsletter.confirm.button') }}</button>
                </form>
                <x-ui.button :href="route('blog.index')" variant="outline" size="lg">{{ __('blog.back') }}</x-ui.button>
            </x-slot:actions>
        </x-ui.success-hero>
    </x-ui.section>
@endsection
