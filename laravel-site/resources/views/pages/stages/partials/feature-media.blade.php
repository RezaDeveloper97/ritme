{{-- Mock-up column of a feature split: the phone on a lavender radial glow. Decorative. --}}
<div aria-hidden="true" class="flex flex-1 justify-center bg-[radial-gradient(circle,var(--color-lavender),transparent_70%)] py-5 max-lg:basis-75 max-sm:basis-full">
    @include('pages.stages.mock.phone', ['size' => 'feature', 'screen' => $feature->screen, 'data' => $feature->screenData])
</div>
