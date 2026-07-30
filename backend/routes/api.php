<?php

use App\Http\Controllers\Api\V1\ArticleController;
use App\Http\Controllers\Api\V1\BannerController;
use App\Http\Controllers\Api\V1\CycleCalculationController;
use App\Http\Controllers\Api\V1\DailyHealthLogController;
use App\Http\Controllers\Api\V1\HomeController;
use App\Http\Controllers\Api\V1\MessageController;
use App\Http\Controllers\Api\V1\OtpAuthController;
use App\Http\Controllers\Api\V1\PeriodLogController;
use App\Http\Controllers\Api\V1\PhaseContentController;
use App\Http\Controllers\Api\V1\PregnancyAlertController;
use App\Http\Controllers\Api\V1\PregnancyProfileController;
use App\Http\Controllers\Api\V1\PregnancySymptomController;
use App\Http\Controllers\Api\V1\PregnancyWeeklyController;
use App\Http\Controllers\Api\V1\ProfileController;
use App\Http\Controllers\Api\V1\ReminderController;
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

        // Home-page banners / promotions (grouped by slot)
        Route::get('/banners', [BannerController::class, 'index']);

        // Educational article library (admin-published): list + single read
        Route::get('/articles', [ArticleController::class, 'index']);
        Route::get('/articles/{slug}', [ArticleController::class, 'show']);

        // Profile routes
        Route::get('/profile', [ProfileController::class, 'show']);
        Route::post('/profile', [ProfileController::class, 'store']);
        Route::get('/profile/export', [ProfileController::class, 'export']);
        Route::delete('/account', [ProfileController::class, 'destroyAccount']);

        // Reminder routes
        Route::prefix('reminders')->group(function () {
            Route::get('/enums', [ReminderController::class, 'enums']);
            Route::get('/', [ReminderController::class, 'index']);
            Route::post('/', [ReminderController::class, 'store']);
            Route::put('/{id}', [ReminderController::class, 'update']);
            Route::delete('/{id}', [ReminderController::class, 'destroy']);
        });

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

            // Period logging (start / end an actual period)
            Route::get('/period/status', [PeriodLogController::class, 'status']);
            Route::post('/period/start', [PeriodLogController::class, 'start']);
            Route::post('/period/end', [PeriodLogController::class, 'end']);
            Route::get('/period/history', [PeriodLogController::class, 'history']);
            Route::put('/period/{period}', [PeriodLogController::class, 'update']);
            Route::delete('/period/{period}', [PeriodLogController::class, 'destroy']);

            // Matrix Messages (Personalized Phase-based Messages)
            Route::get('/matrix-messages', [CycleCalculationController::class, 'matrixMessages']);
            Route::get('/matrix-enums', [CycleCalculationController::class, 'matrixEnums']);

            // Phase Details educational content (DB-driven, admin-editable)
            Route::get('/phase-content/{phase}', [PhaseContentController::class, 'show']);
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

        // Unified Message System (works for both cycle and pregnancy modes)
        Route::prefix('messages')->group(function () {
            Route::get('/daily', [MessageController::class, 'daily']);
            Route::get('/enums', [MessageController::class, 'enums']);
            Route::get('/mode', [MessageController::class, 'mode']);
        });

        // Home Page (aggregated dashboard + per-section endpoints + actions)
        Route::prefix('home')->group(function () {
            Route::get('/', [HomeController::class, 'index']);
            Route::get('/sections/{section}', [HomeController::class, 'section']);

            // Section actions
            Route::post('/tasks/{task}/toggle', [HomeController::class, 'toggleTask']);
            Route::post('/challenges/{challenge}/toggle', [HomeController::class, 'toggleChallenge']);

            // Notifications (header bell)
            Route::get('/notifications', [HomeController::class, 'notifications']);
            Route::post('/notifications/read-all', [HomeController::class, 'markAllNotificationsRead']);
            Route::post('/notifications/{notification}/read', [HomeController::class, 'markNotificationRead']);
        });
    });
});
