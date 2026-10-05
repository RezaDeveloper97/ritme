<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Category as BlogCategory;
use App\Domain\Blog\Models\Post;
use App\Domain\Directory\Models\City;
use App\Domain\Seo\Redirects\Actions\ExportRedirects;
use App\Domain\Seo\Redirects\Actions\FlushRedirectStats;
use App\Domain\Seo\Redirects\Actions\ImportRedirects;
use App\Domain\Seo\Redirects\Actions\PurgeNotFoundLogs;
use App\Domain\Seo\Redirects\Actions\SaveRedirect;
use App\Domain\Seo\Redirects\Contracts\RedirectMapRepository;
use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Support\HitBuffer;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Seo\NotFoundLogs\NotFoundLogResource;
use App\Filament\Resources\Seo\NotFoundLogs\Pages\ListNotFoundLogs;
use App\Filament\Resources\Seo\Redirects\Pages\CreateRedirect;
use App\Filament\Resources\Seo\Redirects\RedirectResource;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Testing\TestAction;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\DB;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;

const REDIRECTS_HUMAN_UA = 'Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/128.0 Mobile Safari/537.36';

/**
 * @param  array<string, mixed>  $data
 */
function saveRedirect(array $data, ?Redirect $redirect = null): Redirect
{
    return app(SaveRedirect::class)->handle($redirect ?? new Redirect, $data);
}

function redirectsAdmin(?AdminRole $role = AdminRole::SeoManager): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class]);
});

it('redirects a 404 path in one hop, keeping the query string and encoding Persian paths', function (): void {
    saveRedirect(['from_path' => '/old-page/', 'to_url' => '/blog', 'code' => 301]);
    saveRedirect(['from_path' => '/مطلب-قدیمی', 'to_url' => '/راهنما-جدید', 'code' => 302]);

    $this->get('/old-page?utm=x')->assertStatus(301)->assertRedirect(url('/blog').'?utm=x');
    // Trailing slash: straight to the target, not via CanonicalizeUrl's slash redirect first.
    $this->get('/old-page/')->assertStatus(301)->assertRedirect(url('/blog'));
    // Case-insensitive source.
    $this->get('/OLD-PAGE')->assertStatus(301)->assertRedirect(url('/blog'));

    $this->get('/'.rawurlencode('مطلب-قدیمی'))->assertStatus(302)
        ->assertHeader('Location', url('/'.rawurlencode('راهنما-جدید')));
});

it('never shadows a live page, but wins over the built-in legacy map', function (): void {
    saveRedirect(['from_path' => '/cycle', 'to_url' => '/blog']);
    $this->get('/cycle')->assertOk();

    saveRedirect(['from_path' => '/cycle.html', 'to_url' => '/ttc']);
    $this->get('/cycle.html')->assertStatus(301)->assertRedirect(url('/ttc'));
    $this->get('/pregnancy.html')->assertStatus(301)->assertRedirect(url('/pregnancy')); // LegacyUrlMap untouched
});

it('answers 410 for gone URLs and supports regex sources with captures', function (): void {
    saveRedirect(['from_path' => '/discontinued', 'code' => 410, 'to_url' => '/ignored']);
    saveRedirect(['from_path' => '/category/(.+)', 'to_url' => '/blog/category/$1', 'is_regex' => true]);

    expect(Redirect::query()->where('code', 410)->value('to_url'))->toBeNull();
    $this->get('/discontinued')->assertStatus(410);
    $this->get('/category/cycle')->assertStatus(301)->assertRedirect(url('/blog/category/cycle'));
    $this->get('/category')->assertNotFound();
});

it('rejects invalid regex patterns and regex loops', function (array $data): void {
    expect(fn () => saveRedirect($data))->toThrow(InvalidRedirect::class);
})->with([
    'broken pattern' => [['from_path' => '/a/(', 'to_url' => '/b', 'is_regex' => true]],
    'self-matching target' => [['from_path' => '/news/.*', 'to_url' => '/news/all', 'is_regex' => true]],
]);

