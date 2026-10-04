<?php

declare(strict_types=1);

use App\Domain\Blog\Actions\FindOrCreateTag;
use App\Domain\Blog\Actions\PublishScheduledPosts;
use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Data\PageContext;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\SeoManager;
use App\Filament\Auth\AdminRole;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Components\Seo\SerpMeasure;
use App\Filament\Resources\Blog\Authors\Pages\CreateAuthor;
use App\Filament\Resources\Blog\Categories\CategoryResource;
use App\Filament\Resources\Blog\Categories\Pages\EditCategory;
use App\Filament\Resources\Blog\Posts\EditorImages;
use App\Filament\Resources\Blog\Posts\Pages\CreatePost;
use App\Filament\Resources\Blog\Posts\Pages\EditPost;
use App\Filament\Resources\Blog\Posts\Pages\ListPosts;
use App\Filament\Resources\Blog\Posts\PostPreviewController;
use App\Filament\Resources\Blog\Posts\PostResource;
use App\Filament\Resources\Blog\Tags\TagResource;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Testing\TestAction;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\Storage;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Media\MediaFixtures;

function blogAdmin(?AdminRole $role = AdminRole::Editor): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return $user;
}

function blogImage(): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload(MediaFixtures::jpeg(1600, 1000), 'cover.jpg', alt: 'تصویر جلد'));
}

beforeEach(function (): void {
    Storage::fake('public');
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
});

