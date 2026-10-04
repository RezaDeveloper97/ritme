<?php

declare(strict_types=1);

namespace App\Providers\Filament;

use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Media\Models\Media;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\GeneralSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\InitialsAvatarProvider;
use App\Filament\Commands\CreateAdminCommand;
use App\Filament\Http\Middleware\EnforceSessionTimeout;
use App\Filament\Http\Middleware\RequireMultiFactorForRoles;
use App\Filament\Listeners\RecordLastLogin;
use App\Filament\Policies\ActivityPolicy;
use App\Filament\Policies\UserPolicy;
use App\Filament\Resources\Blog\Authors\AuthorPolicy;
use App\Filament\Resources\Blog\Categories\CategoryPolicy;
use App\Filament\Resources\Blog\Posts\PostPolicy;
use App\Filament\Resources\Blog\Posts\PostPreviewController;
use App\Filament\Resources\Blog\Tags\TagPolicy;
use App\Filament\Resources\Media\MediaPolicy;
use App\Filament\Widgets\AdminOverview;
use App\Models\User;
use Filament\Auth\MultiFactor\App\AppAuthentication;
use Filament\Facades\Filament;
use Filament\FontProviders\LocalFontProvider;
use Filament\Http\Middleware\Authenticate;
use Filament\Http\Middleware\AuthenticateSession;
use Filament\Http\Middleware\DisableBladeIconComponents;
use Filament\Http\Middleware\DispatchServingFilamentEvent;
use Filament\Pages\Dashboard;
use Filament\Panel;
use Filament\PanelProvider;
use Filament\Support\Colors\Color;
use Filament\Widgets\AccountWidget;
use Illuminate\Auth\Events\Login;
use Illuminate\Cookie\Middleware\AddQueuedCookiesToResponse;
use Illuminate\Cookie\Middleware\EncryptCookies;
use Illuminate\Foundation\Http\Middleware\VerifyCsrfToken;
use Illuminate\Routing\Middleware\SubstituteBindings;
use Illuminate\Session\Middleware\StartSession;
use Illuminate\Support\Facades\Event;
use Illuminate\Support\Facades\Gate;
use Illuminate\Support\Facades\Route;
use Illuminate\View\Middleware\ShareErrorsFromSession;
use Spatie\Activitylog\Models\Activity;
use Throwable;

/**
 * The Filament admin panel (thin delivery layer): Persian/RTL (APP_LOCALE=fa), self-hosted Vazirmatn through the Vite
 * theme, brand colour from the design tokens, brand name from settings, deny-by-default policies, app (TOTP) MFA
 * required for super-admins, login throttling (Filament: 5/min/IP) and an inactivity timeout.
 */
final class AdminPanelProvider extends PanelProvider
{
    public function register(): void
    {
        parent::register();

        $this->commands([CreateAdminCommand::class]);
    }

    public function boot(): void
    {
        Gate::policy(User::class, UserPolicy::class);
        Gate::policy(Activity::class, ActivityPolicy::class);
        Gate::policy(Media::class, MediaPolicy::class);
        Gate::policy(Post::class, PostPolicy::class);
        Gate::policy(Category::class, CategoryPolicy::class);
        Gate::policy(Tag::class, TagPolicy::class);
        Gate::policy(Author::class, AuthorPolicy::class);

        Event::listen(Login::class, RecordLastLogin::class);
    }

    public function panel(Panel $panel): Panel
    {
        $path = config('filament.admin.path', 'admin');
        $brandColor = config('filament.admin.brand_color', '#6e54f0');

        return $panel
            ->default()
            ->id('admin')
            ->path(is_string($path) ? trim($path, '/') : 'admin')
            ->login()
            ->profile(isSimple: false)
            ->multiFactorAuthentication(
                [AppAuthentication::make()->recoverable()],
                isRequired: true,
            )
            ->multiFactorAuthenticationRequiredMiddlewareName(RequireMultiFactorForRoles::class)
            ->brandName(static fn (): string => self::brandName())
            ->colors(['primary' => Color::hex(is_string($brandColor) ? $brandColor : '#6e54f0')])
            ->font('Vazirmatn', provider: LocalFontProvider::class)
            ->defaultAvatarProvider(InitialsAvatarProvider::class)
            ->viteTheme('resources/css/filament/admin/theme.css')
            ->spa(false)
            ->discoverResources(in: app_path('Filament/Resources'), for: 'App\Filament\Resources')
            ->discoverPages(in: app_path('Filament/Pages'), for: 'App\Filament\Pages')
            // Draft preview: temporary signed URL, no admin login needed (shareable with a reviewer), noindex.
            ->routes(static function (): void {
                Route::get('blog/preview/{post}', PostPreviewController::class)
                    ->middleware('signed')
                    ->name(PostPreviewController::ROUTE);
            })
            ->pages([
                Dashboard::class,
            ])
            ->widgets([
                AccountWidget::class,
                AdminOverview::class,
            ])
            ->middleware([
                EncryptCookies::class,
                AddQueuedCookiesToResponse::class,
                StartSession::class,
                AuthenticateSession::class,
                ShareErrorsFromSession::class,
                VerifyCsrfToken::class,
                SubstituteBindings::class,
                DisableBladeIconComponents::class,
                DispatchServingFilamentEvent::class,
            ])
            ->authMiddleware([
                Authenticate::class,
                EnforceSessionTimeout::class,
            ]);
    }

    private static function brandName(): string
    {
        try {
            $general = app(SettingsRepository::class)->group(SettingGroup::General);
        } catch (Throwable) {
            return 'ریتمی';
        }

        return $general instanceof GeneralSettings && $general->siteName !== '' ? $general->siteName : 'ریتمی';
    }
}
