<?php

declare(strict_types=1);

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Settings\Data\AppLinksSettings;
use Carbon\CarbonImmutable;
use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\Blade;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.test']);
});

it('renders the /_components kit page as a valid shell without inline styles or external hosts', function (): void {
    $html = $this->get('/_components')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->toContain('id="download"')
        ->and($html)->toContain('aria-label="کد QR لینک دریافت اپ"')
        ->and($html)->toContain('href="tel:115"');

    preg_match_all('/\s(?:src|href|action|srcset)="((?:https?:)?\/\/[^"\/]+)/i', $html, $matches);
    $hosts = array_unique(array_map(static fn (string $url): string => (string) parse_url(str_starts_with($url, '//') ? 'http:'.$url : $url, PHP_URL_HOST), $matches[1]));
    expect(array_values(array_diff($hosts, ['ritme.test', 'localhost', '127.0.0.1'])))->toBe([]);
});

it('keeps the kit components free of inline styles, raw hex and physical left/right utilities', function (): void {
    $files = array_merge(
        glob(resource_path('views/components/{ui,cards,stage}/*.blade.php'), GLOB_BRACE) ?: [],
        glob(resource_path('views/components/ui/form/*.blade.php')) ?: [],
    );
    expect(count($files))->toBeGreaterThan(30);

    foreach ($files as $file) {
        $source = (string) preg_replace('/\{\{--.*?--\}\}/s', '', (string) file_get_contents($file));
        expect($source)->not->toMatch('/\sstyle\s*=/i', $file)
            ->not->toMatch('/(?<![\w&])#[0-9a-f]{6}\b/i', $file)
            ->not->toMatch('/(?<![\w-])(?:pl|pr|ml|mr|left|right)-[\d\[]/', $file)
            ->not->toMatch('/\btext-(?:left|right)\b/', $file);
    }
});

it('renders link-based chip navigation with aria-current instead of JS tabs', function (): void {
    $html = Blade::render('<x-ui.chip-nav label="دسته‌ها" :items="$items"/>', ['items' => [
        ['label' => 'همه', 'href' => '/blog', 'active' => true],
        ['label' => 'بارداری', 'href' => '/blog?c=pregnancy', 'icon' => 'drop'],
    ]]);

    expect($html)->toContain('<nav aria-label="دسته‌ها"')
        ->toMatch('/href="\/blog"\s+aria-current="page"/')
        ->not->toContain('role="tab"')
        ->not->toContain('data-module');
    expect(substr_count($html, 'aria-current'))->toBe(1);
});

it('formats ratings and prices with Persian digits', function (): void {
    $rating = Blade::render('<x-ui.rating :value="4.8" :count="1260"/>');
    expect($rating)->toContain('۴٫۸')->toContain('(۱٬۲۶۰)')->toContain('امتیاز ۴٫۸ از ۵ از ۱٬۲۶۰ نظر');

    $price = Blade::render('<x-ui.price :amount="485000" :compare="1250000"/>');
    expect($price)->toContain('۴۸۵ هزار')->toContain('<s')->toContain('۱٬۲۵۰ هزار')->toContain('قیمت قبلی');

    expect(Blade::render('<x-ui.price :amount="12500" from unit="هر جلسه" size="inline"/>'))
        ->toContain('از <b class="text-ink">۱۲٬۵۰۰</b> تومان · هر جلسه');
});

it('hides empty store links and accepts the AppLinksSettings DTO', function (): void {
    $links = new AppLinksSettings(bazaar: 'https://cafebazaar.ir/app/x', myket: null, googlePlay: '', appStore: 'javascript:alert(1)', webApp: null);
    $html = Blade::render('<x-ui.store-badges :links="$links"/>', ['links' => $links]);

    expect($html)->toContain('کافه‌بازار')->toContain('rel="noopener"')
        ->not->toContain('مایکت')->not->toContain('گوگل‌پلی')->not->toContain('javascript:');
    expect(trim(Blade::render('<x-ui.store-badges :links="null"/>')))->toBe('');
});

it('renders a local, cached SVG QR code and nothing for an empty URL', function (): void {
    $first = Blade::render('<x-ui.qr url="https://ritme.test/app"/>');
    expect($first)->toContain('<svg fill="currentColor"')->toContain('<figcaption')
        ->not->toContain('<?xml')->not->toMatch('/#[0-9a-f]{6}/i');

    expect(Blade::render('<x-ui.qr url="https://ritme.test/app"/>'))->toBe($first);
    expect(trim(Blade::render('<x-ui.qr :url="null"/>')))->toBe('');
});

