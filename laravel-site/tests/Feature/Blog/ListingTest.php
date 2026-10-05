<?php

declare(strict_types=1);

use App\Domain\Blog\Actions\SyncPostTags;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Blog\Sitemap\AuthorSitemapProvider;
use App\Domain\Blog\Sitemap\TagSitemapProvider;
use App\Domain\Blog\Support\PostListIndexing;
use App\Domain\Newsletter\Actions\Subscribe;
use App\Domain\Newsletter\Enums\SubscribeOutcome;
use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Newsletter\Mail\ConfirmSubscriptionMail;
use App\Domain\Newsletter\Models\Subscriber;
use App\Domain\Seo\Models\SeoMeta;
use Illuminate\Support\Facades\Mail;
use Illuminate\Support\Facades\RateLimiter;

/**
 * @return array<string, mixed>
 */
function listingGraph(string $html): array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);

    return json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR);
}

/**
 * @return list<string>
 */
function listingTypes(string $html): array
{
    return array_map(static fn (array $node): string => (string) $node['@type'], listingGraph($html)['@graph'] ?? []);
}

function listingProduction(): void
{
    config(['app.env' => 'production']);
}

beforeEach(fn () => RateLimiter::clear('newsletter'));

// ---- /blog -------------------------------------------------------------------------------------------------------

it('renders the magazine with the featured hero, the grid without the hero, chips and the newsletter form', function (): void {
    $cycle = Category::factory()->create(['name' => 'چرخه و پریود', 'slug' => 'cycle', 'life_stage' => LifeStage::Cycle]);
    $featured = Post::factory()->published(now()->subDay())->create(['title' => 'نوشته ویژه', 'slug' => 'featured', 'is_featured' => true, 'category_id' => $cycle->id]);
    Post::factory()->published(now()->subDays(2))->count(3)->create();

    $response = $this->get('/blog')->assertOk();
    $html = (string) $response->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and(substr_count($html, 'href="'.route('blog.show', 'featured').'"'))->toBe(1) // not repeated in the grid
        ->and($html)->toContain('خواندنی‌های کوتاه، دقیق و بی‌قضاوت')
        ->and($html)->toContain('href="'.route('blog.category', 'cycle').'"')
        ->and($html)->toContain('action="'.route('newsletter.store').'"')
        ->and($html)->toContain('name="website"')
        ->and($html)->toContain('id="download"')
        ->and($html)->not->toContain('style=');

    $graph = listingGraph($html)['@graph'];
    $types = listingTypes($html);
    $itemList = collect($graph)->firstWhere('@type', 'ItemList');
    expect($types)->toContain('CollectionPage', 'ItemList', 'BreadcrumbList')
        ->and($itemList['numberOfItems'])->toBe(4)
        ->and($itemList['itemListElement'][0]['url'])->toEndWith('/blog/featured');
    expect($featured->id)->toBeInt();
});

it('paginates with self-canonical pages, numbered titles and 404s out of range', function (): void {
    listingProduction();
    Post::factory()->published()->count(14)->create();

    $first = (string) $this->get('/blog')->assertOk()->getContent();
    expect($first)->toContain('rel="next"')->and($first)->toContain('<meta name="robots" content="index,follow')
        ->and($first)->toContain('href="'.url('/blog').'?page=2"');

    $second = (string) $this->get('/blog?page=2')->assertOk()->getContent();
    expect($second)->toMatch('#<link rel="canonical" href="https://[^"]+/blog\?page=2">#')
        ->and($second)->toContain('صفحه ۲')
        ->and($second)->toContain('rel="prev"')
        ->and($second)->toContain('<meta name="robots" content="index,follow');

    $this->get('/blog?page=1')->assertRedirect(url('/blog'))->assertStatus(301);
    $this->get('/blog?page=3')->assertNotFound();
    $this->get('/blog?page=abc')->assertNotFound();
    $this->get('/blog?page=0')->assertNotFound();
});

