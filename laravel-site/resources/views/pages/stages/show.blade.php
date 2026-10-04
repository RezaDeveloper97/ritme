{{--
    Life-stage page template (L3-03): cycle, ttc, pregnancy, postpartum, menopause, teen. $page is a
    StagePageData DTO (App\Domain\Content\Stages\StagePageBuilder) — no lookups here. Sections (AUDIT §2.4):
    stage pills + hero (dark block) → feature splits (first = #how) → tools → help (optional) → emergency note →
    readings → FAQ → #download app CTA.
--}}
@extends('layouts.app')

@section('hero')
    @include('pages.stages.partials.stage-nav', ['items' => $page->nav, 'label' => $page->navLabel])
    @include('pages.stages.partials.hero', ['hero' => $page->hero])
@endsection

@section('content')
    @foreach ($page->features as $feature)
        @include($feature->partial, ['feature' => $feature])
    @endforeach

    <x-stage.tools-block :items="$page->toolProps()" :eyebrow="$page->copy['toolsEyebrow']" :title="$page->copy['toolsTitle']"/>

    @if ($page->help !== [])
        <x-stage.help-block :items="$page->helpProps()" :eyebrow="$page->copy['helpEyebrow']" :title="$page->copy['helpTitle']"/>
    @endif

    <x-ui.section as="div" pad="none" class="pt-16 max-lg:pt-[35.2px]">
        <x-ui.alert-emergency :number="$page->emergencyNumber">{{ $page->emergencyText }}</x-ui.alert-emergency>
    </x-ui.section>

    <x-stage.readings :posts="$page->readings" :more-href="$page->readingsMoreUrl" :more-label="$page->copy['readingsMore']" :eyebrow="$page->copy['readingsEyebrow']" :title="$page->copy['readingsTitle']"/>

    @include('pages.stages.partials.faq', ['eyebrow' => $page->copy['faqEyebrow'], 'title' => $page->faqTitle, 'items' => $page->faqProps(), 'group' => $page->faqGroup])

    <x-ui.app-cta :title="$page->appCtaTitle" :lead="$page->appCtaLead" :links="$page->appLinks" :qr-url="$page->qrUrl"/>
@endsection
