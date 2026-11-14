{{--
    <x-home.ring class="…"/>   Decorative dot ring of the home hero (design Main.html): three pulsing outlines, a dashed
    orbit and 28 dots on r=88 that turn slowly. aria-hidden; colours are token fill/stroke classes (no inline style);
    the motion lives in the «home» block of app.css and stops under prefers-reduced-motion.
--}}
@php
    $dots = [];
    for ($i = 0; $i < 28; $i++) {
        $angle = deg2rad($i * 360 / 28);
        $dots[] = [
            'x' => round(88 * sin($angle), 1),
            'y' => round(-88 * cos($angle), 1),
            'class' => match (true) {
                $i < 5 => 'fill-rose',
                $i >= 11 && $i < 17 => 'fill-rose-mid',
                default => 'fill-rose/20',
            },
            'r' => $i < 5 ? 4.2 : 3.2,
        ];
    }
@endphp
<svg aria-hidden="true" viewBox="-110 -110 220 220" {{ $attributes->class('pointer-events-none block overflow-visible') }}>
    @foreach (['home-pulse', 'home-pulse home-pulse-2', 'home-pulse home-pulse-3'] as $pulse)
        <circle class="{{ $pulse }} fill-none stroke-rose" cx="0" cy="0" r="40" stroke-width=".6"/>
    @endforeach
    <circle class="fill-none stroke-rose/20" cx="0" cy="0" r="66" stroke-width=".8" stroke-dasharray="2 4"/>
    <g class="home-ring">
        @foreach ($dots as $dot)
            <circle class="{{ $dot['class'] }}" cx="{{ $dot['x'] }}" cy="{{ $dot['y'] }}" r="{{ $dot['r'] }}"/>
        @endforeach
    </g>
</svg>