it('makes filtered listings noindex', function (): void {
    listingProduction();
    Post::factory()->published()->create();

    expect((string) $this->get('/blog?sort=new')->getContent())->toContain('content="noindex,follow"')
        ->and((string) $this->get('/blog?utm_source=x')->getContent())->toContain('<meta name="robots" content="index,follow');
});

it('uses the admin seo_meta of the blog route', function (): void {
    SeoMeta::query()->create(['route_name' => 'blog.index', 'title' => 'عنوان سفارشی مجله برای آزمایش سئو']);

    expect((string) $this->get('/blog')->getContent())->toContain('<title>عنوان سفارشی مجله برای آزمایش سئو');
});

// ---- category / tag / author -------------------------------------------------------------------------------------

it('lists a category with its sub-categories, marks the chip and 404s unknown slugs', function (): void {
    $cycle = Category::factory()->create(['name' => 'چرخه و پریود', 'slug' => 'cycle']);
    $child = Category::factory()->create(['name' => 'درد', 'slug' => 'pain', 'parent_id' => $cycle->id]);
    Post::factory()->published()->create(['slug' => 'in-cycle', 'category_id' => $cycle->id]);
    Post::factory()->published()->create(['slug' => 'in-child', 'category_id' => $child->id]);
    Post::factory()->published()->create(['slug' => 'elsewhere']);

    $html = (string) $this->get('/blog/category/cycle')->assertOk()->getContent();

    expect($html)->toContain(route('blog.show', 'in-cycle'))
        ->and($html)->toContain(route('blog.show', 'in-child'))
        ->and($html)->not->toContain(route('blog.show', 'elsewhere'))
        ->and($html)->toMatch('#<a href="'.preg_quote(route('blog.category', 'cycle'), '#').'" aria-current="page"#')
        ->and($html)->toContain('<title>چرخه و پریود؛ مقاله‌های مجله ریتمی')
        ->and(listingTypes($html))->toContain('CollectionPage', 'ItemList');
    $crumbs = collect(listingGraph($html)['@graph'])->firstWhere('@type', 'BreadcrumbList')['itemListElement'];
    expect(array_column($crumbs, 'name'))->toBe(['خانه', 'مجله ریتمی', 'چرخه و پریود']);

    $this->get('/blog/category/missing')->assertNotFound();
});

it('keeps thin tag pages noindex,follow until they reach the threshold', function (): void {
    listingProduction();
    $tag = Tag::factory()->create(['name' => 'تخمک‌گذاری', 'slug' => 'ovulation']);
    $posts = Post::factory()->published()->count(2)->create();
    foreach ($posts as $post) {
        app(SyncPostTags::class)->handle($post, [$tag->id]);
    }

    $thin = (string) $this->get('/blog/tag/ovulation')->assertOk()->getContent();
    expect($thin)->toContain('content="noindex,follow"')->and($thin)->toContain('تخمک‌گذاری');

    app(SyncPostTags::class)->handle(Post::factory()->published()->create(), [$tag->id]);
    expect((string) $this->get('/blog/tag/ovulation')->getContent())->toContain('<meta name="robots" content="index,follow');

    expect(collect(app(TagSitemapProvider::class)->entries())->pluck('loc')->all())->toBe([route('blog.tag', 'ovulation')]);
    $this->get('/blog/tag/missing')->assertNotFound();
});

it('renders author pages with a Person node, credentials and their posts', function (): void {
    $author = Author::factory()->create([
        'name' => 'مریم نویسنده', 'slug' => 'maryam', 'job_title' => 'ماما', 'credentials' => 'کارشناس مامایی',
        'bio' => 'ماما و نویسنده مجله.', 'same_as' => ['https://example.test/maryam'],
    ]);
    Post::factory()->published()->create(['slug' => 'by-maryam', 'author_id' => $author->id]);
    Post::factory()->published()->create(['slug' => 'by-other']);

    $html = (string) $this->get('/blog/author/maryam')->assertOk()->getContent();
    $person = collect(listingGraph($html)['@graph'])->firstWhere('@type', 'Person');
    $page = collect(listingGraph($html)['@graph'])->firstWhere('@type', 'ProfilePage');

    expect($html)->toContain(route('blog.show', 'by-maryam'))
        ->and($html)->not->toContain(route('blog.show', 'by-other'))
        ->and($html)->toContain('کارشناس مامایی')
        ->and($html)->toContain('rel="me noopener"')
        ->and($person['name'])->toBe('مریم نویسنده')
        ->and($person['hasCredential']['credentialCategory'])->toBe('کارشناس مامایی')
        ->and($page['mainEntity']['@id'])->toBe($person['@id'])
        ->and(collect(app(AuthorSitemapProvider::class)->entries())->pluck('loc')->all())->toBe([route('blog.author', 'maryam')]);

    $this->get('/blog/author/missing')->assertNotFound();
});

