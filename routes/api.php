<?php

use App\Http\Controllers\Api\V1\OtpAuthController;
use App\Http\Controllers\Api\V1\ProfileController;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| API Routes
|--------------------------------------------------------------------------
|
| Here is where you can register API routes for your application. These
| routes are loaded by the RouteServiceProvider and all of them will
| be assigned to the "api" middleware group. Make something great!
|
*/

// API Version 1
Route::prefix('v1')->group(function () {
    // Public routes - OTP Authentication
    Route::prefix('auth')->group(function () {
        Route::post('/send-otp', [OtpAuthController::class, 'sendOtp']);
        Route::post('/verify-otp', [OtpAuthController::class, 'verifyOtp']);
    });

    // Protected routes
    Route::middleware('auth:api')->group(function () {
        Route::prefix('auth')->group(function () {
            Route::post('/logout', [OtpAuthController::class, 'logout']);
            Route::get('/user', [OtpAuthController::class, 'user']);
        });

        // Profile routes
        Route::get('/profile', [ProfileController::class, 'show']);
        Route::post('/profile', [ProfileController::class, 'store']);
    });
});
