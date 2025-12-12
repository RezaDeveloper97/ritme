<?php

use App\Http\Controllers\TestPageController;
use Illuminate\Support\Facades\Route;

Route::get('/', function () {
    return view('welcome');
});

// Test Page Routes (for testing profile and cycle calculations)
Route::prefix('test-page')->group(function () {
    Route::get('/', [TestPageController::class, 'index']);
    Route::get('/profile', [TestPageController::class, 'getProfile']);
    Route::post('/profile', [TestPageController::class, 'updateProfile']);
    Route::post('/cycle-data', [TestPageController::class, 'getCycleData']);
});
