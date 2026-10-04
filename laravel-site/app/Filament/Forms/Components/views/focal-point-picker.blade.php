@php
    $imageUrl = $getImageUrl();
@endphp
<x-dynamic-component :component="$getFieldWrapperView()" :field="$field">
    <div
        x-data="{
            state: $wire.$entangle(@js($getStatePath())),
            get x() { return Number(this.state?.x ?? 0.5) },
            get y() { return Number(this.state?.y ?? 0.5) },
            set(x, y) {
                const clamp = (v) => Math.round(Math.min(1, Math.max(0, v)) * 10000) / 10000
                this.state = { x: clamp(x), y: clamp(y) }
            },
            pick(event) {
                const box = event.currentTarget.getBoundingClientRect()
                this.set((event.clientX - box.left) / box.width, (event.clientY - box.top) / box.height)
            },
        }"
        class="flex flex-col gap-2"
    >
        @if ($imageUrl)
            <div
                class="relative inline-block max-w-full cursor-crosshair self-start overflow-hidden rounded-lg ring-1 ring-gray-950/10 dark:ring-white/10"
                role="slider"
                tabindex="0"
                aria-label="نقطه کانونی تصویر"
                x-bind:aria-valuetext="`افقی ${Math.round(x * 100)}٪، عمودی ${Math.round(y * 100)}٪`"
                x-on:click="pick($event)"
                x-on:keydown.arrow-left.prevent="set(x - 0.05, y)"
                x-on:keydown.arrow-right.prevent="set(x + 0.05, y)"
                x-on:keydown.arrow-up.prevent="set(x, y - 0.05)"
                x-on:keydown.arrow-down.prevent="set(x, y + 0.05)"
                dir="ltr"
                data-focal-picker
            >
                <img src="{{ $imageUrl }}" alt="" class="block max-h-96 w-auto max-w-full select-none" draggable="false">
                <svg class="pointer-events-none absolute inset-0 h-full w-full" aria-hidden="true">
                    <line x1="0" x2="100%" x-bind:y1="`${y * 100}%`" x-bind:y2="`${y * 100}%`" class="stroke-white/70" stroke-width="1"></line>
                    <line y1="0" y2="100%" x-bind:x1="`${x * 100}%`" x-bind:x2="`${x * 100}%`" class="stroke-white/70" stroke-width="1"></line>
                    <circle r="11" x-bind:cx="`${x * 100}%`" x-bind:cy="`${y * 100}%`" class="fill-primary-500/40 stroke-white" stroke-width="3"></circle>
                </svg>
            </div>
            <p class="text-sm text-gray-500 dark:text-gray-400">
                افقی <span x-text="Math.round(x * 100)"></span>٪ — عمودی <span x-text="Math.round(y * 100)"></span>٪
            </p>
        @else
            <p class="text-sm text-gray-500 dark:text-gray-400">پیش‌نمایش در دسترس نیست.</p>
        @endif
    </div>
</x-dynamic-component>
