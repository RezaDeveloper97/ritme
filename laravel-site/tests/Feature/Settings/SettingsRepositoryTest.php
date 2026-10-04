<?php

declare(strict_types=1);

use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\AppLinksSettings;
use App\Domain\Settings\Data\LayoutSettings;
use App\Domain\Settings\Data\SeoDefaults;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Exceptions\UnknownSettingException;
use App\Domain\Settings\Models\Setting;
use App\Domain\Settings\Repositories\CachedSettingsRepository;
use App\Domain\Settings\View\LayoutSettingsComposer;
use App\Support\Cache\NamespaceVersions;
use Database\Seeders\SettingsSeeder;
use Illuminate\Contracts\View\View;
use Illuminate\Support\Facades\DB;

function settingsQueryCount(Closure $callback): int
{
    DB::flushQueryLog();
    DB::enableQueryLog();
    $callback();
    $count = count(array_filter(DB::getQueryLog(), static fn (array $q): bool => str_contains((string) $q['query'], '"settings"')));
    DB::disableQueryLog();

    return $count;
}

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    $this->settings = app(SettingsRepository::class);
});

it('resolves the contract to the cached decorator', function (): void {
    expect($this->settings)->toBeInstanceOf(CachedSettingsRepository::class);
});

it('reads typed groups seeded from the design', function (): void {
    $all = $this->settings->all();

    expect($all->general->siteName)->toBe('ریتمی')
        ->and($all->contact->supportEmail)->toBe('[ایمیل پشتیبانی]')
        ->and($all->seo->titleTemplate)->toBe('%s — ریتمی')
        ->and($all->seo->title('درباره ما'))->toBe('درباره ما — ریتمی')
        ->and($all->seo->title(''))->toBe($all->seo->defaultTitle)
        ->and($all->organization->contactPoint->contactType)->toBe('customer support')
        ->and($all->legal->enamadAllowedTags)->toBe(['a', 'img'])
        ->and($all->pwa->themeColor)->toBe('#17112B')
        ->and($this->settings->group(SettingGroup::AppLinks))->toBeInstanceOf(AppLinksSettings::class)
        ->and($this->settings->get(SettingGroup::General, 'emergency_number'))->toBe('115');
});

it('falls back to typed defaults when nothing is stored', function (): void {
    Setting::query()->delete();

    $seo = $this->settings->group(SettingGroup::Seo);

    expect($seo)->toBeInstanceOf(SeoDefaults::class)
        ->and($seo->toArray()['title_template'])->toBe('%s — ریتمی')
        ->and($this->settings->all()->appLinks->bazaar)->toBeNull();
});

it('runs one query on a cold read and none once cached', function (): void {
    expect(settingsQueryCount(function (): void {
        $this->settings->all();
        $this->settings->group(SettingGroup::Contact);
        $this->settings->get(SettingGroup::Seo, 'separator');
    }))->toBe(1);

    expect(settingsQueryCount(function (): void {
        $this->settings->all();
        $this->settings->get(SettingGroup::Social, 'instagram');
    }))->toBe(0);
});

it('bumps the cache on update so the next read sees the new value', function (): void {
    $versions = app(NamespaceVersions::class);
    $this->settings->all(); // warm
    $before = ['settings' => $versions->version('settings'), 'seo' => $versions->version('seo'), 'pages' => $versions->version('pages')];

    $result = app(UpdateSettings::class)->handle(SettingGroup::AppLinks, [
        'bazaar' => 'https://cafebazaar.ir/app/ir.ritmeapp.ritme',
        'myket' => '  ',
    ]);

    expect($versions->version('settings'))->toBeGreaterThan($before['settings'])
        ->and($versions->version('seo'))->toBeGreaterThan($before['seo'])
        ->and($versions->version('pages'))->toBeGreaterThan($before['pages'])
        ->and($result)->toBeInstanceOf(AppLinksSettings::class)
        ->and($this->settings->all()->appLinks->bazaar)->toBe('https://cafebazaar.ir/app/ir.ritmeapp.ritme')
        ->and($this->settings->all()->appLinks->myket)->toBeNull();
});

it('invalidates when a setting row changes outside the repository', function (): void {
    $this->settings->all();

    Setting::query()->where('group', 'general')->where('key', 'tagline')->firstOrFail()->update(['value' => 'تازه']);

    expect($this->settings->all()->general->tagline)->toBe('تازه');
});

it('coerces values through the group DTO on write', function (): void {
    $this->settings->put(SettingGroup::Seo, ['default_og_media_id' => '42', 'verification' => ['google' => ' abc ', 'bad' => ['x']]]);

    expect($this->settings->all()->seo->defaultOgMediaId)->toBe(42)
        ->and($this->settings->all()->seo->verification)->toBe(['google' => 'abc'])
        ->and(Setting::query()->where('group', 'seo')->where('key', 'default_og_media_id')->value('value'))->toBe(42);
});

it('throws on an unknown key or group', function (Closure $call): void {
    expect(fn () => $call($this->settings))->toThrow(UnknownSettingException::class);
})->with([
    'get unknown key' => [fn (SettingsRepository $s) => $s->get(SettingGroup::Contact, 'fax')],
    'put unknown key' => [fn (SettingsRepository $s) => $s->put(SettingGroup::Social, ['myspace' => 'x'])],
    'unknown group' => [fn () => SettingGroup::fromName('nope')],
]);

it('does not write anything when one key of a batch is unknown', function (): void {
    expect(fn () => $this->settings->put(SettingGroup::Social, ['instagram' => 'https://instagram.com/ritme', 'myspace' => 'x']))
        ->toThrow(UnknownSettingException::class);

    expect($this->settings->all()->social->instagram)->toBeNull();
});

it('keeps admin edits when the seeder runs again', function (): void {
    $this->settings->put(SettingGroup::Contact, ['phone' => '021-0000']);

    $this->seed(SettingsSeeder::class);

    expect($this->settings->all()->contact->phone)->toBe('021-0000');
});

it('shares only the layout groups with layout views', function (): void {
    $view = Mockery::mock(View::class);
    $view->shouldReceive('with')->once()->with('siteSettings', Mockery::type(LayoutSettings::class))->andReturnSelf();

    app(LayoutSettingsComposer::class)->compose($view);

    expect(array_keys(get_class_vars(LayoutSettings::class)))->toBe(['general', 'contact', 'social', 'appLinks', 'legal']);
});
