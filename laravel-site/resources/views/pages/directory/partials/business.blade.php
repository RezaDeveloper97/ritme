{{-- «مجموعه‌ای برای مادر و کودک داری؟» band → the business page (design section 5). --}}
<section aria-labelledby="business-title" class="mx-30 mt-0 mb-20 flex items-center gap-10 rounded-7xl bg-lavender px-14 py-12 max-lg:mx-5 max-lg:flex-wrap max-sm:rounded-4xl max-sm:px-6 max-sm:py-8">
    <span class="flex size-24 shrink-0 items-center justify-center rounded-full bg-surface">
        <x-icon name="store" class="size-11 text-primary"/>
    </span>
    <div class="flex flex-1 flex-col gap-2 max-lg:basis-75 max-sm:basis-full">
        <h2 id="business-title" class="m-0 font-display text-[34px] leading-heading font-normal text-ink">{{ __('directory.business.title') }}</h2>
        <p class="m-0 text-lg leading-loose font-medium text-muted">{{ __('directory.business.text') }}</p>
    </div>
    <a href="{{ route('directory.business') }}" class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-primary bg-primary px-6.5 text-[15.5px] font-extrabold text-white hover:bg-primary-hover hover:text-white">
        <x-icon name="arrow-left" class="size-4.5 text-white"/>{{ __('directory.business.cta') }}
    </a>
</section>
