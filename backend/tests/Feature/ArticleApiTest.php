<?php

namespace Tests\Feature;

use App\Models\Article;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Locks the contract the React article library (list) and article page (single)
 * render from: only published rows, localized to the reader, paginated, and
 * with a body that is safe to inject as HTML.
 */
class ArticleApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    private function article(array $overrides = []): Article
    {
        static $n = 0;
        $n++;

        return Article::create(array_merge([
            'slug' => 'article-'.$n,
            'title' => ['fa' => 'عنوان '.$n, 'en' => 'Title '.$n],
            'is_published' => true,
            'published_at' => now()->subDays($n),
        ], $overrides));
    }

    public function test_list_returns_only_published_articles(): void
    {
        $this->actingUser();
        $this->article(['slug' => 'published-one']);
        $this->article(['slug' => 'draft-one', 'is_published' => false]);

        $items = $this->getJson('/api/v1/articles', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.items');

        $this->assertSame(['published-one'], array_column($items, 'slug'));
    }

    public function test_list_localizes_titles_to_the_requested_locale(): void
    {
        $this->actingUser();
        $this->article(['title' => ['fa' => 'قاعدگی', 'en' => 'Menstruation']]);

        $fa = $this->getJson('/api/v1/articles', ['Accept-Language' => 'fa'])->json('data.items.0.title');
        $en = $this->getJson('/api/v1/articles', ['Accept-Language' => 'en-US,en;q=0.9'])->json('data.items.0.title');

        $this->assertSame('قاعدگی', $fa);
        $this->assertSame('Menstruation', $en);
    }

    public function test_list_strips_markup_from_card_excerpts(): void
    {
        $this->actingUser();
        $this->article(['excerpt' => ['fa' => '<p>خلاصه</p><p>دوم</p>']]);

        $excerpt = $this->getJson('/api/v1/articles', ['Accept-Language' => 'fa'])
            ->json('data.items.0.excerpt');

        $this->assertSame('خلاصه دوم', $excerpt);
    }

    public function test_list_filters_by_category_and_reports_the_available_ones(): void
    {
        $this->actingUser();
        $this->article(['slug' => 'health-one', 'category' => 'سلامت']);
        $this->article(['slug' => 'nutrition-one', 'category' => 'تغذیه']);

        $response = $this->getJson('/api/v1/articles?category='.rawurlencode('تغذیه'), ['Accept-Language' => 'fa'])->assertOk();

        $this->assertSame(['nutrition-one'], array_column($response->json('data.items'), 'slug'));
        $this->assertSame(['تغذیه', 'سلامت'], $response->json('data.categories'));
    }

    public function test_list_searches_the_title_in_the_readers_locale(): void
    {
        $this->actingUser();
        $this->article(['slug' => 'clots', 'title' => ['fa' => 'لخته خون', 'en' => 'Blood clots']]);
        $this->article(['slug' => 'sleep', 'title' => ['fa' => 'خواب بهتر', 'en' => 'Better sleep']]);

        $items = $this->getJson('/api/v1/articles?q='.rawurlencode('لخته'), ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.items');

        $this->assertSame(['clots'], array_column($items, 'slug'));
    }

    public function test_list_paginates_and_reports_meta(): void
    {
        $this->actingUser();
        foreach (range(1, 5) as $i) {
            $this->article(['slug' => 'p-'.$i]);
        }

        $response = $this->getJson('/api/v1/articles?per_page=2&page=2')->assertOk();

        $this->assertCount(2, $response->json('data.items'));
        $this->assertSame(
            ['current_page' => 2, 'last_page' => 3, 'per_page' => 2, 'total' => 5],
            $response->json('data.meta'),
        );
    }

    public function test_list_rejects_an_oversized_page_size(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/articles?per_page=500')->assertStatus(422);
    }

    public function test_single_article_returns_its_body(): void
    {
        $this->actingUser();
        $this->article([
            'slug' => 'blood-clots',
            'title' => ['fa' => 'لخته خون'],
            'body' => ['fa' => '<p>متن اصلی مقاله</p>'],
            'read_time_minutes' => 4,
        ]);

        $article = $this->getJson('/api/v1/articles/blood-clots', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.article');

        $this->assertSame('لخته خون', $article['title']);
        $this->assertSame('<p>متن اصلی مقاله</p>', $article['body']);
        $this->assertSame(4, $article['read_time_minutes']);
    }

    public function test_single_article_body_is_sanitized(): void
    {
        $this->actingUser();
        $this->article([
            'slug' => 'xss',
            'body' => ['fa' => '<p onclick="steal()">سلام<script>alert(1)</script></p><a href="javascript:alert(1)">لینک</a>'],
        ]);

        $body = $this->getJson('/api/v1/articles/xss', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.article.body');

        $this->assertStringNotContainsString('<script', $body);
        $this->assertStringNotContainsString('onclick', $body);
        $this->assertStringNotContainsString('javascript:', $body);
        $this->assertStringContainsString('سلام', $body);
        $this->assertStringContainsString('لینک', $body);
    }

    public function test_single_article_suggests_related_reads_from_the_same_category(): void
    {
        $this->actingUser();
        $this->article(['slug' => 'main', 'category' => 'سلامت']);
        $this->article(['slug' => 'sibling', 'category' => 'سلامت']);
        $this->article(['slug' => 'stranger', 'category' => 'تغذیه']);

        $related = $this->getJson('/api/v1/articles/main', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.related');

        $this->assertSame(['sibling'], array_column($related, 'slug'));
    }

    public function test_unpublished_article_is_not_readable(): void
    {
        $this->actingUser();
        $this->article(['slug' => 'secret', 'is_published' => false]);

        $this->getJson('/api/v1/articles/secret')->assertNotFound();
        $this->getJson('/api/v1/articles/does-not-exist')->assertNotFound();
    }

    public function test_endpoints_require_authentication(): void
    {
        $this->getJson('/api/v1/articles')->assertUnauthorized();
        $this->getJson('/api/v1/articles/anything')->assertUnauthorized();
    }
}
