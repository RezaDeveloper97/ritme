<section aria-labelledby="home-partner-title" class="bg-surface px-30 py-24 max-xl:px-6 max-lg:px-5 max-lg:py-14">
    <div class="flex items-center gap-18 max-lg:flex-col max-lg:items-stretch max-lg:gap-8">
        <div class="flex-1">
            <span class="relative block h-105 overflow-hidden rounded-4xl bg-canvas max-lg:h-72">
                @if ($photos[\App\Domain\Content\Enums\SitePhoto::PartnerTea->value] ?? null)
                    <x-picture :media="$photos[\App\Domain\Content\Enums\SitePhoto::PartnerTea->value]" :alt="\App\Domain\Content\Enums\SitePhoto::PartnerTea->alt()" sizes="(max-width: 1024px) 100vw, 50vw" class="size-full object-cover"/>
                @endif
            </span>
        </div>
        <div class="flex flex-1 flex-col gap-4">
            <x-site.heading id="home-partner-title" :eyebrow="__('home.partner.eyebrow')" :title="__('home.partner.title')" :lead="__('home.partner.text')" class="max-sm:[&_h2]:text-[28px]"/>
            <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
                @foreach (__('home.partner.points') as $point)
                    <li class="flex items-start gap-2.5 text-lg leading-[1.9] font-semibold max-sm:text-[15.5px]"><span aria-hidden="true" class="mt-2.25 size-2 shrink-0 rounded-full bg-rose"></span>{{ $point }}</li>
                @endforeach
            </ul>
        </div>
    </div>
</section>
