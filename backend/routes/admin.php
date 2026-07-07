<?php

use App\Http\Controllers\Admin\AccountController;
use App\Http\Controllers\Admin\AdminController;
use App\Http\Controllers\Admin\AffirmationController;
use App\Http\Controllers\Admin\ArticleController;
use App\Http\Controllers\Admin\AuthController;
use App\Http\Controllers\Admin\ChallengeController;
use App\Http\Controllers\Admin\DashboardController;
use App\Http\Controllers\Admin\MessageContentController;
use App\Http\Controllers\Admin\PregnancyWeekController;
use App\Http\Controllers\Admin\TaskTemplateController;
use App\Http\Controllers\Admin\UserController;
use Illuminate\Support\Facades\Response;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| Admin Panel Routes (Blade)
|--------------------------------------------------------------------------
| Mounted under the "/admin" prefix and "admin." name, guarded by the
| dedicated `admin` session guard. Only loaded when ADMIN_PANEL_ENABLED=true
| (see bootstrap/app.php).
|
| The stylesheet is served THROUGH this "/admin" prefix (not /assets) so a
| single reverse-proxy rule ("/admin -> backend") covers the whole panel —
| making it trivial to later move the panel to its own subdomain.
*/

// Stylesheet (public) — kept under /admin so one proxy rule serves everything.
Route::get('style.css', fn () => Response::file(public_path('assets/admin.css'), [
    'Content-Type' => 'text/css',
    'Cache-Control' => 'public, max-age=86400',
]))->name('style');

// Guest
Route::get('login', [AuthController::class, 'showLogin'])->name('login');
Route::post('login', [AuthController::class, 'login'])->name('login.attempt');

// Authenticated admins
Route::middleware(['auth:admin', 'admin.active'])->group(function () {
    Route::post('logout', [AuthController::class, 'logout'])->name('logout');

    Route::get('/', [DashboardController::class, 'index'])->name('dashboard');

    // Own account
    Route::get('account/password', [AccountController::class, 'editPassword'])->name('password.edit');
    Route::put('account/password', [AccountController::class, 'updatePassword'])->name('password.update');

    // User management
    Route::get('users', [UserController::class, 'index'])->name('users.index');
    Route::get('users/{user}', [UserController::class, 'show'])->name('users.show');
    Route::put('users/{user}', [UserController::class, 'update'])->name('users.update');
    Route::post('users/{user}/block', [UserController::class, 'block'])->name('users.block');
    Route::post('users/{user}/unblock', [UserController::class, 'unblock'])->name('users.unblock');
    Route::delete('users/{user}', [UserController::class, 'destroy'])->name('users.destroy');

    // Content management
    Route::post('articles/{article}/toggle', [ArticleController::class, 'toggle'])->name('articles.toggle');
    Route::resource('articles', ArticleController::class)->except('show');

    Route::post('affirmations/{affirmation}/toggle', [AffirmationController::class, 'toggle'])->name('affirmations.toggle');
    Route::resource('affirmations', AffirmationController::class)->except('show');

    Route::post('challenges/{challenge}/toggle', [ChallengeController::class, 'toggle'])->name('challenges.toggle');
    Route::resource('challenges', ChallengeController::class)->except('show');

    Route::post('task-templates/{taskTemplate}/toggle', [TaskTemplateController::class, 'toggle'])->name('task-templates.toggle');
    Route::resource('task-templates', TaskTemplateController::class)->except('show')->parameters(['task-templates' => 'taskTemplate']);

    Route::resource('pregnancy-weeks', PregnancyWeekController::class)->except('show')->parameters(['pregnancy-weeks' => 'pregnancyWeek']);

    // Smart messages (editable, moderated)
    Route::get('messages', [MessageContentController::class, 'index'])->name('messages.index');
    Route::get('messages/{message}/edit', [MessageContentController::class, 'edit'])->name('messages.edit');
    Route::put('messages/{message}', [MessageContentController::class, 'update'])->name('messages.update');
    Route::post('messages/{message}/approve', [MessageContentController::class, 'approve'])->name('messages.approve');
    Route::post('messages/{message}/toggle', [MessageContentController::class, 'toggle'])->name('messages.toggle');

    // Admin account management (super admins only)
    Route::middleware('admin.super')->group(function () {
        Route::resource('admins', AdminController::class)->except('show');
    });
});
