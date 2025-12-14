<?php

use App\Http\Controllers\TestPageController;
use App\Http\Controllers\TestPregnancyController;
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

// Test Pregnancy Routes (for testing pregnancy mode APIs)
Route::prefix('test-pregnancy')->group(function () {
    Route::get('/', [TestPregnancyController::class, 'index']);
    Route::get('/enums', [TestPregnancyController::class, 'getEnums']);

    // Profile & Onboarding
    Route::post('/activate', [TestPregnancyController::class, 'activatePregnancy']);
    Route::post('/deactivate', [TestPregnancyController::class, 'deactivatePregnancy']);
    Route::post('/onboarding', [TestPregnancyController::class, 'onboarding']);
    Route::get('/profile', [TestPregnancyController::class, 'getProfile']);
    Route::post('/profile', [TestPregnancyController::class, 'updateProfile']);
    Route::get('/status', [TestPregnancyController::class, 'getStatus']);

    // Symptoms
    Route::get('/symptoms', [TestPregnancyController::class, 'getSymptoms']);
    Route::post('/symptoms', [TestPregnancyController::class, 'storeSymptom']);
    Route::get('/symptoms/{date}', [TestPregnancyController::class, 'getSymptomByDate']);
    Route::delete('/symptoms/{date}', [TestPregnancyController::class, 'deleteSymptom']);

    // Weekly
    Route::get('/weekly', [TestPregnancyController::class, 'getWeeklyLogs']);
    Route::post('/weekly', [TestPregnancyController::class, 'storeWeeklyLog']);
    Route::get('/weekly/{week}', [TestPregnancyController::class, 'getWeeklyLog']);

    // Fetal Movement
    Route::get('/fetal-movement', [TestPregnancyController::class, 'getFetalMovements']);
    Route::post('/fetal-movement', [TestPregnancyController::class, 'storeFetalMovement']);

    // Content
    Route::get('/content/{week}', [TestPregnancyController::class, 'getWeeklyContent']);

    // Alerts
    Route::get('/alerts', [TestPregnancyController::class, 'getAlerts']);
    Route::get('/alerts/summary', [TestPregnancyController::class, 'getAlertsSummary']);
    Route::post('/alerts/{id}/read', [TestPregnancyController::class, 'markAlertAsRead']);
    Route::post('/alerts/read-all', [TestPregnancyController::class, 'markAllAlertsAsRead']);
    Route::post('/alerts/{id}/dismiss', [TestPregnancyController::class, 'dismissAlert']);
});