it('lets editors and SEO managers into the magazine and denies everyone else', function (): void {
    $post = Post::factory()->create();

    $this->get(PostResource::getUrl('index'))->assertRedirect('/admin/login');

    foreach ([AdminRole::Editor, AdminRole::SeoManager] as $role) {
        $user = blogAdmin($role);
        $this->actingAs($user)->get(PostResource::getUrl('index'))->assertOk();
        $this->actingAs($user)->get(PostResource::getUrl('edit', ['record' => $post]))->assertOk();
        $this->actingAs($user)->get(CategoryResource::getUrl('index'))->assertOk();
    }

    foreach ([AdminRole::Support, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
        $this->actingAs(blogAdmin($role))->get(PostResource::getUrl('index'))->assertForbidden();
    }

    $seo = blogAdmin(AdminRole::SeoManager);
    $this->actingAs($seo)->get(PostResource::getUrl('create'))->assertForbidden();
    expect($seo->can('delete', $post))->toBeFalse()
        ->and($seo->can('replicate', $post))->toBeFalse()
        ->and(blogAdmin()->can('replicate', $post))->toBeTrue()
        ->and(blogAdmin()->can('delete', $post))->toBeTrue();
});

it('creates a scheduled post that auto-publishes and is served with the SEO tab meta', function (): void {
    $this->actingAs($editor = blogAdmin());
    $image = blogImage();
    $category = Category::factory()->create(['name' => 'چرخه']);
    $author = Author::factory()->create(['name' => 'نویسنده']);
    $reviewer = Author::factory()->create(['name' => 'دکتر بازبین', 'is_medical_reviewer' => true]);
    $tag = Tag::factory()->create(['name' => 'درد']);
    $publishAt = now()->addHours(2)->startOfMinute();

    Livewire::test(CreatePost::class)
        ->fillForm([
            'title' => 'درد پریود چه زمانی طبیعی است',
            'slug' => 'period-pain-guide',
            'excerpt' => 'خلاصه‌ای کوتاه درباره درد پریود.',
            'body' => '<h1>تیتر</h1><p>متن مقاله <script>alert(1)</script>با جزئیات.</p><img data-id="'.$image->id.'" src="/media/x.jpg" alt="تصویر"><img src="https://evil.example/x.png">',
            'status' => PostStatus::Scheduled->value,
            'published_at' => $publishAt->format('Y-m-d H:i:s'),
            'category_id' => $category->id,
            'author_id' => $author->id,
            'reviewer_id' => $reviewer->id,
            'tag_ids' => [$tag->id],
            'cover_media_id' => $image->id,
            'seoMeta.title' => 'راهنمای درد پریود',
            'seoMeta.description' => 'چه دردی طبیعی است و چه زمانی بهتر است با پزشک صحبت کنی؛ نشانه‌ها و راه‌های کاهش درد.',
            'seoMeta.focus_keyword' => 'درد پریود',
        ])
        ->call('create')
        ->assertHasNoFormErrors();

    $post = Post::query()->where('slug', 'period-pain-guide')->firstOrFail();
    expect($post->status)->toBe(PostStatus::Scheduled)
        ->and($post->published_at?->equalTo($publishAt))->toBeTrue()
        ->and($post->body)->not->toContain('<script')->not->toContain('<h1')->not->toContain('evil.example')
        ->and($post->body)->toContain('data-media-id="'.$image->id.'"')->not->toContain('data-id=')
        ->and($post->tags()->pluck('blog_tags.id')->all())->toBe([$tag->id])
        ->and($post->reviewer_id)->toBe($reviewer->id);

    $meta = SeoMeta::query()->where('seoable_type', 'blog_post')->where('seoable_id', $post->id)->firstOrFail();
    expect($meta->title)->toBe('راهنمای درد پریود')
        ->and($meta->robots)->toBeNull()
        ->and($meta->sitemap_include)->toBeTrue()
        ->and($meta->focus_keyword)->toBe('درد پریود');

    expect(Activity::query()->where('log_name', 'blog')->where('event', 'created')->where('subject_id', $post->id)->where('causer_id', $editor->id)->exists())->toBeTrue();

    $posts = app(PostRepository::class);
    expect($posts->findPublishedBySlug('period-pain-guide'))->toBeNull();

    $this->travelTo($publishAt->copy()->addMinute());
    expect(app(PublishScheduledPosts::class)->handle())->toBe(1);

    $published = $posts->findPublishedBySlug('period-pain-guide');
    expect($published)->not->toBeNull()
        ->and(collect($posts->latest()->items)->pluck('slug')->all())->toContain('period-pain-guide');

    $head = app(SeoManager::class)->for($post->refresh())
        ->context(new PageContext(url('/blog/period-pain-guide'), 'blog.show'))
        ->resolve();
    expect($head->title)->toContain('راهنمای درد پریود')
        ->and($head->description)->toBe('چه دردی طبیعی است و چه زمانی بهتر است با پزشک صحبت کنی؛ نشانه‌ها و راه‌های کاهش درد.')
        ->and($head->canonical)->toEndWith('/blog/period-pain-guide');

    $this->get('/blog')->assertOk();
});

it('renders the SEO tab with counters, SERP and share previews and no external assets', function (): void {
    $this->actingAs(blogAdmin());
    $post = Post::factory()->create(['title' => 'عنوان آزمایشی مقاله', 'slug' => 'test-serp', 'excerpt' => 'خلاصه برای پیش‌نمایش گوگل.']);

    $html = $this->get(PostResource::getUrl('edit', ['record' => $post]))
        ->assertOk()
        ->assertSee('پیش‌نمایش در گوگل')
        ->assertSee('پیش‌نمایش کارت اشتراک‌گذاری')
        ->assertSee('data-seo-serp-preview', false)
        ->assertSee('عنوان آزمایشی مقاله — ریتمی')
        ->assertSee('خلاصه برای پیش‌نمایش گوگل.')
        ->assertSee('test-serp')
        ->assertSee('پیکسل')
        ->assertSee('تصویر از کتابخانه رسانه')
        ->getContent();

    preg_match_all('/<(?:script|link)\b[^>]*(?:src|href)="(https?:)?\/\/([^"\/]+)/i', (string) $html, $matches);
    $own = parse_url((string) config('app.url'), PHP_URL_HOST);
    $hosts = array_unique(array_filter($matches[2], static fn (string $host): bool => explode(':', $host)[0] !== $own));
    expect($hosts)->toBe([]);

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->fillForm(['seoMeta.title' => str_repeat('عنوان بسیار طولانی برای نتیجه گوگل ', 6), 'seoMeta.robots_index' => false])
        ->assertSee('noindex')
        ->assertSee('…');
});

it('lets an SEO manager change only the SEO tab', function (): void {
    $post = Post::factory()->create(['title' => 'عنوان اصلی', 'body' => '<p>متن اصلی</p>']);
    $this->actingAs(blogAdmin(AdminRole::SeoManager));

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->assertFormFieldIsDisabled('title')
        ->assertFormFieldIsEnabled('seoMeta.title')
        ->set('data.title', 'دست‌کاری شده')
        ->fillForm([
            'seoMeta.title' => 'عنوان سئو تازه',
            'seoMeta.robots_index' => false,
            'seoMeta.sitemap_include' => false,
            'seoMeta.sitemap_priority' => '0.8',
        ])
        ->call('save')
        ->assertHasNoFormErrors();

    $post->refresh();
    expect($post->title)->toBe('عنوان اصلی')
        ->and($post->body)->toContain('متن اصلی');

    $meta = $post->seoMeta()->firstOrFail();
    expect($meta->title)->toBe('عنوان سئو تازه')
        ->and($meta->robots)->toBe('noindex,follow')
        ->and($meta->sitemap_include)->toBeFalse()
        ->and((float) $meta->sitemap_priority)->toBe(0.8);

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->assertSchemaStateSet(['seoMeta.robots_index' => false, 'seoMeta.robots_follow' => true, 'seoMeta.sitemap_priority' => '0.8']);

    $category = Category::factory()->create(['name' => 'دسته اصلی']);
    Livewire::test(EditCategory::class, ['record' => $category->getRouteKey()])
        ->set('data.name', 'دست‌کاری')
        ->fillForm(['seoMeta.description' => 'توضیح دسته'])
        ->call('save')
        ->assertHasNoFormErrors();
    expect($category->refresh()->name)->toBe('دسته اصلی')
        ->and($category->seoMeta()->value('description'))->toBe('توضیح دسته');
});

it('keeps a revision trail of content and SEO changes', function (): void {
    $post = Post::factory()->create(['title' => 'عنوان قبلی']);
    $this->actingAs($editor = blogAdmin());

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->fillForm(['title' => 'عنوان جدید', 'seoMeta.focus_keyword' => 'کلیدواژه'])
        ->call('save')
        ->assertHasNoFormErrors();

    $entry = Activity::query()->where('log_name', 'blog')->where('event', 'updated')->latest('id')->firstOrFail();
    expect($entry->causer_id)->toBe($editor->id)
        ->and($entry->properties->get('old'))->toMatchArray(['title' => 'عنوان قبلی', 'seo.focus_keyword' => null])
        ->and($entry->properties->get('attributes'))->toBe(['title' => 'عنوان جدید', 'seo.focus_keyword' => 'کلیدواژه']);

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->mountAction('revisions')
        ->assertMountedActionModalSee('عنوان قبلی ← عنوان جدید')
        ->assertMountedActionModalSee('کلیدواژه کانونی');
});

it('duplicates a post as a draft with its tags and SEO but without the canonical override', function (): void {
    $this->actingAs(blogAdmin());
    $post = Post::factory()->published()->featured()->create(['title' => 'اصل مقاله', 'slug' => 'original']);
    $post->tags()->attach(Tag::factory()->create()->id);
    $post->seoMeta()->create(['title' => 'سئو اصل', 'canonical_url' => 'https://example.org/source']);

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->callAction('duplicate')
        ->assertNotified('نسخه پیش‌نویس ساخته شد.');

    $copy = Post::query()->whereKeyNot($post->id)->with(['tags', 'seoMeta'])->firstOrFail();
    expect($copy->title)->toBe('اصل مقاله (کپی)')
        ->and($copy->slug)->not->toBe('original')
        ->and($copy->status)->toBe(PostStatus::Draft)
        ->and($copy->is_featured)->toBeFalse()
        ->and($copy->published_at)->toBeNull()
        ->and($copy->tags)->toHaveCount(1)
        ->and($copy->seoMeta?->title)->toBe('سئو اصل')
        ->and($copy->seoMeta?->canonical_url)->toBeNull();

    $this->actingAs(blogAdmin(AdminRole::SeoManager));
    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])->assertActionHidden('duplicate');
});

