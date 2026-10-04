<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Support\PostContent;

it('sanitises the body on save, adds heading ids and noopener', function (): void {
    config(['app.url' => 'https://ritme.ir']);
    app()->forgetInstance(PostContent::class);

    $post = Post::factory()->create([
        'body' => '<h1>تیتر</h1><p onclick="x()">متن <script>alert(1)</script><a href="https://example.test">منبع</a> <a href="https://ritme.ir/faq">داخلی</a></p>'
            .'<img src="https://evil.test/x.jpg" alt="x"><img data-media-id="4" alt="تصویر"><h2>بخش دوم</h2>',
        'sources' => '<ul><li><a href="https://who.int/x" onclick="y()">WHO</a></li></ul><script>z()</script>',
        'excerpt' => '<p>خلاصه <b>پررنگ</b></p>',
    ]);

    expect($post->body)->toBe('<h2 id="تیتر">تیتر</h2><p>متن <a href="https://example.test" rel="noopener">منبع</a> <a href="https://ritme.ir/faq">داخلی</a></p><img data-media-id="4" alt="تصویر"><h2 id="بخش-دوم">بخش دوم</h2>')
        ->and($post->sources)->toBe('<ul><li><a href="https://who.int/x" rel="noopener">WHO</a></li></ul>')
        ->and($post->excerpt)->toBe('خلاصه پررنگ')
        ->and(Post::query()->find($post->id)?->body)->toBe($post->body);
});

it('computes word count and reading time from the Persian body', function (): void {
    $post = Post::factory()->create(['body' => '<p>'.str_repeat('کلمه‌ای ساده ', 250).'</p>']);

    expect($post->word_count)->toBe(500)
        ->and($post->reading_time)->toBe(3);

    $post->update(['body' => '<p>کوتاه</p>']);

    expect($post->fresh()?->reading_time)->toBe(1)
        ->and($post->fresh()?->word_count)->toBe(1);
});

it('stores empty sources and excerpt as null', function (): void {
    $post = Post::factory()->create(['sources' => '<p> </p>', 'excerpt' => '  ']);

    expect($post->sources)->toBeNull()->and($post->excerpt)->toBeNull();
});
