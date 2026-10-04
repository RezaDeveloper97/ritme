<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\Data\FaqGroupData;
use App\Domain\Faq\Data\FaqItemData;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Search\Actions\SearchSite;
use App\Domain\Search\Support\SearchTermLog;
use App\Domain\Search\Support\SearchTerms;
use App\Http\Controllers\FeedController;
use App\Http\Controllers\SearchController;
use Carbon\CarbonImmutable;
use Illuminate\Support\Facades\Storage;
use Tests\Feature\Media\MediaFixtures;

function fakeFaq(): void
{
    app()->instance(FaqRepository::class, new class implements FaqRepository
    {
        public function group(string $slug): ?FaqGroupData
        {
            return null;
        }

        public function listed(): array
        {
            return [new FaqGroupData(1, 'privacy', 'حریم خصوصی', true, [
                new FaqItemData(10, 'آيا داده‌هاي من فروخته مي‌شود؟', '<p>خیر، داده‌های تو فروخته نمی‌شود.</p>'),
                new FaqItemData(11, 'اپ رایگان است؟', '<p>بله، نسخه پایه رایگان است.</p>'),
            ])];
        }
    });
}

/**
 * @return array<string, mixed>
 */
function feedSearchGraph(string $html): array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);

    return json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR);
}

// ---- Persian normalisation ---------------------------------------------------------------------------------------

it('normalises Arabic letters, ZWNJ, tatweel, diacritics and digits', function (string $input, string $expected): void {
    expect(SearchTerms::normalize($input))->toBe($expected);
})->with([
    'arabic yeh + kaf' => ['كيك', 'کیک'],
    'alef maksura' => ['موسى', 'موسی'],
    'zwnj' => ["می\u{200C}خواهم", 'میخواهم'],
    'tatweel' => ['بـــارداری', 'بارداری'],
    'harakat' => ['دَرد', 'درد'],
    'persian digits' => ['هفته ۱۲', 'هفته 12'],
    'arabic digits' => ['هفته ١٢', 'هفته 12'],
    'teh marbuta' => ['تغذیة', 'تغذیه'],
    'case + punctuation + spaces' => ['  PMS،   درد!  ', 'pms درد'],
]);

it('tokenises the query: at least two letters, unique, at most five tokens, capped length', function (): void {
    $terms = SearchTerms::fromInput('a درد درد ب پریود یک دو سه چهار پنج');

    expect($terms->tokens)->toBe(['درد', 'پریود', 'یک', 'دو', 'سه'])
        ->and(SearchTerms::fromInput('ا')->isEmpty())->toBeTrue()
        ->and(mb_strlen(SearchTerms::fromInput(str_repeat('ب', 300))->query))->toBe(SearchTerms::MAX_LENGTH);
});

it('matches stored Arabic-form text and ZWNJ in SQL whatever the visitor types', function (): void {
    Post::factory()->published(now()->subDay())->create(['title' => 'نكاتي براي كاهش درد', 'slug' => 'arabic-form', 'excerpt' => 'خلاصه']);
    Post::factory()->published(now()->subDay())->create(['title' => 'عنوان دیگر', 'slug' => 'zwnj', 'excerpt' => "وقتی بدن استراحت می\u{200C}خواهد"]);
    Post::factory()->published(now()->subDay())->create(['title' => 'هفته ۱۲ بارداری', 'slug' => 'digits', 'excerpt' => 'خلاصه']);
    Post::factory()->create(['title' => 'نکاتی پیش‌نویس', 'slug' => 'draft']); // not published

    $search = app(SearchSite::class);
    $urls = static fn (string $q): array => array_map(static fn ($hit): string => $hit->url, $search->handle($q)->hits);

    expect($urls('نکاتی برای کاهش'))->toBe([route('blog.show', 'arabic-form')])
        ->and($urls('ميخواهد'))->toBe([route('blog.show', 'zwnj')])
        ->and($urls('هفته 12'))->toBe([route('blog.show', 'digits')])
        ->and($urls('هفته ١٢'))->toBe([route('blog.show', 'digits')])
        ->and($urls('50%'))->toBe([]); // LIKE wildcards are escaped
});

// ---- /search -----------------------------------------------------------------------------------------------------

