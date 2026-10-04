{{--
    <x-ui.promise-banner href="{{ route('social-responsibility') }}"/>
    «آگاهی، حق همه زنان ایران است» (AUDIT §2.4, index + about): blush → mist gradient, radius 40, 120px heart medallion,
    stage-red eyebrow, h2 38, lead with the «همیشه رایگان می‌مانند» emphasis, ink CTA. Copy defaults are the design's;
    override with props (`lead` accepts an HtmlString for the bold phrase).
--}}
@props([
    'href',
    'eyebrow' => 'مسئولیت اجتماعی ریتمی',
    'title' => 'آگاهی، حق همه زنان ایران است',
    'lead' => null,
    'cta' => 'تعهد ما را بخوان',
])
@php
    $lead ??= new \Illuminate\Support\HtmlString('سلامت زنان و تربیت کودکانشان نباید به توان پرداخت گره بخورد. برای همین، بخش‌های اصلی ریتمی <b class="text-stage-cycle">همیشه رایگان می‌مانند</b> و ریتمی برای خدمات محوری‌اش هزینه‌ای نمی‌گیرد.');
@endphp
<section aria-label="تعهد رایگان ماندن" {{ $attributes->class('mx-30 mt-0 mb-24 flex items-center gap-14 rounded-7xl border border-line bg-linear-135/srgb from-blush to-mist px-16 py-14 max-lg:flex-wrap max-sm:mx-5 max-sm:mb-12 max-sm:rounded-4xl max-sm:px-5.5 max-sm:py-7') }}>
    <x-ui.icon-tile icon="heart" color="surface" size="lg"/>
    <div class="flex flex-1 flex-col gap-3 max-lg:basis-75 max-sm:basis-full">
        <x-ui.eyebrow color="cycle">{{ $eyebrow }}</x-ui.eyebrow>
        <h2 class="m-0 font-display text-[38px] leading-heading font-normal text-ink">{{ $title }}</h2>
        <p class="m-0 max-w-190 text-xl leading-loose font-semibold text-ink max-sm:max-w-full">{{ $lead }}</p>
    </div>
    <a href="{{ $href }}" class="flex h-13.5 items-center gap-2 rounded-full bg-ink px-6 text-md font-extrabold whitespace-nowrap text-white transition-colors hover:bg-primary-hover hover:text-white">{{ $cta }}<x-icon name="arrow-left" class="size-4"/></a>
</section>
