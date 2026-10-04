<?php

declare(strict_types=1);

use Illuminate\Foundation\Vite;
use Illuminate\Support\Facades\Blade;

it('renders hashed build URLs through @vite', function (): void {
    if (! is_file(public_path('build/manifest.json'))) {
        $this->markTestSkipped('Run npm run build first.');
    }

    app()->forgetInstance(Vite::class); // the base TestCase fakes Vite

    $html = Blade::render("@vite(['resources/css/app.css', 'resources/js/app.js'])");

    expect($html)
        ->toMatch('#/build/assets/app-[A-Za-z0-9_-]+\.css#')
        ->toMatch('#/build/assets/app-[A-Za-z0-9_-]+\.js#');
});
