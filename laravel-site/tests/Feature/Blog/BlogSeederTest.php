<?php

declare(strict_types=1);

use App\Domain\Blog\Contracts\CategoryRepository;
use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Post;
use Database\Seeders\BlogSeeder;

it('seeds the design article, blog samples and seven categories idempotently', function (): void {
    $this->seed(BlogSeeder::class);
    $this->seed(BlogSeeder::class);

    $posts = app(PostRepository::class);
    $article = $posts->findPublishedBySlug(BlogSeeder::ARTICLE_SLUG);

    expect(Post::query()->count())->toBe(6)
        ->and(count(app(CategoryRepository::class)->all()))->toBe(7)
        ->and($article?->title)->toBe('درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟')
        ->and(array_column($article?->outline() ?? [], 'text'))->toBe([
            'درد پریود چرا ایجاد می‌شود', 'چه چیزهایی ممکن است کمک کند', 'کی ارزش پیگیری دارد', 'در ریتمی چطور ثبت کنیم',
        ])
        ->and($article?->reviewer?->isMedicalReviewer)->toBeTrue()
        ->and($article?->category?->label)->toBe('چرخه')
        ->and($posts->featured()?->slug)->toBe(BlogSeeder::ARTICLE_SLUG)
        ->and($posts->currentSlugFor(BlogSeeder::ARTICLE_OLD_SLUG))->toBe(BlogSeeder::ARTICLE_SLUG)
        ->and($posts->related($article->id ?? 0))->toHaveCount(3);
});

it('keeps demo copy inside the content red lines', function (): void {
    $this->seed(BlogSeeder::class);

    $text = Post::query()->get()->map(static fn (Post $p): string => $p->title.' '.$p->excerpt.' '.$p->body.' '.$p->sources)->implode(' ')
        .' '.Author::query()->get()->map(static fn (Author $a): string => $a->name.' '.$a->bio)->implode(' ');

    foreach (['حتماً', 'حتما', 'قطعاً', 'قطعا', 'دقیق‌ترین', 'دقیقترین', 'تضمینی', 'تشخیص'] as $word) {
        expect($text)->not->toContain($word);
    }
});

it('is not part of the production DatabaseSeeder', function (): void {
    expect((string) file_get_contents(database_path('seeders/DatabaseSeeder.php')))->not->toContain('BlogSeeder');
});
