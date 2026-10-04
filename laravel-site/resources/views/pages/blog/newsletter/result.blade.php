{{--
    Newsletter confirm / unsubscribe result (noindex, never page-cached). $title (h1), $text, $tone, $icon.
--}}
@extends('layouts.app', ['appCta' => false])

@section('content')
    <x-ui.section pad="lg">
        <x-ui.success-hero :icon="$icon" :tone="$tone" :title="$title">
            {{ $text }}
            <x-slot:actions>
                <x-ui.button :href="route('blog.index')" size="lg" icon-end="arrow-left">{{ __('blog.back') }}</x-ui.button>
            </x-slot:actions>
        </x-ui.success-hero>
    </x-ui.section>
@endsection