it('collapses chains on save in both directions', function (): void {
    $a = saveRedirect(['from_path' => '/a', 'to_url' => '/b']);
    saveRedirect(['from_path' => '/b', 'to_url' => '/c']);   // A→B + B→C ⇒ A→C

    expect($a->refresh()->to_url)->toBe('/c');

    $x = saveRedirect(['from_path' => '/x', 'to_url' => '/a']);  // X→A, A→C ⇒ X→C
    expect($x->to_url)->toBe('/c');

    saveRedirect(['from_path' => '/c', 'code' => 410]);       // the end of the chain is gone ⇒ everything is 410
    expect(Redirect::query()->whereIn('from_path', ['/a', '/b', '/x'])->pluck('code')->map->value->unique()->all())->toBe([410]);

    $this->get('/x')->assertStatus(410);
});

it('rejects loops, self redirects and duplicate sources', function (): void {
    saveRedirect(['from_path' => '/one', 'to_url' => '/two']);

    expect(fn () => saveRedirect(['from_path' => '/two', 'to_url' => '/one']))->toThrow(InvalidRedirect::class)
        ->and(fn () => saveRedirect(['from_path' => '/self', 'to_url' => '/self/']))->toThrow(InvalidRedirect::class)
        ->and(fn () => saveRedirect(['from_path' => '/ONE/', 'to_url' => '/three']))->toThrow(InvalidRedirect::class)
        ->and(fn () => saveRedirect(['from_path' => '/', 'to_url' => '/three']))->toThrow(InvalidRedirect::class)
        ->and(fn () => saveRedirect(['from_path' => '/z', 'to_url' => 'javascript:alert(1)']))->toThrow(InvalidRedirect::class)
        ->and(Redirect::query()->count())->toBe(1);
});

it('treats absolute URLs on the own host as local and keeps external targets', function (): void {
    $local = saveRedirect(['from_path' => url('/old-local'), 'to_url' => url('/blog')]);
    $external = saveRedirect(['from_path' => '/partner', 'to_url' => 'https://example.org/page?ref=ritme']);

    expect($local->from_path)->toBe('/old-local')->and($local->to_url)->toBe('/blog')
        ->and($external->to_url)->toBe('https://example.org/page?ref=ritme');

    $this->get('/partner?x=1')->assertRedirect('https://example.org/page?ref=ritme');
});

it('reads the redirect map from one cache entry (no query on a warm cache) and refreshes it on change', function (): void {
    saveRedirect(['from_path' => '/cached', 'to_url' => '/blog']);
    $this->get('/cached')->assertRedirect(url('/blog'));

    DB::enableQueryLog();
    $this->get('/cached')->assertRedirect(url('/blog'));
    expect(DB::getQueryLog())->toBe([]);
    DB::disableQueryLog();

    Redirect::query()->firstOrFail()->delete();
    $this->get('/cached')->assertNotFound();
});

it('counts redirect hits in a buffer and flushes them in one batch', function (): void {
    $redirect = saveRedirect(['from_path' => '/counted', 'to_url' => '/blog']);

    foreach (range(1, 3) as $i) {
        $this->get('/counted');
    }
    expect($redirect->refresh()->hits)->toBe(0);

    Artisan::call('seo:flush-redirect-stats');

    expect($redirect->refresh()->hits)->toBe(3)->and($redirect->last_hit_at)->not->toBeNull();
});

it('creates a 301 when a blog category slug changes, without loops when it changes back', function (): void {
    $category = BlogCategory::factory()->create(['slug' => 'old-cat']);
    $category->update(['slug' => 'new-cat']);

    $row = Redirect::query()->sole();
    expect($row->from_path)->toBe('/blog/category/old-cat')->and($row->to_url)->toBe('/blog/category/new-cat')
        ->and($row->is_auto)->toBeTrue();
    $this->get('/blog/category/old-cat')->assertStatus(301)->assertRedirect(url('/blog/category/new-cat'));

    $category->update(['slug' => 'third-cat']);   // old-cat → third-cat, new-cat → third-cat (chain collapsed)
    expect(Redirect::query()->pluck('to_url')->unique()->all())->toBe(['/blog/category/third-cat']);

    $category->update(['slug' => 'old-cat']);     // back to the first slug: no redirect away from a live URL
    expect(Redirect::query()->where('from_path', '/blog/category/old-cat')->exists())->toBeFalse()
        ->and(Redirect::query()->pluck('to_url')->unique()->all())->toBe(['/blog/category/old-cat']);
});

