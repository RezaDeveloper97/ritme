<?php

declare(strict_types=1);

use App\Domain\Faq\Actions\InvalidateFaqCache;
use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\Models\FaqGroup;
use App\Domain\Faq\Models\FaqItem;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Faq\FaqGroupResource;
use App\Filament\Resources\Faq\ItemsRelationManager;
use App\Filament\Resources\Faq\Pages\EditFaqGroup;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\FaqSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\DB;
use Livewire\Livewire;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.test']);
});

/**
 * @return list<array<string, mixed>>
 */
function faqNodes(string $html): array
{
    preg_match('~<script type="application/ld\+json">(.*?)</script>~s', $html, $m);
    $graph = json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR)['@graph'] ?? [];

    return array_values(array_filter($graph, static fn (array $node): bool => $node['@type'] === 'FAQPage'));
}

/**
 * @return list<string>
 */
function faqQuestions(string $html): array
{
    $nodes = faqNodes($html);

    return array_map(static fn (array $q): string => $q['name'], $nodes[0]['mainEntity'] ?? []);
}

it('seeds every design FAQ: five /faq categories, contextual groups and six stage groups of three', function (): void {
    $this->seed(FaqSeeder::class);

    expect(FaqGroup::query()->count())->toBe(15)
        ->and(FaqGroup::query()->where('is_listed', true)->orderBy('sort_order')->pluck('title')->all())
        ->toBe(['شروع کار', 'دقت و سلامت', 'حریم خصوصی', 'پرداخت', 'خدمات و فروشگاه']);

    foreach (FaqSeeder::STAGES as $stage) {
        expect(FaqItem::query()->whereRelation('group', 'slug', "stage-{$stage}")->count())->toBe(3);
    }
    expect(FaqItem::query()->whereRelation('group', 'is_listed', true)->count())->toBe(12);

    // Idempotent and never overwrites admin edits.
    FaqItem::query()->first()?->update(['question' => 'ویرایش ادمین؟']);
    $this->seed(FaqSeeder::class);
    expect(FaqItem::query()->count())->toBe(45)
        ->and(FaqItem::query()->where('question', 'ویرایش ادمین؟')->exists())->toBeTrue();
});

it('renders /faq grouped by category with anchor nav, a real search box and one FAQPage of the visible items', function (): void {
    $this->seed(FaqSeeder::class);
    FaqItem::query()->where('question', 'چطور اشتراک را لغو کنم؟')->update(['is_published' => false]);
    app(InvalidateFaqCache::class)();

    $html = $this->get('/faq')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('جواب سؤال‌های رایج')
        ->and($html)->toContain('<title>سؤالات متداول — جواب سؤال‌های رایج درباره ریتمی</title>')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->toContain('href="#getting-started" aria-current="true"')
        ->and($html)->toContain('id="getting-started"')
        ->and($html)->toContain('href="#services-shop"')
        ->and($html)->toContain('type="search"')
        ->and($html)->toContain('data-module="faq-filter"')
        ->and(substr_count($html, '<details'))->toBe(11)
        ->and($html)->not->toContain('چطور اشتراک را لغو کنم؟')
        ->and($html)->not->toContain('قبل از نصب') // contextual groups are not listed
        ->and(substr_count($html, 'id="download"'))->toBe(1);

    expect(faqNodes($html))->toHaveCount(1);
    $questions = faqQuestions($html);
    expect($questions)->toHaveCount(11)
        ->and($questions)->toContain('ریتمی چیست؟')
        ->and($questions)->not->toContain('چطور اشتراک را لغو کنم؟');
});

it('renders /faq without categories or FAQPage when nothing is seeded', function (): void {
    $html = $this->get('/faq')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->not->toContain('<details')
        ->and(faqNodes($html))->toBe([]);
});

it('updates /faq after an admin edit (faq + pages namespaces bumped)', function (): void {
    $this->seed(FaqSeeder::class);
    $this->get('/faq')->assertOk();
    $this->get('/faq')->assertOk()->assertHeader('X-Page-Cache', 'HIT');

    FaqItem::query()->where('question', 'ریتمی چیست؟')->firstOrFail()->update(['question' => 'ریتمی دقیقاً چه می‌کند؟']);

    $html = $this->get('/faq')->assertOk()->getContent();
    expect($html)->toContain('ریتمی دقیقاً چه می‌کند؟')
        ->and(faqQuestions($html))->toContain('ریتمی دقیقاً چه می‌کند؟');
})->skip(fn (): bool => ! (bool) config('pagecache.enabled', true), 'page cache disabled');

it('serves FAQ lookups from the cache once warm', function (): void {
    $this->seed(FaqSeeder::class);
    $repo = app(FaqRepository::class);
    $repo->listed();

    DB::enableQueryLog();
    DB::flushQueryLog();
    expect($repo->group('home')?->items)->toHaveCount(4)
        ->and($repo->group('nope'))->toBeNull()
        ->and($repo->listed())->toHaveCount(5)
        ->and(DB::getQueryLog())->toBe([]);
});

it('sanitises answers on save', function (): void {
    $group = FaqGroup::query()->create(['title' => 'آزمایش', 'slug' => 'Test Group']);
    $item = FaqItem::query()->create(['faq_group_id' => $group->id, 'question' => 'س؟', 'answer' => '<p onclick="x()">جواب <a href="javascript:alert(1)">بد</a> <a href="https://ritme.ir">خوب</a></p><script>alert(1)</script><img src="https://evil.test/a.png">']);
    $plain = FaqItem::query()->create(['faq_group_id' => $group->id, 'question' => 'س۲؟', 'answer' => 'متن ساده & ساده']);

    expect($group->fresh()?->slug)->toBe('test-group')
        ->and($item->fresh()?->answer)->not->toContain('script')->not->toContain('onclick')->not->toContain('javascript')->not->toContain('<img')
        ->toContain('href="https://ritme.ir"')
        ->and($plain->fresh()?->answer)->toBe('<p>متن ساده &amp; ساده</p>');
});