it('indexes a reviewer profile without posts only once it has a real bio', function (): void {
    listingProduction();
    $reviewer = Author::factory()->reviewer()->create(['name' => 'متخصص بازبین', 'slug' => 'reviewer', 'bio' => '[معرفی کوتاه متخصص بازبین]']);
    $robots = static function (string $html): string {
        preg_match('#<meta name="robots" content="([^"]+)"#', $html, $m);

        return $m[1] ?? '';
    };
    // The router caches controller instances (with the previous request's scoped SeoManager): one per request.
    $fresh = static fn () => app('router')->getRoutes()->getByName('blog.author')?->flushController();
    $description = static function (string $html): string {
        preg_match('#<meta name="description" content="([^"]+)"#', $html, $m);

        return html_entity_decode($m[1] ?? '');
    };

    // Placeholder bio: noindex, and the lang default stands in for the description.
    $placeholder = (string) $this->get('/blog/author/reviewer')->assertOk()->getContent();
    expect($robots($placeholder))->toContain('noindex')
        ->and($description($placeholder))->not->toContain('[')
        ->and(mb_strlen($description($placeholder)))->toBeGreaterThanOrEqual(PostListIndexing::MIN_BIO_LENGTH);

    // Too short to be a real bio: still noindex.
    $reviewer->update(['bio' => 'متخصص زنان و زایمان.']);
    $fresh();
    expect($robots((string) $this->get('/blog/author/reviewer')->getContent()))->toContain('noindex');

    // A real bio: the credentials page is indexable without own articles, and the bio is the description.
    $bio = 'متخصص زنان و زایمان با پانزده سال تجربه بالینی که مقاله‌های سلامت چرخه و بارداری مجله ریتمی را پیش از انتشار بازبینی می‌کند.';
    $reviewer->update(['bio' => $bio]);
    $fresh();
    $real = (string) $this->get('/blog/author/reviewer')->getContent();
    expect($robots($real))->toStartWith('index')
        ->and($description($real))->toStartWith('متخصص زنان و زایمان با پانزده سال');

    // A plain author without posts stays noindex whatever the bio.
    Author::factory()->create(['slug' => 'writer', 'bio' => $bio]);
    $fresh();
    expect($robots((string) $this->get('/blog/author/writer')->getContent()))->toContain('noindex');

    expect(PostListIndexing::hasRealBio(null))->toBeFalse()
        ->and(PostListIndexing::hasRealBio('   '))->toBeFalse()
        ->and(PostListIndexing::hasRealBio($bio.' [نام]'))->toBeFalse()
        ->and(PostListIndexing::hasRealBio($bio))->toBeTrue();
});

// ---- newsletter --------------------------------------------------------------------------------------------------

it('subscribes with double opt-in and confirms with the mailed link', function (): void {
    Mail::fake();

    $this->post('/newsletter', ['email' => ' Reader@Example.com ', 'source' => 'blog/category/cycle'])
        ->assertRedirect(url('blog/category/cycle').'#newsletter')
        ->assertSessionHas('newsletter_status');

    $subscriber = Subscriber::query()->sole();
    expect($subscriber->email)->toBe('reader@example.com')
        ->and($subscriber->source)->toBe('blog/category/cycle')
        ->and($subscriber->status())->toBe(SubscriptionStatus::Pending)
        ->and($subscriber->consent_at)->not->toBeNull();

    $confirmUrl = null;
    Mail::assertSent(ConfirmSubscriptionMail::class, function (ConfirmSubscriptionMail $mail) use (&$confirmUrl, $subscriber): bool {
        $confirmUrl = $mail->confirmUrl;

        return $mail->hasTo('reader@example.com') && str_contains($mail->unsubscribeUrl, $subscriber->token);
    });

    $this->get((string) $confirmUrl)->assertOk()->assertSee('عضویتت تأیید شد')->assertSee('noindex', false);
    expect($subscriber->fresh()?->status())->toBe(SubscriptionStatus::Active);
});

