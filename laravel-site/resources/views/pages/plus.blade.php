{{--
    Free vs Plus pricing page, design/Web_Pricing.html (+ _M). Data from App\Http\Controllers\PlusController:
      $plans       list<{key, title, price, unit, note, featured}> — the four plan cards (price = «قیمت در اپ» when
                   the lang switch `plus.pricing.show` is off, then unit is empty)
      $showPrices  bool, $chooseUrl — target of the «انتخاب» buttons (web app or #download)
      $faq         ?FaqGroupData (FaqServiceProvider composer, group `plus`; FAQPage JSON-LD already registered)
      $appLinks    AppLinksSettings — the #download band
--}}
@extends('layouts.app')

@section('content')
<div class="bleed-rows bg-surface">
    @php($gutter = 'px-30 max-xl:px-6 max-lg:px-5')
    <section aria-labelledby="plus-title" class="flex flex-col items-center gap-4 pt-22 pb-4 text-center {{ $gutter }} max-lg:pt-12">
        <span class="text-base font-extrabold text-primary">{{ __('plus.intro.eyebrow') }}</span>
        <h1 id="plus-title" class="m-0 font-display text-[56px] leading-[1.25] font-normal text-ink max-sm:text-[38px]">{{ __('plus.intro.title_line1') }}<br>{{ __('plus.intro.title_line2') }}</h1>
        <p class="m-0 mx-auto max-w-140 text-xl leading-loose font-medium text-muted">{{ __('plus.intro.lead') }}</p>
    </section>

    <section aria-label="{{ __('plus.cards.label') }}" class="pt-6 pb-24 {{ $gutter }} max-lg:pb-14">
        <div class="flex flex-col gap-6">
            <div class="flex justify-center">
                <span class="rounded-xl bg-sand px-3.5 py-1.5 text-center text-md font-extrabold text-ink">{{ __('plus.intro.trial') }}</span>
            </div>

            <ul class="m-0 mx-auto grid w-full max-w-300 list-none grid-cols-4 gap-5 p-0 pt-3 max-lg:grid-cols-2 max-sm:grid-cols-1 max-sm:gap-6">
                @foreach ($plans as $plan)
                    <li @class([
                        'relative flex flex-col gap-3.5 rounded-4xl px-6 pt-8 pb-6',
                        'bg-rose text-white shadow-rose' => $plan['featured'],
                        'border border-line bg-surface text-ink' => ! $plan['featured'],
                    ])>
                        @if ($plan['featured'])
                            <span class="absolute inset-x-0 -top-3 mx-auto w-fit rounded-lg border-[1.5px] border-rose bg-surface px-3 py-0.5 text-sm font-extrabold whitespace-nowrap text-rose">{{ __('plus.cards.recommended') }}</span>
                        @endif
                        <span @class(['text-sm font-extrabold', 'text-rose-pale' => $plan['featured'], 'text-muted' => ! $plan['featured']])>{{ __('plus.cards.brand') }}</span>
                        <h2 class="m-0 font-sans text-4xl font-bold">{{ $plan['title'] }}</h2>
                        <div class="flex flex-col gap-0.5">
                            <span class="font-display {{ $showPrices ? 'text-d-lg' : 'text-4xl' }} leading-[1.1]">{{ $plan['price'] }}</span>
                            @if ($plan['unit'] !== '')
                                <span @class(['text-sm font-bold', 'text-rose-pale' => $plan['featured'], 'text-muted' => ! $plan['featured']])>{{ $plan['unit'] }}</span>
                            @endif
                        </div>
                        @if ($plan['note'] !== '')
                            <span @class(['border-t pt-3 text-sm leading-relaxed font-semibold', 'border-white/30 text-rose-pale' => $plan['featured'], 'border-line text-muted' => ! $plan['featured']])>{{ $plan['note'] }}</span>
                        @endif
                        <x-site.button :href="$chooseUrl" :variant="$plan['featured'] ? 'white' : 'outline'" size="md" class="mt-auto">{{ __('plus.cards.choose') }}</x-site.button>
                    </li>
                @endforeach
            </ul>

            <p class="m-0 text-center text-md leading-loose font-semibold text-muted">{{ __('plus.cards.why_six') }}</p>

            <div class="mx-auto flex w-full max-w-300 flex-col gap-4 rounded-4xl bg-canvas px-9 py-8 max-sm:px-5 max-sm:py-6">
                <h2 class="m-0 font-sans text-3xl font-bold">{{ __('plus.includes.title') }}</h2>
                <ul class="m-0 grid list-none grid-cols-2 gap-x-8 gap-y-2 p-0 max-lg:grid-cols-1">
                    @foreach (__('plus.includes.items') as $item)
                        <li class="flex items-start gap-2.5 text-lg leading-loose font-semibold"><span aria-hidden="true" class="mt-1.5 shrink-0"><x-icon name="check" class="size-4.5 text-success"/></span>{{ $item }}</li>
                    @endforeach
                </ul>
            </div>

            <div class="mx-auto flex w-full max-w-300 items-center justify-between gap-4 rounded-4xl border border-line px-9 py-6 max-sm:flex-col max-sm:items-start max-sm:px-5">
                <div class="flex flex-col gap-1">
                    <h2 class="m-0 font-sans text-3xl font-bold">{{ __('plus.free.title') }}</h2>
                    <p class="m-0 text-md leading-loose font-medium text-muted">{{ __('plus.free.text') }}</p>
                </div>
                <x-site.button href="#download" variant="outline" size="md">{{ __('plus.free.cta') }}</x-site.button>
            </div>

            <p class="m-0 text-center text-sm font-semibold text-muted">{{ __('plus.free.draft_note') }}</p>
        </div>
    </section>

    <section aria-labelledby="plus-compare-title" class="flex flex-col gap-10 bg-canvas py-24 {{ $gutter }} max-lg:gap-6 max-lg:py-14">
        <x-site.heading id="plus-compare-title" :eyebrow="__('plus.compare.eyebrow')" :title="__('plus.compare.title')" align="center"/>
        <div class="mx-auto w-full max-w-220 overflow-x-auto">
            <table class="w-full border-collapse text-lg font-semibold">
                <caption class="sr-only">{{ __('plus.compare.caption') }}</caption>
                <thead>
                    <tr class="border-b-2 border-ink text-md font-extrabold">
                        <th scope="col" class="py-3.5 text-start font-extrabold">{{ __('plus.compare.feature') }}</th>
                        <th scope="col" class="w-40 py-3.5 text-center font-extrabold max-sm:w-20">{{ __('plus.compare.free') }}</th>
                        <th scope="col" class="w-40 py-3.5 text-center font-extrabold max-sm:w-20">{{ __('plus.compare.plus') }}</th>
                    </tr>
                </thead>
                <tbody>
                    @foreach (__('plus.compare.rows') as [$label, $inFree, $inPlus])
                        <tr class="border-b border-line">
                            <th scope="row" class="py-3.5 text-start font-semibold">{{ $label }}</th>
                            @foreach ([$inFree, $inPlus] as $has)
                                <td class="py-3.5 text-center">
                                    @if ($has)
                                        <x-icon name="check" class="inline size-5 text-success" aria-hidden="true"/><span class="sr-only">{{ __('plus.compare.yes') }}</span>
                                    @else
                                        <span aria-hidden="true" class="inline-block h-0.5 w-3 rounded-xs bg-line align-middle"></span><span class="sr-only">{{ __('plus.compare.no') }}</span>
                                    @endif
                                </td>
                            @endforeach
                        </tr>
                    @endforeach
                </tbody>
            </table>
        </div>
    </section>

    @if ($faq !== null && ! $faq->isEmpty())
        <x-site.faq :items="array_map(static fn ($item) => $item->toArray(), $faq->items)" :title="__('plus.faq.title')" id="plus-faq" class="pt-24 max-lg:pt-14"/>
    @endif

    <section aria-labelledby="plus-why-title" class="bg-surface pb-24 {{ $gutter }} max-lg:pb-14">
        <div class="mx-auto flex max-w-220 flex-col gap-3">
            <span class="text-base font-extrabold text-primary">{{ __('plus.why.eyebrow') }}</span>
            <h2 id="plus-why-title" class="m-0 font-display text-d-lg leading-[1.25] font-normal text-ink">{{ __('plus.why.title') }}</h2>
            <p class="m-0 max-w-180 text-xl leading-loose font-medium text-muted">{{ __('plus.why.text') }}</p>
        </div>
    </section>

    <x-site.download-band :title="__('plus.app_cta.title')" :lead="__('plus.app_cta.lead')" :links="$appLinks"/>
</div>
@endsection