it('searches posts and FAQ, title hits first, noindex and never page-cached', function (): void {
    fakeFaq();
    Post::factory()->published(now()->subDays(3))->create(['title' => 'چرا داده‌ها مهم‌اند', 'slug' => 'data-title', 'excerpt' => 'خلاصه']);
    Post::factory()->published(now()->subDay())->create(['title' => 'حریم خصوصی در اپ', 'slug' => 'data-body', 'excerpt' => 'درباره داده‌های سلامت']);

    $response = $this->get('/search?q='.rawurlencode('دادههای'))->assertOk();
    $html = (string) $response->getContent();

    expect($response->headers->get('X-Page-Cache'))->toBe('BYPASS')
        ->and($response->headers->get('X-Robots-Tag'))->toBe('noindex, follow')
        ->and($html)->toMatch('#<meta name="robots" content="noindex,\s*follow#')
        ->and(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('role="search"')
        ->and($html)->toContain('href="'.route('blog.show', 'data-body').'"')
        ->and($html)->toContain('href="'.route('faq').'#privacy"')
        ->and($html)->toContain('پرسش پرتکرار')
        ->and($html)->not->toContain('href="'.route('blog.show', 'data-title').'"'); // «داده‌ها» ≠ «داده‌های»

    $types = array_column(feedSearchGraph($html)['@graph'], '@type');
    expect($types)->toContain('SearchResultsPage');

    // The FAQ question matches in its title (score 2), the post only in its excerpt (score 1).
    $hits = app(SearchSite::class)->handle('دادههای')->hits;
    expect($hits[0]->score)->toBe(2)
        ->and($hits[0]->type)->toBe('faq')
        ->and(end($hits)->score)->toBe(1);
});

it('shows a friendly empty state and the too-short hint', function (): void {
    fakeFaq();

    $this->get('/search?q='.rawurlencode('واژه‌ای که نیست'))->assertOk()->assertSee('چیزی پیدا نشد');
    $this->get('/search?q=a')->assertOk()->assertSee(__('search.too_short'));
    $this->get('/search')->assertOk()->assertSee(__('search.lead'))->assertDontSee('نتیجه');
});

it('paginates results keeping the query in the links', function (): void {
    fakeFaq();
    Post::factory()->published(now()->subDay())->count(SearchController::PER_PAGE + 2)->create(['excerpt' => 'نکته‌ای درباره خواب']);

    $first = (string) $this->get('/search?q='.rawurlencode('خواب'))->assertOk()->getContent();
    $second = (string) $this->get('/search?q='.rawurlencode('خواب').'&page=2')->assertOk()->getContent();

    expect(substr_count($first, 'data-type="posts"'))->toBe(SearchController::PER_PAGE)
        ->and(substr_count($second, 'data-type="posts"'))->toBe(2)
        ->and($first)->toContain('rel="next"')
        ->and($first)->toContain(e('/search?q='.rawurlencode('خواب').'&page=2'));
});

it('rate limits searches per IP', function (): void {
    fakeFaq();
    foreach (range(1, SearchController::PER_MINUTE) as $i) {
        $this->get('/search?q=test'.$i)->assertOk();
    }

    $this->get('/search?q=again')->assertStatus(429);
});

it('caches results briefly by normalised query and logs anonymised terms', function (): void {
    fakeFaq();
    Post::factory()->published(now()->subDay())->create(['title' => 'کیک سالم', 'slug' => 'cake']);

    $search = app(SearchSite::class);
    expect($search->handle('كيك')->total)->toBe(1);

    Post::query()->getConnection()->table('blog_posts')->update(['title' => 'عنوان تازه']); // no observer → no bump
    expect($search->handle('کیک')->total)->toBe(1); // same normalised key → cached

    $search->handle('شماره 09121234567');
    $search->handle('چیزی که نیست');
    $log = app(SearchTermLog::class);

    expect($log->day(CarbonImmutable::now()))->toHaveKey('کیک')
        ->and($log->day(CarbonImmutable::now())['کیک']['count'])->toBe(2)
        ->and($log->day(CarbonImmutable::now()))->toHaveKey('شماره #')
        ->and($log->zeroResults(CarbonImmutable::now()))->toHaveKey('چیزی که نیست');
});

it('adds the WebSite SearchAction to the JSON-LD graph', function (): void {
    $graph = feedSearchGraph((string) $this->get('/blog')->assertOk()->getContent());
    $website = collect($graph['@graph'])->firstWhere('@type', 'WebSite');

    expect($website['potentialAction']['@type'])->toBe('SearchAction')
        ->and($website['potentialAction']['target']['urlTemplate'])->toBe(rtrim((string) config('app.url'), '/').'/search?q={search_term_string}')
        ->and($website['potentialAction']['query-input'])->toBe('required name=search_term_string');
});

// ---- /blog/feed --------------------------------------------------------------------------------------------------

it('serves a valid RSS 2.0 feed of the newest published posts with absolute https URLs', function (): void {
    Storage::fake('public');
    $cover = app(StoreMedia::class)->handle(new MediaUpload(MediaFixtures::jpeg(1600, 1000), 'cover.jpg', alt: 'کاور'));
    $category = Category::factory()->create(['name' => 'چرخه & پریود', 'slug' => 'cycle']);
    Post::factory()->published(now()->subHour())->create([
        'title' => 'نوشته <تازه> & مهم', 'slug' => 'newest', 'excerpt' => 'خلاصه کامل "با" نشانه‌ها', 'category_id' => $category->id,
        'cover_media_id' => $cover->id,
    ]);
    Post::factory()->published(now()->subDays(2))->count(FeedController::LIMIT + 2)->create();
    Post::factory()->create(['slug' => 'draft-post']);
    Post::factory()->scheduled(now()->addDay())->create(['slug' => 'future-post']);

    $response = $this->get('https://ritme.test/blog/feed')->assertOk();
    $body = (string) $response->getContent();

    expect($response->headers->get('Content-Type'))->toStartWith('application/rss+xml')
        ->and($body)->toStartWith('<?xml version="1.0" encoding="UTF-8"?>');

    libxml_use_internal_errors(true);
    $xml = simplexml_load_string($body);
    expect($xml)->not->toBeFalse()
        ->and(libxml_get_errors())->toBe([]);

    // W3C feed validator rules: rss@version, channel title/link/description, atom:link rel=self matching the URL,
    // RFC 822 dates, unique permalink guids, absolute links, enclosure url/length/type.
    $rfc822 = '/^(Mon|Tue|Wed|Thu|Fri|Sat|Sun), \d{2} (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{4} \d{2}:\d{2}:\d{2} [+-]\d{4}$/';
    $channel = $xml->channel;
    $atom = $channel->children('http://www.w3.org/2005/Atom');

    expect((string) $xml['version'])->toBe('2.0')
        ->and((string) $channel->title)->not->toBe('')
        ->and((string) $channel->description)->not->toBe('')
        ->and((string) $channel->link)->toBe('https://ritme.test/blog')
        ->and((string) $channel->language)->toBe('fa-IR')
        ->and((string) $channel->lastBuildDate)->toMatch($rfc822)
        ->and((string) $atom->link->attributes()['href'])->toBe('https://ritme.test/blog/feed')
        ->and((string) $atom->link->attributes()['rel'])->toBe('self')
        ->and(count($channel->item))->toBe(FeedController::LIMIT);

    $guids = [];
    foreach ($channel->item as $item) {
        expect((string) $item->title)->not->toBe('')
            ->and((string) $item->link)->toStartWith('https://ritme.test/blog/')
            ->and((string) $item->guid)->toBe((string) $item->link)
            ->and((string) $item->guid['isPermaLink'])->toBe('true')
            ->and((string) $item->pubDate)->toMatch($rfc822);
        $guids[] = (string) $item->guid;
    }
    expect(array_unique($guids))->toHaveCount(count($guids))
        ->and($body)->not->toContain('draft-post')
        ->and($body)->not->toContain('future-post');

    $first = $channel->item[0];
    expect((string) $first->title)->toBe('نوشته <تازه> & مهم')
        ->and((string) $first->description)->toBe('خلاصه کامل "با" نشانه‌ها')
        ->and((string) $first->category)->toBe('چرخه & پریود')
        ->and((string) $first->enclosure['url'])->toStartWith('https://ritme.test/')
        ->and((int) $first->enclosure['length'])->toBeGreaterThan(0)
        ->and((string) $first->enclosure['type'])->toBe('image/jpeg');
});

it('caches the feed in the blog namespace and refreshes it when a post is published', function (): void {
    Post::factory()->published(now()->subDay())->create(['title' => 'اولی', 'slug' => 'first']);
    expect((string) $this->get('/blog/feed')->getContent())->toContain('اولی');

    Post::query()->getConnection()->table('blog_posts')->update(['title' => 'بدون بامپ']);
    expect((string) $this->get('/blog/feed')->getContent())->toContain('اولی'); // served from cache

    Post::factory()->published(now())->create(['title' => 'دومی', 'slug' => 'second']); // observer bumps `blog`
    expect((string) $this->get('/blog/feed')->getContent())->toContain('دومی');
});

it('advertises the feed in the head of public pages', function (): void {
    $html = (string) $this->get('/blog')->assertOk()->getContent();

    expect($html)->toContain('<link rel="alternate" type="application/rss+xml"')
        ->and($html)->toContain('href="'.route('blog.feed').'"');
});

it('never gives a post a slug reserved by a /blog route', function (): void {
    $post = Post::factory()->create(['slug' => 'feed']);

    expect($post->fresh()->slug)->toBe('feed-2');
});
