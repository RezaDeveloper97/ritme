<?php

declare(strict_types=1);

use Illuminate\Support\Facades\Route;

// Placeholder until L1-05 registers the real page routes (no closures: route:cache must work).
Route::view('/', 'welcome')->name('home');
