{{--
    <x-shop.cart-link class="…"/>
    Cart button with a count badge. The badge is NOT rendered from the server (pages are full-page cached for
    guests): the tiny `cart-badge` module reads the `ritme_cart_count` cookie (an integer the cart of L6-04 writes,
    readable by JS) and fills / shows it. Without the cookie or JS the badge stays hidden.
--}}
<a href="{{ route('shop.cart') }}" data-module="cart-badge" data-label="{{ __('shop.cart.label') }}" data-label-count="{{ __('shop.cart.count') }}" aria-label="{{ __('shop.cart.label') }}"
   {{ $attributes->class('relative flex size-11 shrink-0 items-center justify-center rounded-full border-[1.5px] border-line bg-surface text-ink hover:border-primary hover:text-primary') }}>
    <x-icon name="cart" class="size-5"/>
    <span data-cart-count hidden aria-hidden="true" class="absolute -end-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-primary px-1 text-2xs font-extrabold text-white"></span>
</a>
