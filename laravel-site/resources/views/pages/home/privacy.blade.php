<section aria-labelledby="home-privacy-title" class="bg-surface px-30 py-24 max-xl:px-6 max-lg:px-5 max-lg:py-14">
    <div class="flex items-center gap-20 rounded-6xl bg-canvas px-16 py-14 max-lg:flex-col max-lg:items-stretch max-lg:gap-3 max-lg:px-8 max-sm:rounded-4xl max-sm:px-5 max-sm:pt-7 max-sm:pb-3">
        <div class="flex flex-1 flex-col gap-4.5">
            <span class="text-base font-extrabold text-primary">{{ __('home.privacy.eyebrow') }}</span>
            <h2 id="home-privacy-title" class="m-0 font-display text-[44px] leading-[1.3] font-normal text-ink max-sm:text-[30px]">{{ __('home.privacy.title_line_1') }}<br>{{ __('home.privacy.title_line_2') }}</h2>
            <div><x-site.text-link :href="route('privacy')" class="text-[15.5px]">{{ __('home.privacy.link') }}</x-site.text-link></div>
        </div>
        <ul class="m-0 flex flex-1 list-none flex-col p-0">
            @foreach (__('home.privacy.items') as $item)
                <li @class(['flex flex-col gap-1 py-5 max-sm:py-4', 'border-t border-line' => ! $loop->first])>
                    <h3 class="m-0 text-3xl font-bold max-sm:text-xl">{{ $item['title'] }}</h3>
                    <span class="text-lg font-medium text-muted max-sm:text-md">{{ $item['text'] }}</span>
                </li>
            @endforeach
        </ul>
    </div>
</section>