it('answers the same for every address and never mails an active one', function (): void {
    Mail::fake();
    $subscribe = app(Subscribe::class);

    expect($subscribe->handle('a@example.com'))->toBe(SubscribeOutcome::Created)
        ->and($subscribe->handle('a@example.com'))->toBe(SubscribeOutcome::Throttled);
    Subscriber::query()->update(['confirmed_at' => now()]);
    expect($subscribe->handle('a@example.com'))->toBe(SubscribeOutcome::AlreadyActive);
    Mail::assertSentCount(1);

    $this->travel(11)->minutes();
    Subscriber::query()->update(['confirmed_at' => null]);
    expect($subscribe->handle('a@example.com'))->toBe(SubscribeOutcome::Resent);
    Mail::assertSentCount(2);
});

it('drops honeypot submissions silently and validates the address in Persian', function (): void {
    Mail::fake();

    $this->post('/newsletter', ['email' => 'bot@example.com', 'website' => 'http://spam.test'])
        ->assertRedirect(route('blog.index').'#newsletter')->assertSessionHas('newsletter_status');
    expect(Subscriber::query()->count())->toBe(0);

    $this->post('/newsletter', ['email' => 'not-an-email', 'source' => 'https://evil.test'])
        ->assertRedirect(route('blog.index').'#newsletter')
        ->assertSessionHasErrors(['email' => 'این نشانی ایمیل درست به نظر نمی‌رسد.'], errorBag: 'newsletter');
    Mail::assertNothingSent();
});

it('rate limits sign-ups per IP', function (): void {
    Mail::fake();

    foreach (range(1, 3) as $i) {
        $this->post('/newsletter', ['email' => "r{$i}@example.com"])->assertRedirect();
    }
    $this->post('/newsletter', ['email' => 'r4@example.com'])->assertStatus(429);
});

it('unsubscribes via the confirmation form or one-click POST, idempotently', function (): void {
    $subscriber = Subscriber::query()->create([
        'email' => 'u@example.com', 'token' => Subscriber::newToken(), 'consent_at' => now(), 'confirmed_at' => now(),
    ]);

    $this->get(route('newsletter.unsubscribe', [$subscriber->token]))->assertOk()
        ->assertSee('action="'.route('newsletter.unsubscribe.store', [$subscriber->token]).'"', false);
    expect($subscriber->fresh()?->status())->toBe(SubscriptionStatus::Active); // GET alone changes nothing

    $this->post(route('newsletter.unsubscribe.store', [$subscriber->token]))->assertOk()->assertSee('عضویتت لغو شد');
    $this->post(route('newsletter.unsubscribe.store', [$subscriber->token]))->assertOk()->assertSee('عضویتت لغو شد');
    expect($subscriber->fresh()?->status())->toBe(SubscriptionStatus::Unsubscribed);

    // An old confirm link never re-activates an unsubscribed address; a new sign-up rotates the token.
    $this->get(route('newsletter.confirm', [$subscriber->token]))->assertOk()->assertSee('این پیوند معتبر نیست');
    Mail::fake();
    expect(app(Subscribe::class)->handle('u@example.com'))->toBe(SubscribeOutcome::Resubscribed)
        ->and($subscriber->fresh()?->token)->not->toBe($subscriber->token)
        ->and($subscriber->fresh()?->status())->toBe(SubscriptionStatus::Pending);

    $this->get(route('newsletter.confirm', ['unknowntoken']))->assertOk()->assertSee('این پیوند معتبر نیست');
});