it('creates a regex redirect for a renamed directory city that covers its landing URLs', function (): void {
    $city = City::factory()->create(['slug' => 'old-city']);
    $city->update(['slug' => 'new-city']);

    expect(Redirect::query()->sole()->is_regex)->toBeTrue();
    $this->get('/directory/old-city')->assertStatus(301)->assertRedirect(url('/directory/new-city'));
    $this->get('/directory/old-city/clinic')->assertStatus(301)->assertRedirect(url('/directory/new-city/clinic'));
});

it('leaves post slug changes to the post slug history (no redundant redirect row)', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'first-slug']);
    $post->update(['slug' => 'second-slug']);

    expect(Redirect::query()->count())->toBe(0);
    $this->get('/blog/first-slug')->assertStatus(301)->assertRedirect(url('/blog/second-slug'));
});

it('aggregates 404s cheaply: deduped, buffered, without bots, assets or personal data', function (): void {
    foreach (range(1, 5) as $i) {
        $this->withHeaders(['User-Agent' => REDIRECTS_HUMAN_UA, 'Referer' => 'https://google.com/search?q=private'])->get('/missing-page?id='.$i)->assertNotFound();
    }
    $this->withHeaders(['User-Agent' => 'Googlebot/2.1 (+http://www.google.com/bot.html)'])->get('/bot-only')->assertNotFound();
    $this->withHeaders(['User-Agent' => REDIRECTS_HUMAN_UA])->get('/old/image.png')->assertNotFound();
    $this->withHeaders(['User-Agent' => REDIRECTS_HUMAN_UA])->get('/wp-login.php')->assertNotFound();

    expect(NotFoundLog::query()->count())->toBe(0); // nothing written during requests

    app(FlushRedirectStats::class)->handle();

    $row = NotFoundLog::query()->sole();
    expect($row->path)->toBe('/missing-page')
        ->and($row->hits)->toBe(5)
        ->and($row->referer)->toBe('https://google.com/search')
        ->and($row->agent->value)->toBe('human')
        ->and(array_keys($row->getAttributes()))->not->toContain('ip');

    $this->withHeaders(['User-Agent' => REDIRECTS_HUMAN_UA])->get('/missing-page');
    app(FlushRedirectStats::class)->handle();
    expect($row->refresh()->hits)->toBe(6)->and(NotFoundLog::query()->count())->toBe(1);
});

it('caps the 404 buffer and table so a scanner cannot flood them', function (): void {
    $buffer = app(HitBuffer::class);
    foreach (range(1, HitBuffer::MAX_PENDING + 20) as $i) {
        $buffer->hit('not-found', sha1((string) $i), ['path' => '/p'.$i, 'referer' => null, 'agent' => 'human']);
    }

    expect(count($buffer->drain('not-found')))->toBe(HitBuffer::MAX_PENDING);
});

it('does not query the database for an unmatched 404 on a warm cache', function (): void {
    $this->withHeaders(['User-Agent' => REDIRECTS_HUMAN_UA])->get('/warm-404')->assertNotFound();

    DB::enableQueryLog();
    $this->withHeaders(['User-Agent' => REDIRECTS_HUMAN_UA])->get('/another-404')->assertNotFound();
    expect(DB::getQueryLog())->toBe([]);
});

it('purges 404 rows not seen for 90 days', function (): void {
    NotFoundLog::query()->create(['path' => '/old', 'path_hash' => sha1('/old'), 'hits' => 1, 'last_seen_at' => now()->subDays(91)]);
    NotFoundLog::query()->create(['path' => '/recent', 'path_hash' => sha1('/recent'), 'hits' => 1, 'last_seen_at' => now()->subDays(10)]);

    expect(app(PurgeNotFoundLogs::class)->handle())->toBe(1)
        ->and(NotFoundLog::query()->pluck('path')->all())->toBe(['/recent']);
});

