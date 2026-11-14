<section aria-labelledby="home-steps-title" class="flex flex-col gap-10 bg-surface px-30 py-24 max-xl:px-6 max-lg:gap-6 max-lg:px-5 max-lg:py-14">
    <x-site.heading id="home-steps-title" :eyebrow="__('home.steps.eyebrow')" :title="__('home.steps.title')"/>
    <div class="relative">
        <span aria-hidden="true" class="absolute start-7 top-7 end-62 h-0.5 bg-line max-lg:hidden"></span>
        <ol class="m-0 grid list-none grid-cols-4 gap-8 p-0 max-lg:flex max-lg:flex-col max-lg:gap-0">
            @foreach (__('home.steps.items') as $i => $step)
                <li class="relative flex flex-col gap-3.5 max-lg:flex-row max-lg:gap-4 max-lg:pb-6.5">
                    @unless ($loop->last)
                        <span aria-hidden="true" class="absolute start-5.25 top-11.5 bottom-1 w-0.5 bg-line lg:hidden"></span>
                    @endunless
                    <span @class(['relative z-1 flex size-14 shrink-0 items-center justify-center rounded-full border-2 border-rose font-display text-[28px] max-lg:size-11 max-lg:text-[22px]', 'bg-rose text-white' => $i === 0, 'bg-surface text-rose' => $i !== 0])>{{ fa_digits($i + 1) }}</span>
                    <div class="flex flex-col gap-1 max-lg:pt-1.5 lg:gap-3.5">
                        <h3 class="m-0 text-3xl font-bold max-lg:text-xl">{{ $step['title'] }}</h3>
                        <p class="m-0 max-w-60 text-md leading-[1.9] font-medium text-muted max-lg:max-w-none">{{ $step['text'] }}</p>
                    </div>
                </li>
            @endforeach
        </ol>
    </div>
    <div><x-site.button href="#download" icon="download">{{ __('home.steps.cta') }}</x-site.button></div>
</section>
