{{--
    <x-shop.cart-line :item="$item" :notice="$itemNotices[$item->key] ?? null" :media="$media" :index="$loop->index" :first="$loop->first"/>
    One cart row (L6-04, design shop-cart.html): cover photo (MediaData by id) or the product illustration on a stage
    tint, title + variant label + «حذف», the − / + stepper and the line total. Works without JS: the stepper buttons
    are submit buttons of a POST form carrying the new quantity (`name="quantity" value="n±1"`), «حذف» posts to the
    remove route. The lazy `cart` module intercepts these forms (data-cart-form). A sold-out line shows «ناموجود»
    instead of the stepper and its price is struck (left out of the totals).
--}}
@props(['item', 'notice' => null, 'media' => [], 'index' => 0, 'first' => false])
@php
    /** @var \App\Domain\Shop\Cart\Data\CartItemData $item */
    $tints = ['bg-stage-pregnancy/15', 'bg-primary/13', 'bg-stage-pregnancy/13', 'bg-stage-postpartum/13'];
    $cover = $item->coverMediaId === null ? null : ($media[$item->coverMediaId] ?? $item->coverMediaId);
    $href = route('shop.product', [$item->slug]);
@endphp
<li {{ $attributes->class(['flex items-center gap-4.5 py-4.5 max-lg:flex-wrap', 'border-t border-line' => ! $first]) }}>
    <a href="{{ $href }}" tabindex="-1" aria-hidden="true" class="relative size-26 shrink-0 overflow-hidden rounded-3xl {{ $tints[$index % count($tints)] }}">
        @if ($cover)
            <x-picture :media="$cover" :alt="$item->title" decorative sizes="104px" class="size-full object-cover"/>
        @elseif ($item->illustration)
            <x-illustration :name="$item->illustration" class="size-full"/>
        @endif
    </a>
    <div class="flex min-w-0 grow flex-col gap-1.5">
        <a href="{{ $href }}" class="text-lg font-bold text-ink hover:text-primary">{{ $item->title }}</a>
        @if ($item->variantLabel || $item->isDemo)
            <span class="text-sm-plus font-semibold text-muted">{{ implode(' · ', array_filter([$item->variantLabel, $item->isDemo ? __('shop.cart.demo') : null])) }}</span>
        @endif
        @if ($notice)
            <span class="flex items-center gap-1.5 text-sm font-bold text-danger"><x-icon name="alert-triangle" class="size-4 shrink-0"/>{{ $notice }}</span>
        @endif
        <form method="post" action="{{ route('shop.cart.remove', [$item->key]) }}" data-cart-form>
            @csrf
            <button type="submit" aria-label="{{ __('shop.cart.remove_label', ['name' => $item->title]) }}" class="flex cursor-pointer items-center gap-1 border-0 bg-transparent p-0 text-sm font-bold text-muted hover:text-danger">
                <x-icon name="trash" class="size-[15px]"/>{{ __('shop.cart.remove') }}
            </button>
        </form>
    </div>
    @if ($item->available)
        <form method="post" action="{{ route('shop.cart.update', [$item->key]) }}" data-cart-form class="flex items-center gap-2.5" aria-label="{{ __('shop.cart.quantity', ['name' => $item->title]) }}">
            @csrf
            <button type="submit" name="quantity" value="{{ $item->quantity + 1 }}" aria-label="{{ __('shop.cart.more') }}" @disabled($item->quantity >= $item->maxQuantity)
                    class="flex size-8.5 cursor-pointer items-center justify-center rounded-full border-[1.5px] border-line bg-transparent text-ink hover:border-primary disabled:cursor-not-allowed disabled:opacity-40">
                <x-icon name="plus" class="size-[15px]"/>
            </button>
            <b class="min-w-3.5 text-center text-base" aria-label="{{ __('shop.cart.quantity_value', ['count' => fa_digits($item->quantity)]) }}">{{ fa_digits($item->quantity) }}</b>
            <button type="submit" name="quantity" value="{{ $item->quantity - 1 }}" aria-label="{{ __('shop.cart.less') }}" @disabled($item->quantity <= 1)
                    class="flex size-8.5 cursor-pointer items-center justify-center rounded-full border-[1.5px] border-line bg-transparent text-ink hover:border-primary disabled:cursor-not-allowed disabled:opacity-40">
                <x-icon name="minus" class="size-[15px]"/>
            </button>
        </form>
        <div class="w-37.5 text-end max-lg:w-auto">
            <x-ui.price :amount="$item->lineTotal()->toToman()"/>
            @if ($item->quantity > 1)
                <div class="text-xs font-semibold text-muted">{{ __('shop.cart.unit_price', ['amount' => $item->unitPrice->formatShort()]) }}</div>
            @endif
        </div>
    @else
        <x-ui.badge tone="danger">{{ __('shop.cart.unavailable') }}</x-ui.badge>
        <div class="w-37.5 text-end text-muted line-through max-lg:w-auto">
            <x-ui.price :amount="$item->lineTotal()->toToman()"/>
        </div>
    @endif
</li>
