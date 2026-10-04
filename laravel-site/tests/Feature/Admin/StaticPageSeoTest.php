<?php

declare(strict_types=1);

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Actions\ListStaticPageSeo;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Seo\StaticPageSeo\Pages\EditStaticPageSeo;
use App\Filament\Resources\Seo\StaticPageSeo\Pages\ListStaticPageSeoPages;
use App\Filament\Resources\Seo\StaticPageSeo\StaticPageSeoResource;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Testing\TestAction;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\Auth;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;

function seoAdmin(?AdminRole $role = AdminRole::SeoManager): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return $user;
}

/**
 * @return array{title: string|null, description: string|null}
 */
function renderedHead(string $html): array
{
    preg_match('#<title>(.*?)</title>#s', $html, $title);
    preg_match('#<meta name="description" content="([^"]*)"#', $html, $description);
    $decode = static fn (?string $v): ?string => $v === null ? null : html_entity_decode($v, ENT_QUOTES | ENT_HTML5, 'UTF-8');

    return ['title' => $decode($title[1] ?? null), 'description' => $decode($description[1] ?? null)];
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
});

it('lets SEO managers and super-admins in and denies everyone else', function (): void {
    $index = StaticPageSeoResource::getUrl('index');
    $edit = StaticPageSeoResource::getUrl('edit', ['page' => 'stage-cycle']);

    $this->get($index)->assertRedirect('/admin/login');

    $manager = seoAdmin();
    $this->actingAs($manager)->get($index)->assertOk();
    $this->actingAs($manager)->get($edit)->assertOk();

    // Super-admins must enrol MFA before any admin page (RequireMultiFactorForRoles); the policy allows them.
    $super = seoAdmin(AdminRole::SuperAdmin);
    expect($super->can('viewAny', SeoMeta::class))->toBeTrue()
        ->and($super->can('update', new SeoMeta(['route_name' => 'home'])))->toBeTrue();

    foreach ([AdminRole::Editor, AdminRole::Support, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
        $user = seoAdmin($role);
        $this->actingAs($user)->get($index)->assertForbidden();
        $this->actingAs($user)->get($edit)->assertForbidden();
    }

    $inactive = seoAdmin();
    $inactive->forceFill(['is_active' => false])->save();
    expect($inactive->can('viewAny', SeoMeta::class))->toBeFalse();
});

it('404s for unknown and non-indexable pages', function (string $key): void {
    $this->actingAs(seoAdmin())->get(StaticPageSeoResource::getUrl('index').'/'.$key)->assertNotFound();
})->with(['nope', 'directory-join', 'shop-cart']);

it('lists every indexable registry page with its effective title, even without a seo_meta row', function (): void {
    SeoMeta::query()->create(['route_name' => 'stage.cycle', 'title' => 'عنوان سفارشی چرخه برای آزمون مدیر سئو']);
    $this->actingAs(seoAdmin());

    $pages = ListStaticPageSeo::pages();
    expect($pages)->toHaveCount(20)
        ->and($pages)->not->toContain(StaticPage::DirectoryJoin, StaticPage::ShopCart, StaticPage::ShopCheckout);

    $test = Livewire::test(ListStaticPageSeoPages::class)->assertOk();
    foreach ($pages as $page) {
        $test->assertSee($page->label())->assertSee($page->path());
    }

    $template = app(SettingsRepository::class)->all()->seo;
    $test->assertSee($template->title('عنوان سفارشی چرخه برای آزمون مدیر سئو'))
        ->assertSee('سفارشی')
        ->assertSee((string) __('home.seo.title'))
        ->assertTableActionVisible('reset', 'stage-cycle')
        ->assertTableActionHidden('reset', 'home')
        ->searchTable('چرخه')
        ->assertSee('/cycle')
        ->assertDontSee('/about');
});

it('shows the same default title and description the live page renders', function (): void {
    $rows = app(ListStaticPageSeo::class)->handle();

    foreach ($rows as $row) {
        $head = renderedHead((string) $this->get($row->page->path())->assertOk()->getContent());

        expect($row->titleOverridden)->toBeFalse()
            ->and($head['title'])->toBe($row->title, "title of {$row->page->path()}")
            ->and($head['description'])->toBe($row->description, "description of {$row->page->path()}");
    }
});

it('changes the live <title> of /cycle on the next request, through the full-page cache', function (): void {
    $this->get('/cycle')->assertOk();
    $before = renderedHead((string) $this->get('/cycle')->assertOk()->assertHeader('X-Page-Cache', 'HIT')->getContent());

    $manager = seoAdmin();
    $this->actingAs($manager);

    Livewire::test(EditStaticPageSeo::class, ['page' => 'stage-cycle'])
        ->assertOk()
        ->assertSee('data-seo-serp-preview', false)
        ->assertSee('data-og-telegram', false)
        ->assertSee('data-og-whatsapp', false)
        ->assertSee($before['title'])
        ->fillForm([
            'seoMeta.title' => 'پیگیری پریود با پیش‌بینی بازه‌ای',
            'seoMeta.description' => 'توضیح تازه صفحه چرخه که مدیر سئو در پنل نوشته است تا در نتایج جست‌وجو دیده شود و طولش کافی باشد.',
            'seoMeta.focus_keyword' => 'پیگیری پریود',
            'seoMeta.robots_follow' => false,
        ])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertNotified();

    $meta = SeoMeta::query()->where('route_name', 'stage.cycle')->sole();
    expect($meta->seoable_type)->toBeNull()
        ->and($meta->focus_keyword)->toBe('پیگیری پریود')
        ->and($meta->robots)->toStartWith('index,nofollow')
        ->and(Activity::query()->where('log_name', 'seo')->where('description', 'seo.static_page.saved')->exists())->toBeTrue();

    Auth::forgetGuards();
    $html = (string) $this->get('/cycle')->assertOk()->getContent();
    $expected = app(SettingsRepository::class)->all()->seo->title('پیگیری پریود با پیش‌بینی بازه‌ای');
    expect(renderedHead($html)['title'])->toBe($expected)
        ->and(renderedHead($html)['description'])->toStartWith('توضیح تازه صفحه چرخه');

    // The admin list reflects the override.
    $row = app(ListStaticPageSeo::class)->find(StaticPage::Cycle);
    expect($row->title)->toBe($expected)->and($row->titleOverridden)->toBeTrue()->and($row->indexable)->toBeTrue();
})->skip(fn (): bool => ! (bool) config('pagecache.enabled', true), 'page cache disabled');

it('resets a page to its defaults from the editor and the list', function (): void {
    SeoMeta::query()->create(['route_name' => 'about', 'title' => 'درباره ریتمی، عنوان سفارشی مدیر سئو', 'robots' => 'noindex,follow']);
    SeoMeta::query()->create(['route_name' => 'faq', 'title' => 'سؤال‌ها، عنوان سفارشی مدیر سئو']);

    expect(app(ListStaticPageSeo::class)->find(StaticPage::About)->indexable)->toBeFalse();

    $this->actingAs(seoAdmin());
    Livewire::test(EditStaticPageSeo::class, ['page' => 'about'])
        ->assertFormSet(['seoMeta.title' => 'درباره ریتمی، عنوان سفارشی مدیر سئو', 'seoMeta.robots_index' => false])
        ->callAction('reset')
        ->assertNotified()
        ->assertFormSet(['seoMeta.title' => null]);

    Livewire::test(ListStaticPageSeoPages::class)
        ->callAction(TestAction::make('reset')->table('faq'));

    expect(SeoMeta::query()->whereIn('route_name', ['about', 'faq'])->exists())->toBeFalse();

    Auth::forgetGuards();
    expect(renderedHead((string) $this->get('/about')->assertOk()->getContent())['title'])->toBe((string) __('about.seo.title'));
});

it('measures complete default titles without the site template', function (): void {
    $this->actingAs(seoAdmin());
    $home = (string) __('home.seo.title');

    Livewire::test(EditStaticPageSeo::class, ['page' => 'home'])
        ->assertOk()
        ->assertSee($home)
        ->assertDontSee(app(SettingsRepository::class)->all()->seo->title($home));
});
