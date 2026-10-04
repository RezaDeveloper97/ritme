{{--
    <x-shop.subnav :departments="$subnav['departments']" :links="$subnav['links']" :label="$subnav['label']"/>
    Shop sub-navigation bar under the header (shop-list.html): department switch (root categories as pill links, the
    current one filled primary) + the current department's categories + the cart button. Plain links, no JS.
    departments: [label, href, active, icon] · links: [label, href, active].
--}}
@props(['departments' => [], 'links' => [], 'label' => ''])
<div {{ $attributes->class('flex items-center justify-between gap-4 border-b border-line bg-surface px-30 py-3.5 max-lg:flex-wrap max-lg:px-5') }}>
    <nav aria-label="{{ __('shop.subnav.departments') }}">
        <ul class="m-0 flex list-none gap-1.5 rounded-[26px] border border-line bg-canvas p-1 max-lg:flex-wrap">
            @foreach ($departments as $department)
                <li><a href="{{ $department['href'] }}" @if ($department['active']) aria-current="true" @endif @class([
                    'flex h-11 items-center gap-2 rounded-full px-5 text-base font-extrabold',
                    'bg-primary text-white hover:text-white' => $department['active'],
                    'text-muted hover:text-ink' => ! $department['active'],
                ])><x-icon :name="$department['icon']" class="size-[17px]"/>{{ $department['label'] }}</a></li>
            @endforeach
        </ul>
    </nav>
    <div class="flex items-center gap-5.5 max-lg:flex-wrap">
        @if ($links !== [])
            <nav aria-label="{{ $label }}">
                <ul class="m-0 flex list-none flex-wrap gap-x-5.5 gap-y-2 p-0">
                    @foreach ($links as $link)
                        <li><a href="{{ $link['href'] }}" @if ($link['active']) aria-current="page" @endif @class([
                            'text-base font-bold hover:text-primary',
                            'text-ink' => $link['active'],
                            'text-muted' => ! $link['active'],
                        ])>{{ $link['label'] }}</a></li>
                    @endforeach
                </ul>
            </nav>
        @endif
        <x-shop.cart-link/>
    </div>
</div>