it('bulk publishes and unpublishes posts', function (): void {
    $this->actingAs(blogAdmin());
    $drafts = Post::factory()->count(2)->create();

    Livewire::test(ListPosts::class)
        ->callTableBulkAction('publish', $drafts)
        ->assertNotified();

    foreach ($drafts as $post) {
        expect($post->refresh()->isPublished())->toBeTrue();
    }

    Livewire::test(ListPosts::class)->callTableBulkAction('unpublish', $drafts);
    expect($drafts->first()->refresh()->status)->toBe(PostStatus::Draft)
        ->and(Activity::query()->where('log_name', 'blog')->whereIn('event', ['published', 'unpublished'])->count())->toBe(4);

    $this->actingAs(blogAdmin(AdminRole::SeoManager));
    Livewire::test(ListPosts::class)->assertTableBulkActionHidden('publish');
});

it('serves a noindex draft preview only through a valid signed URL', function (): void {
    $post = Post::factory()->create(['title' => 'پیش‌نویس محرمانه', 'body' => '<p>متن پیش‌نویس</p>']);

    $this->get('/admin/blog/preview/'.$post->id)->assertForbidden();

    $response = $this->get(PostPreviewController::url($post))
        ->assertOk()
        ->assertHeader('X-Robots-Tag', 'noindex, nofollow')
        ->assertSee('پیش‌نویس محرمانه')
        ->assertSee('متن پیش‌نویس')
        ->assertSee('noindex', false);
    expect(substr_count((string) $response->getContent(), '<h1'))->toBe(1)
        ->and((string) $response->headers->get('Cache-Control'))->toContain('no-store');

    $url = PostPreviewController::url($post);
    $this->travel(PostPreviewController::TTL_HOURS + 1)->hours();
    $this->get($url)->assertForbidden();
});

