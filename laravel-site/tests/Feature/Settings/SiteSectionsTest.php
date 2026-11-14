<?php

declare(strict_types=1);

use App\Domain\Content\Enums\SiteSection;
use App\Domain\Content\SiteNavigation;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Settings\PagesSettingsPage;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Livewire\Livewire;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
});

function switchOff(SiteSection ...$sections): void
{
    $values = [];
    foreach ($sections as $section) {
        $values[$section->value] = false;
    }
    app(UpdateSettings::class)->handle(SettingGroup::Pages, $values);
}

it('maps routes, sitemaps and search providers to sections', function (): void {
    expect(SiteSection::forRoute('stage.ttc'))->toBe(SiteSection::Ttc)
        ->and(SiteSection::forRoute('contact.store'))->toBe(SiteSection::Contact)
        ->and(SiteSection::forRoute('blog.show'))->toBe(SiteSection::Blog)
        ->and(SiteSection::forRoute('shop.cart.add'))->toBe(SiteSection::Shop)
        ->and(SiteSection::forRoute('newsletter.confirm'))->toBe(SiteSection::Newsletter)
        ->and(SiteSection::forRoute('home'))->toBeNull()
        ->and(SiteSection::forRoute('privacy'))->toBeNull()
        ->and(SiteSection::forRoute('terms'))->toBeNull()
        ->and(SiteSection::forSitemap('posts'))->toBe(SiteSection::Blog)
        ->and(SiteSection::forSitemap('blog-tags'))->toBe(SiteSection::Blog)
        ->and(SiteSection::forSitemap('shop-products'))->toBe(SiteSection::Shop)
        ->and(SiteSection::forSitemap('pages'))->toBeNull()
        ->and(SiteSection::forSearchProvider('places'))->toBe(SiteSection::Directory);
});

it('puts every section in exactly one admin group', function (): void {
    $grouped = array_merge(SiteSection::modules(), SiteSection::stages(), SiteSection::pages());

    expect(count($grouped))->toBe(count(SiteSection::cases()))
        ->and(array_unique(array_map(static fn (SiteSection $s): string => $s->value, $grouped)))->toHaveCount(count(SiteSection::cases()));
});

it('has every section on by default', function (): void {
    $pages = app(SettingsRepository::class)->all()->pages;

    foreach (SiteSection::cases() as $section) {
        expect($pages->enabled($section))->toBeTrue();
    }
});

it('answers 404 for a switched-off section and keeps the rest of the site', function (): void {
    $this->get('/about')->assertOk();
    $this->get('/blog')->assertOk();

    switchOff(SiteSection::About, SiteSection::Blog);

    $this->get('/about')->assertNotFound();
    $this->get('/blog')->assertNotFound();
    $this->get('/blog/category/anything')->assertNotFound();
    $this->get('/blog/feed')->assertNotFound();
    $this->get('/faq')->assertOk();
    $this->get('/privacy')->assertOk();
});

it('does not serve a page cached before its section was switched off', function (): void {
    $this->get('/tools')->assertOk();
    $this->get('/tools')->assertOk(); // HIT

    switchOff(SiteSection::Tools);

    $this->get('/tools')->assertNotFound();
});

it('drops switched-off sections from the main nav and footer', function (): void {
    switchOff(SiteSection::Shop, SiteSection::Blog, SiteSection::Cycle, SiteSection::Tools, SiteSection::About);
    $navigation = app(SiteNavigation::class);

    $main = array_map(static fn ($link): string => $link->label, $navigation->main('home'));
    expect($main)->toBe(['خانه', 'امکانات', 'سؤالات متداول']);

    $footer = $navigation->footer();
    $labels = collect($footer)->flatMap(static fn ($column): array => array_map(static fn ($link): string => $link->label, $column->links))->all();

    expect($labels)->not->toContain('مجله')->not->toContain('درباره ما')->not->toContain('چرخه و پریود')->toContain('اقدام به بارداری');
});

it('drops a footer column when every link in it is off', function (): void {
    switchOff(...SiteSection::stages());

    $titles = array_map(static fn ($column): string => $column->title, app(SiteNavigation::class)->footer());

    expect($titles)->not->toContain('مرحله‌ها')->toContain('ریتمی');
});

it('leaves switched-off sections out of the sitemap', function (): void {
    $this->get('/sitemaps/pages.xml')->assertOk()->assertSee('/about', false);

    switchOff(SiteSection::About, SiteSection::Shop);

    $this->get('/sitemaps/pages.xml')->assertOk()->assertDontSee('/about', false)->assertDontSee('/shop', false);
    $this->get('/sitemaps/shop-products.xml')->assertNotFound();
    $this->get('/sitemap.xml')->assertOk()->assertDontSee('shop-products', false);
});

it('hides home page stage cards of switched-off sections', function (): void {
    $this->get('/')->assertOk()->assertSee('href="'.route('stage.teen').'"', false);

    switchOff(SiteSection::Teen);

    $this->get('/')->assertOk()->assertDontSee('href="'.route('stage.teen').'"', false);
});

it('lets a super-admin switch sections from the admin page', function (): void {
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');
    $admin = User::factory()->create();
    $admin->assignRole(AdminRole::SuperAdmin->value);
    $this->actingAs($admin);

    Livewire::test(PagesSettingsPage::class)
        ->assertSchemaStateSet(['shop' => true, 'blog' => true], 'form')
        ->fillForm(['shop' => false])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertNotified();

    expect(app(SettingsRepository::class)->all()->pages->enabled(SiteSection::Shop))->toBeFalse()
        ->and(app(SettingsRepository::class)->all()->pages->enabled(SiteSection::Blog))->toBeTrue();
});

it('keeps the page switches away from editors', function (): void {
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');
    $editor = User::factory()->create();
    $editor->assignRole(AdminRole::Editor->value);
    $this->actingAs($editor);

    Livewire::test(PagesSettingsPage::class)->assertForbidden();
});
