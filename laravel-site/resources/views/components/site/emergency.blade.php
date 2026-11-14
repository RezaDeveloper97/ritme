{{--
    <x-site.emergency number="115">اگر خونریزی خیلی زیاد داری، …</x-site.emergency>
    Redesign «کی نباید منتظر ماند» band: rose-soft box (radius 24, p 28/32), rose title, ink text, rose «تماس با ۱۱۵»
    pill that dials the emergency number from settings.
--}}
@props(['number' => '115', 'title' => 'کی نباید منتظر ماند'])
<aside aria-label="{{ $title }}" {{ $attributes->class('flex items-center gap-8 rounded-4xl bg-danger-soft px-8 py-7 max-sm:flex-col max-sm:items-stretch max-sm:gap-4 max-sm:px-5') }}>
    <div class="flex flex-1 flex-col gap-1.5">
        <b class="text-xl text-rose">{{ $title }}</b>
        <p class="m-0 text-lg leading-relaxed font-semibold text-ink">{{ $slot }}</p>
    </div>
    <x-site.button :href="'tel:'.$number" variant="rose" icon="phone" size="md">تماس با {{ fa_digits($number) }}</x-site.button>
</aside>
