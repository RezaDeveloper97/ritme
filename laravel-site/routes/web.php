<?php

declare(strict_types=1);

use Illuminate\Support\Facades\Route;

// Placeholder until L1-05 registers the real page routes (no closures: route:cache must work).
Route::view('/', 'welcome')->name('home');

// L1-02 shell preview for fidelity screenshots (tools/shot.mjs); never registered in production.
if (! app()->isProduction()) {
    Route::view('/_preview/layout/{variant}', 'layouts.preview')->whereIn('variant', ['dark', 'light'])->name('preview.layout');
}
