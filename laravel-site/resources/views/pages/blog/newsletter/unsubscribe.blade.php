{{--
    Unsubscribe confirmation (GET shows this form; mail scanners that follow links do not unsubscribe anyone).
    $action = POST URL with the token (no CSRF needed: the token authorises, see routes/web.php).
--}}
@extends('layouts.app', ['appCta' => false])

@section('content')
    <x-ui.section pad="lg">
        <x-ui.success-hero icon="mail" tone="lavender" :title="__('blog.newsletter.unsubscribe.title')">
            {{ __('blog.newsletter.unsubscribe.text') }}
            <x-slot:actions>
                <form action="{{ $action }}" method="post">
                    <button type="submit" class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-primary bg-primary px-6.5 text-[15.5px] font-extrabold text-white transition-colors hover:bg-primary-hover">{{ __('blog.newsletter.unsubscribe.button') }}</button>
                </form>
                <x-ui.button :href="route('blog.index')" variant="outline" size="lg">{{ __('blog.back') }}</x-ui.button>
            </x-slot:actions>
        </x-ui.success-hero>
    </x-ui.section>
@endsection
