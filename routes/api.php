<?php

use App\Http\Controllers\Api\V1\CycleCalculationController;
use App\Http\Controllers\Api\V1\DailyHealthLogController;
use App\Http\Controllers\Api\V1\OtpAuthController;
use App\Http\Controllers\Api\V1\PregnancyAlertController;
use App\Http\Controllers\Api\V1\PregnancyProfileController;
use App\Http\Controllers\Api\V1\PregnancySymptomController;
use App\Http\Controllers\Api\V1\PregnancyWeeklyController;
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

        // Daily Health Log routes
        Route::prefix('health-logs')->group(function () {
            Route::get('/enums', [DailyHealthLogController::class, 'enums']);
            Route::get('/', [DailyHealthLogController::class, 'index']);
            Route::post('/', [DailyHealthLogController::class, 'store']);
            Route::get('/{date}', [DailyHealthLogController::class, 'show']);
            Route::delete('/{date}', [DailyHealthLogController::class, 'destroy']);
        });

        // Cycle Calculation routes (Health Data Engine)
        Route::prefix('cycle')->group(function () {
            Route::get('/enums', [CycleCalculationController::class, 'enums']);
            Route::get('/status', [CycleCalculationController::class, 'status']);
            Route::get('/today', [CycleCalculationController::class, 'today']);
            Route::get('/date/{date}', [CycleCalculationController::class, 'forDate']);
            Route::get('/month/{year}/{month}', [CycleCalculationController::class, 'month']);
            Route::post('/recalculate', [CycleCalculationController::class, 'recalculate']);

            // Matrix Messages (Personalized Phase-based Messages)
            Route::get('/matrix-messages', [CycleCalculationController::class, 'matrixMessages']);
            Route::get('/matrix-enums', [CycleCalculationController::class, 'matrixEnums']);
        });

        // Pregnancy Mode routes
        Route::prefix('pregnancy')->group(function () {
            // Profile & Onboarding
            Route::post('/activate', [PregnancyProfileController::class, 'activate']);
            Route::post('/deactivate', [PregnancyProfileController::class, 'deactivate']);
            Route::post('/onboarding', [PregnancyProfileController::class, 'onboarding']);
            Route::get('/profile', [PregnancyProfileController::class, 'show']);
            Route::put('/profile', [PregnancyProfileController::class, 'update']);
            Route::post('/confirm', [PregnancyProfileController::class, 'confirm']);
            Route::get('/status', [PregnancyProfileController::class, 'status']);
            Route::get('/enums', [PregnancyProfileController::class, 'enums']);

            // Daily Symptoms
            Route::get('/symptoms/enums', [PregnancySymptomController::class, 'enums']);
            Route::get('/symptoms', [PregnancySymptomController::class, 'index']);
            Route::post('/symptoms', [PregnancySymptomController::class, 'store']);
            Route::get('/symptoms/{date}', [PregnancySymptomController::class, 'show']);
            Route::delete('/symptoms/{date}', [PregnancySymptomController::class, 'destroy']);

            // Weekly Monitoring
            Route::get('/weekly/enums', [PregnancyWeeklyController::class, 'enums']);
            Route::get('/weekly', [PregnancyWeeklyController::class, 'index']);
            Route::post('/weekly', [PregnancyWeeklyController::class, 'store']);
            Route::get('/weekly/{week}', [PregnancyWeeklyController::class, 'show']);

            // Fetal Movement
            Route::get('/fetal-movement', [PregnancyWeeklyController::class, 'indexFetalMovement']);
            Route::post('/fetal-movement', [PregnancyWeeklyController::class, 'storeFetalMovement']);

            // Weekly Content
            Route::get('/content/{week}', [PregnancyWeeklyController::class, 'content']);

            // Alerts
            Route::get('/alerts/summary', [PregnancyAlertController::class, 'summary']);
            Route::get('/alerts', [PregnancyAlertController::class, 'index']);
            Route::get('/alerts/{id}', [PregnancyAlertController::class, 'show']);
            Route::post('/alerts/{id}/read', [PregnancyAlertController::class, 'markAsRead']);
            Route::post('/alerts/read-all', [PregnancyAlertController::class, 'markAllAsRead']);
            Route::post('/alerts/{id}/dismiss', [PregnancyAlertController::class, 'dismiss']);
        });
    });
});