it('manages tags inline and authors with credentials and sameAs', function (): void {
    $this->actingAs(blogAdmin());

    Livewire::test(CreatePost::class)
        ->fillForm(['title' => 'با برچسب تازه', 'body' => '<p>متن</p>', 'status' => PostStatus::Draft->value])
        ->assertFormFieldExists('tag_ids')
        ->callAction(TestAction::make('createOption')->schemaComponent('tag_ids'), data: ['name' => 'برچسب تازه'])
        ->assertHasNoFormErrors()
        ->call('create')
        ->assertHasNoFormErrors();

    $created = Post::query()->where('title', 'با برچسب تازه')->with('tags')->firstOrFail();
    expect($created->tags->pluck('name')->all())->toBe(['برچسب تازه']);

    $tag = app(FindOrCreateTag::class)->handle('  خواب   و استراحت ');
    expect($tag->name)->toBe('خواب و استراحت')
        ->and(app(FindOrCreateTag::class)->handle('خواب و استراحت')->id)->toBe($tag->id);

    Livewire::test(CreateAuthor::class)
        ->fillForm([
            'name' => 'دکتر نمونه',
            'credentials' => 'متخصص زنان و زایمان',
            'is_medical_reviewer' => true,
            'same_as' => ['https://example.org/profile'],
        ])
        ->call('create')
        ->assertHasNoFormErrors();

    $author = Author::query()->where('name', 'دکتر نمونه')->firstOrFail();
    expect($author->same_as)->toBe(['https://example.org/profile'])
        ->and($author->is_medical_reviewer)->toBeTrue()
        ->and($author->slug)->not->toBe('');

    Livewire::test(CreateAuthor::class)
        ->fillForm(['name' => 'نامعتبر', 'same_as' => ['javascript:alert(1)']])
        ->call('create')
        ->assertHasFormErrors();

    $this->get(TagResource::getUrl('index'))->assertOk();
});

it('inserts library images into the editor', function (): void {
    $this->actingAs(blogAdmin());
    $image = blogImage();
    $post = Post::factory()->create();

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->callAction(
            TestAction::make('insertMedia')->schemaComponent('body'),
            data: ['media_id' => $image->id],
            arguments: ['editorSelection' => ['type' => 'text', 'anchor' => 1, 'head' => 1]],
        )
        ->assertHasNoActionErrors()
        ->assertDispatched('run-rich-editor-commands');

    expect(EditorImages::url($image->id))->toContain('cover')
        ->and(EditorImages::url(999999))->toBeNull();
});

it('maps editor image ids to stored media ids and back', function (): void {
    $stored = EditorImages::toStorage('<p>الف</p><img data-id="12" src="/media/a.jpg" alt="ب"><img data-id="../x" src="/media/b.jpg">');
    expect($stored)->toContain('data-media-id="12"')->not->toContain('data-id=')->not->toContain('../x')
        ->and(EditorImages::toEditor($stored))->toContain('data-id="12"')->not->toContain('data-media-id');
});

it('measures SERP text and maps robots toggles', function (): void {
    expect(SerpMeasure::pixels('abc', 20))->toBeGreaterThan(0)
        ->and(SerpMeasure::status(null, SerpMeasure::TITLE_CHARS, 20, 580))->toBe('gray')
        ->and(SerpMeasure::status(str_repeat('کلمه ', 40), SerpMeasure::TITLE_CHARS, 20, 580))->toBe('danger')
        ->and(SerpMeasure::status('عنوانی با طول مناسب برای نتیجه جست‌وجو', SerpMeasure::TITLE_CHARS, 20, 580))->toBe('success')
        ->and(SerpMeasure::truncate(str_repeat('کلمه ', 80), 14, 300))->toEndWith('…')
        ->and(SerpMeasure::digits(120))->toBe('۱۲۰');

    expect(SeoFields::dehydrateRobots(['robots_index' => true, 'robots_follow' => true])['robots'])->toBeNull()
        ->and(SeoFields::dehydrateRobots(['robots_index' => true, 'robots_follow' => false])['robots'])->toBe('index,nofollow,max-image-preview:large,max-snippet:-1,max-video-preview:-1')
        ->and(SeoFields::fillRobots(['robots' => 'noindex,nofollow']))->toMatchArray(['robots_index' => false, 'robots_follow' => false]);
});