it('renders the home FAQ «قبل از نصب» from group home with its FAQPage, and nothing when unseeded', function (): void {
    $empty = $this->get('/')->assertOk()->getContent();
    expect($empty)->not->toContain('<details')->and(faqNodes($empty))->toBe([]);

    $this->seed(FaqSeeder::class);
    $html = $this->get('/')->assertOk()->getContent();

    expect($html)->toContain('قبل از نصب')
        ->and($html)->toContain('aria-labelledby="home-faq-title"')
        ->and(substr_count($html, '<details'))->toBe(4)
        ->and(substr_count($html, '<details open'))->toBe(1)
        ->and(faqQuestions($html))->toBe(['ریتمی رایگان است؟', 'ریتمی جای پزشک را می‌گیرد؟', 'داده‌هایم کجا می‌رود؟', 'با تغییر مرحله زندگی، داده قبلی‌ام چه می‌شود؟']);
});

it('keeps the stage lang FAQ as fallback while stage-<slug> is not seeded', function (): void {
    $html = $this->get('/cycle')->assertOk()->getContent();

    expect(faqQuestions($html))->toBe([
        'اگر چرخه‌ام نامنظم باشد، ریتمی به دردم می‌خورد؟',
        'پیش‌بینی ریتمی برای پیشگیری از بارداری کافی است؟',
        'می‌توانم گزارش را به پزشکم نشان بدهم؟',
    ])->and(substr_count($html, '<details'))->toBe(3);
});

it('wires stage pages to their stage-<slug> group (same copy as the lang file, admin edits show up)', function (): void {
    $this->seed(FaqSeeder::class);
    $seeded = $this->get('/cycle')->assertOk()->getContent();
    expect(faqQuestions($seeded))->toBe([
        'اگر چرخه‌ام نامنظم باشد، ریتمی به دردم می‌خورد؟',
        'پیش‌بینی ریتمی برای پیشگیری از بارداری کافی است؟',
        'می‌توانم گزارش را به پزشکم نشان بدهم؟',
    ])->and(substr_count($seeded, '<details'))->toBe(3)
        ->and($seeded)->toContain('data-faq-group="stage-cycle"');

    // (StagePageController keeps its injected SchemaGraph across requests in tests: assert the HTML from here on.)
    FaqItem::query()->whereRelation('group', 'slug', 'stage-cycle')->orderBy('sort_order')->firstOrFail()
        ->update(['question' => 'سؤال تازه چرخه؟']);
    expect($this->get('/cycle')->assertOk()->getContent())->toContain('سؤال تازه چرخه؟');

    // A seeded group with nothing published hides the block (no fallback to lang).
    FaqItem::query()->whereRelation('group', 'slug', 'stage-cycle')->get()->each->update(['is_published' => false]);
    expect($this->get('/cycle')->assertOk()->getContent())->not->toContain('<details');
});

it('renders the grid variant as cards with h3 questions', function (): void {
    $this->seed(FaqSeeder::class);
    $plus = app(FaqRepository::class)->group('plus');
    $html = Blade::render('<x-faq :faq="$faq" variant="grid" eyebrow="قبل از خرید" :title="$faq->title" bg="surface"/>', ['faq' => $plus]);

    expect(substr_count($html, '<h3'))->toBe(4)
        ->and($html)->toContain('grid-cols-2')
        ->and($html)->not->toContain('<details')
        ->and($html)->toMatch('~<h2\\s+id="faq-plus"~');
});

it('lets editors manage FAQ groups and items in the admin and denies other roles', function (): void {
    $this->seed([AdminRolesSeeder::class, FaqSeeder::class]);
    Filament::setCurrentPanel('admin');
    $group = FaqGroup::query()->where('slug', 'getting-started')->firstOrFail();

    $this->get(FaqGroupResource::getUrl('index'))->assertRedirect('/admin/login');

    $editor = User::factory()->create();
    $editor->assignRole(AdminRole::Editor->value);
    $this->actingAs($editor)->get(FaqGroupResource::getUrl('index'))->assertOk()->assertSee('شروع کار');
    $this->actingAs($editor)->get(FaqGroupResource::getUrl('edit', ['record' => $group]))->assertOk()->assertSee('شروع کار');

    foreach ([AdminRole::Support, AdminRole::ShopManager, AdminRole::SeoManager] as $role) {
        $user = User::factory()->create();
        $user->assignRole($role->value);
        $this->actingAs($user)->get(FaqGroupResource::getUrl('index'))->assertForbidden();
    }

    $this->actingAs($editor);
    $item = $group->items()->orderBy('sort_order')->firstOrFail();
    Livewire::test(ItemsRelationManager::class, ['ownerRecord' => $group, 'pageClass' => EditFaqGroup::class])
        ->assertOk()
        ->assertCanSeeTableRecords($group->items()->get())
        ->call('updateTableColumnState', 'is_published', (string) $item->getKey(), false);

    expect($item->fresh()?->is_published)->toBeFalse();
    expect($this->get('/faq')->getContent())->not->toContain('ریتمی چیست؟');
});