it('imports WordPress redirects from CSV and exports them for a round trip', function (): void {
    $csv = tempnam(sys_get_temp_dir(), 'rd');
    file_put_contents($csv, "\xEF\xBB\xBFfrom,to,code,regex,note\n"
        ."/2019/05/old-post/,/blog/new-post,301,,wp\n"
        ."/tag/(.+),/blog/tag/\$1,301,1,\n"
        ."/removed,,,,\n"
        ."/loop-a,/loop-a,301,,\n"
        ."/2019/05/old-post,/blog/newer-post,302,,update\n");

    $result = app(ImportRedirects::class)->handle($csv);
    unlink($csv);

    expect($result->created)->toBe(3)->and($result->updated)->toBe(1)->and($result->skipped)->toBe(1)
        ->and($result->errors)->toHaveCount(1)
        ->and(Redirect::query()->where('from_path', '/2019/05/old-post')->sole()->to_url)->toBe('/blog/newer-post')
        ->and(Redirect::query()->where('from_path', '/removed')->sole()->code)->toBe(RedirectCode::Gone)
        ->and(Activity::query()->where('log_name', 'seo')->where('event', 'imported')->exists())->toBeTrue();

    $out = fopen('php://memory', 'w+b');
    expect(app(ExportRedirects::class)->handle($out))->toBe(3);
    rewind($out);
    $exported = (string) stream_get_contents($out);
    expect($exported)->toContain('/tag/(.+),/blog/tag/$1,301,1')->toContain('/removed,,410,0');

    // Re-importing the export changes nothing but counts as updates.
    $again = tempnam(sys_get_temp_dir(), 'rd');
    file_put_contents($again, $exported);
    expect(app(ImportRedirects::class)->handle($again)->updated)->toBe(3);
    unlink($again);
});

it('logs redirect changes to the activity log', function (): void {
    $redirect = saveRedirect(['from_path' => '/logged', 'to_url' => '/blog']);

    expect(Activity::query()->where('log_name', 'seo')->where('event', 'created')->where('subject_id', $redirect->id)->exists())->toBeTrue();
});

describe('admin', function (): void {
    beforeEach(function (): void {
        $this->seed([AdminRolesSeeder::class]);
        Filament::setCurrentPanel('admin');
    });

    it('lets SEO managers in and denies every other role', function (): void {
        $urls = [RedirectResource::getUrl('index'), RedirectResource::getUrl('create'), RedirectResource::getUrl('export'), NotFoundLogResource::getUrl('index')];

        foreach ($urls as $url) {
            $this->actingAs(redirectsAdmin())->get($url)->assertOk();
        }
        foreach ([AdminRole::Editor, AdminRole::Support, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
            foreach ($urls as $url) {
                $this->actingAs(redirectsAdmin($role))->get($url)->assertForbidden();
            }
        }

        expect(redirectsAdmin(AdminRole::SuperAdmin)->can('create', Redirect::class))->toBeTrue()
            ->and(redirectsAdmin()->can('update', new NotFoundLog))->toBeFalse();
    });

    it('creates redirects through the form and shows loop errors on the field', function (): void {
        $this->actingAs(redirectsAdmin());
        saveRedirect(['from_path' => '/p', 'to_url' => '/q']);

        Livewire::test(CreateRedirect::class)
            ->fillForm(['from_path' => '/q', 'code' => 301, 'to_url' => '/p', 'is_regex' => false])
            ->call('create')
            ->assertHasFormErrors(['to_url']);

        Livewire::test(CreateRedirect::class)
            ->fillForm(['from_path' => '/r', 'code' => 301, 'to_url' => '/p', 'is_regex' => false])
            ->call('create')
            ->assertHasNoFormErrors();

        expect(Redirect::query()->where('from_path', '/r')->sole()->to_url)->toBe('/q');
    });

    it('turns a logged 404 into a redirect', function (): void {
        $this->actingAs(redirectsAdmin());
        $log = NotFoundLog::query()->create(['path' => '/typo-page', 'path_hash' => sha1('/typo-page'), 'hits' => 4, 'last_seen_at' => now()]);

        Livewire::test(ListNotFoundLogs::class)
            ->callAction(TestAction::make('createRedirect')->table($log), ['code' => 301, 'to_url' => '/faq'])
            ->assertHasNoFormErrors();

        expect(NotFoundLog::query()->count())->toBe(0)
            ->and(Redirect::query()->sole()->to_url)->toBe('/faq');
        $this->get('/typo-page')->assertRedirect(url('/faq'));
        expect(app(RedirectMapRepository::class)->map()->isEmpty())->toBeFalse();
    });
});
