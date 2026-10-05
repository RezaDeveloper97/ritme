<?php

declare(strict_types=1);

namespace App\Providers\Filament;

use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Contact\Models\ContactMessage;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Faq\Models\FaqGroup;
use App\Domain\Faq\Models\FaqItem;
use App\Domain\Media\Models\Media;
use App\Domain\Newsletter\Models\Subscriber;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\GeneralSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category as ShopCategory;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Ordering\Models\Order;
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
use App\Filament\Resources\ContactMessages\ContactMessagePolicy;
use App\Filament\Resources\Directory\BookingRequestPolicy;
use App\Filament\Resources\Directory\DirectoryPolicy;
use App\Filament\Resources\Directory\JoinRequestPolicy;
use App\Filament\Resources\Directory\JoinRequests\PendingMediaController;
use App\Filament\Resources\Directory\PlaceReviewPolicy;
use App\Filament\Resources\Faq\FaqPolicy;
use App\Filament\Resources\Media\MediaPolicy;
use App\Filament\Resources\Newsletter\SubscriberPolicy;
use App\Filament\Resources\Seo\NotFoundLogs\NotFoundLogPolicy;
use App\Filament\Resources\Seo\Redirects\RedirectPolicy;
use App\Filament\Resources\Seo\StaticPageSeo\StaticPageSeoPolicy;
use App\Filament\Resources\Shop\OrderPolicy;
use App\Filament\Resources\Shop\ProductReviewPolicy;
use App\Filament\Resources\Shop\ShopCatalogPolicy;
use App\Filament\Resources\Shop\ShopPolicy;
use App\Filament\Widgets\AdminOverview;
use App\Filament\Widgets\Seo\SeoContentNeedingWork;
use App\Filament\Widgets\Seo\SeoHealthOverview;
use App\Filament\Widgets\Seo\SeoScoreTrend;
use App\Filament\Widgets\Seo\SeoTopIssues;
use App\Filament\Widgets\Seo\SeoTopNotFound;
use App\Filament\Widgets\Seo\SeoZeroResultSearches;
use App\Filament\Widgets\Shop\LowStockProducts;
use App\Filament\Widgets\Shop\ShopOrdersOverview;
use App\Models\User;
use Filament\Auth\MultiFactor\App\AppAuthentication;
use Filament\Auth\Pages\Login as LoginPage;
use Filament\Facades\Filament;
use Filament\FontProviders\LocalFontProvider;
use Filament\Forms\Components\Checkbox;
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
 * required for the roles in `filament.admin.mfa_required_roles`, login throttling (Filament: 5/min/IP), an inactivity
 * timeout and no remember-me.
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
        Gate::policy(FaqGroup::class, FaqPolicy::class);
        Gate::policy(FaqItem::class, FaqPolicy::class);
        Gate::policy(Subscriber::class, SubscriberPolicy::class);
        Gate::policy(ContactMessage::class, ContactMessagePolicy::class);
        Gate::policy(SeoMeta::class, StaticPageSeoPolicy::class);
        Gate::policy(Redirect::class, RedirectPolicy::class);       // L7-03
        Gate::policy(NotFoundLog::class, NotFoundLogPolicy::class); // L7-03
        foreach ([Place::class, PlaceCategory::class, City::class, District::class, Amenity::class, Landing::class] as $model) {
            Gate::policy($model, DirectoryPolicy::class);
        }
        Gate::policy(PlaceReview::class, PlaceReviewPolicy::class);
        Gate::policy(BookingRequest::class, BookingRequestPolicy::class);
        Gate::policy(JoinRequest::class, JoinRequestPolicy::class);
        Gate::policy(Product::class, ShopCatalogPolicy::class);      // L6-06
        Gate::policy(ShopCategory::class, ShopCatalogPolicy::class);
        Gate::policy(Brand::class, ShopPolicy::class);
        Gate::policy(Order::class, OrderPolicy::class);
        Gate::policy(ProductReview::class, ProductReviewPolicy::class);

        Event::listen(Login::class, RecordLastLogin::class);

        // L9-04b: remember-me logins are logged out again (EnforceSessionTimeout, F3), so the login form does not offer
        // the checkbox. Hidden fields are left out of the form state, so a crafted `remember` is ignored as well.
        Checkbox::configureUsing(static function (Checkbox $checkbox): void {
            if ($checkbox->getName() === 'remember') {
                $checkbox->hidden(static fn (mixed $livewire = null): bool => $livewire instanceof LoginPage);
            }
        });
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
            // L9-04b (F17): not-yet-public uploads (pending join photos) — logged-in admins only, never public.
            ->authenticatedRoutes(static function (): void {
                Route::get('pending-media/{path}', PendingMediaController::class)
                    ->where('path', PendingMediaController::PATH_PATTERN)
                    ->middleware(RequireMultiFactorForRoles::class)
                    ->name(PendingMediaController::ROUTE);
            })
            ->pages([
                Dashboard::class,
            ])
            ->widgets([
                AccountWidget::class,
                AdminOverview::class,
                ShopOrdersOverview::class, // L6-06 (shop managers + super-admins only)
                LowStockProducts::class,
                SeoHealthOverview::class, // L7-05 (SEO managers + super-admins only)
                SeoScoreTrend::class,
                SeoTopIssues::class,
                SeoTopNotFound::class,
                SeoZeroResultSearches::class,
                SeoContentNeedingWork::class,
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