it('makes the emergency number a tel: link with Persian digits', function (): void {
    $html = Blade::render('<x-ui.alert-emergency number="115">با اورژانس تماس بگیر.</x-ui.alert-emergency>');
    expect($html)->toContain('href="tel:115"')->toContain('>۱۱۵</a>')->toContain('role="note"');
});

it('renders accordions as native details with the first item open', function (): void {
    $html = Blade::render('<x-ui.accordion :items="$items"/>', ['items' => [
        ['question' => 'سؤال ۱', 'answer' => 'جواب ۱'],
        ['question' => 'سؤال ۲', 'answer' => '<b>x</b>'],
    ]]);
    expect(substr_count($html, '<details'))->toBe(2)
        ->and(preg_match_all('/<details\s+open/', $html))->toBe(1)
        ->and($html)->toContain('&lt;b&gt;x&lt;/b&gt;');
});

it('marks the current step in the stepper and timeline', function (): void {
    $stepper = Blade::render('<x-ui.stepper :current="2" :steps="$steps"/>', ['steps' => [['label' => 'الف'], ['label' => 'ب'], ['label' => 'ج']]]);
    expect(substr_count($stepper, 'aria-current="step"'))->toBe(1)->and($stepper)->toContain('مرحله فعلی')->toContain('۳');

    $bar = Blade::render('<x-ui.stepper variant="bar" :current="1" :steps="$steps"/>', ['steps' => [['label' => 'سبد'], ['label' => 'تأیید']]]);
    expect(substr_count($bar, 'aria-current="step"'))->toBe(1);

    $timeline = Blade::render('<x-ui.timeline :items="$items"/>', ['items' => [
        ['title' => 'ثبت', 'state' => 'done'], ['title' => 'بررسی', 'state' => 'current'], ['title' => 'انتشار', 'state' => 'todo'],
    ]]);
    expect($timeline)->toContain('انجام شد')->toContain('در جریان')->toContain('در انتظار');
});

it('keeps cards free of nested interactive elements', function (): void {
    $product = Blade::render('<x-cards.product href="/shop/product/a" title="بادی" :price="485000" :rating="4.7" :reviews="21"/>');
    expect($product)->toStartWith('<article')->toContain('aria-label="افزودن بادی به علاقه‌مندی‌ها"');
    // The only link is the stretched title link; the wishlist button sits outside it.
    expect(substr_count($product, '<a '))->toBe(1)->and($product)->not->toMatch('/<a [^>]*>(?:(?!<\/a>).)*<button/s');

    $place = Blade::render('<x-cards.place href="/directory/place/a" name="آب‌پری" :slots="[\'امروز ۱۷:۰۰\']" verified :price-from="320000"/>');
    expect(substr_count($place, '<a '))->toBe(1)->and($place)->toContain('امروز ۱۷:۰۰')->toContain('مدارک بررسی شد')->toContain('۳۲۰ هزار');
});

it('renders an article card from a PostCardData DTO with the stage fallback cover', function (): void {
    $post = new PostCardData(
        id: 1, title: 'درد پریود', slug: 'period-pain', excerpt: null, category: null, lifeStage: LifeStage::Cycle,
        coverMediaId: null, mobileCoverMediaId: null, readingTime: 5,
        publishedAt: CarbonImmutable::now(), updatedContentAt: CarbonImmutable::now(),
    );
    $html = Blade::render('<x-cards.article :post="$post"/>', ['post' => $post]);

    expect($html)->toContain(route('blog.show', 'period-pain'))
        ->toContain('from-stage-cycle/33')
        ->toContain('text-stage-cycle')
        ->toContain('چرخه و پریود')
        ->toContain('مطالعه ۵ دقیقه')
        ->toContain('<h3');
});

it('renders stage blocks and hides readings when there are no posts', function (): void {
    $tools = Blade::render('<x-stage.tools-block :items="$items"/>', ['items' => [
        ['href' => '/tools', 'icon' => 'calculator', 'color' => 'ttc', 'title' => 'محاسبه', 'where' => 'روی سایت'],
    ]]);
    expect($tools)->toContain('کارهای کوچک، آمادگی بیشتر')->toContain('aria-labelledby="stage-tools-title"')->toContain('id="stage-tools-title"');

    expect(trim(Blade::render('<x-stage.readings :posts="[]"/>')))->toBe('');
});

it('renders the newsletter as a labelled form that is disabled until wired', function (): void {
    $html = Blade::render('<x-ui.newsletter/>');
    expect($html)->toContain('<label for="newsletter-input"')->toContain('disabled')->not->toContain('action=');
});
