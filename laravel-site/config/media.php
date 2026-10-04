<?php

declare(strict_types=1);

/*
|--------------------------------------------------------------------------
| Media pipeline (App\Domain\Media)
|--------------------------------------------------------------------------
|
| Every upload is validated (sniffed mime, size, pixel limits, SVG sanitised),
| auto-oriented, stripped of metadata and stored as a capped original. The
| OptimizeMedia job then writes the variants below. Until a variant exists the
| original is served, so a slow or failed queue never produces a broken image.
|
|   php artisan media:regenerate                    all media, automatic presets
|   php artisan media:regenerate --id=12 --preset=og
|
*/

return [

    /*
    | Disk the files are written to. The `public` disk writes into
    | public/media (no storage:link needed on cPanel) — see filesystems.php.
    */
    'disk' => env('MEDIA_DISK', 'public'),

    /*
    | Image driver: auto (Imagick when the extension is loaded, else GD), gd or imagick.
    */
    'driver' => env('MEDIA_DRIVER', 'auto'),

    /*
    | Queue for OptimizeMedia. null connection = QUEUE_CONNECTION (database in
    | production, drained by the cron-driven scheduler; sync works too).
    */
    'queue' => [
        'connection' => env('MEDIA_QUEUE_CONNECTION'),
        'name' => env('MEDIA_QUEUE', 'default'),
    ],

    /*
    | PHP memory limit raised (never lowered) while decoding/encoding. A decoded
    | 8000 px bitmap needs ~256 MB on GD.
    */
    'memory_limit' => '512M',

    /*
    | Validation limits.
    */
    'max_bytes' => 15 * 1024 * 1024,
    'max_dimension' => 8000,

    /*
    | Sniffed mime types accepted on upload. SVG is accepted only after
    | sanitising (enshrined/svg-sanitize); set svg to false to reject it.
    */
    'mimes' => ['image/jpeg', 'image/png', 'image/webp', 'image/gif', 'image/avif'],
    'svg' => true,

    /*
    | The stored original is scaled down to fit this box (px). Never upscaled.
    */
    'original_max' => 2560,

    /*
    | Variant presets. `widths` = responsive widths (aspect kept); width+height
    | with crop = cropped around the focal point. `auto` = generated for every
    | upload (others only via media:regenerate --preset or an explicit request).
    | `formats` lists formats from `formats` below; `fallback` = jpg for photos,
    | png when the image has transparency. Never upscaled: widths larger than
    | the original are skipped, crops shrink to fit.
    */
    'presets' => [
        'mobile' => ['widths' => [480, 768], 'auto' => true],
        'desktop' => ['widths' => [1280, 1920], 'auto' => true],
        'thumb' => ['width' => 320, 'height' => 320, 'crop' => true, 'auto' => true],
        'og' => ['width' => 1200, 'height' => 630, 'crop' => true, 'auto' => true, 'formats' => ['fallback']],
        'square' => ['width' => 600, 'height' => 600, 'crop' => true, 'auto' => false],
    ],

    /*
    | Animated GIFs are kept as-is; only a still poster (first frame, scaled to
    | this width) and the thumb are generated.
    */
    'poster_width' => 1280,

    /*
    | Variant formats in preference order. avif is skipped automatically when
    | the driver cannot encode it (GD needs imageavif + AVIF support).
    */
    'formats' => ['avif', 'webp', 'fallback'],

    'quality' => [
        'avif' => 55,
        'webp' => 78,
        'jpg' => 82,
        'original' => 86, // re-encoded original (jpg/webp/avif)
    ],

    /*
    | Low-quality image placeholder: tiny WebP data URI, at most max_bytes.
    */
    'lqip' => [
        'width' => 16,
        'quality' => 40,
        'max_bytes' => 600,
    ],

];
