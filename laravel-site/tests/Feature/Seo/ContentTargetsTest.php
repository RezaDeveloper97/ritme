<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Audit\Crawl\ContentTarget;
use App\Domain\Seo\Audit\Crawl\ContentTargets;
use App\Domain\Shop\Catalog\Models\Product;

it('links a content page to its admin edit form with focus keyword and body for the analyser', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'خواب-نوزاد', 'body' => '<p>متن مقاله</p>']);
    $post->seoMeta()->create(['focus_keyword' => '  خواب نوزاد ']);

    $target = app(ContentTargets::class)->for('blog.show', ['slug' => rawurlencode('خواب-نوزاد')]);

    expect($target->type)->toBe(ContentType::Post)
        ->and($target->editUrl)->toBe("/admin/blog/posts/{$post->id}/edit")
        ->and($target->focusKeyword)->toBe('خواب نوزاد')
        ->and($target->contentHtml)->toBe('<p>متن مقاله</p>');
});

it('keeps the content type but no fix link when the record is gone, and no keyword without seo meta', function (): void {
    $targets = app(ContentTargets::class);
    $product = Product::factory()->create(['slug' => 'bodysuit']);

    expect($targets->for('shop.product', ['slug' => 'deleted']))->toEqual(new ContentTarget(type: ContentType::Product))
        ->and($targets->for('shop.product', ['slug' => 'bodysuit'])->editUrl)->toBe("/admin/shop/products/{$product->id}/edit")
        ->and($targets->for('shop.product', ['slug' => 'bodysuit'])->focusKeyword)->toBe('');
});

it('maps static pages, directory landings and unknown routes', function (): void {
    $targets = app(ContentTargets::class);

    expect($targets->for('stage.cycle', [])->editUrl)->toBe('/admin/seo/static-pages/stage-cycle')
        ->and($targets->for('stage.cycle', [])->type)->toBeNull()
        ->and($targets->for('directory.city', ['city' => 'tehran'])->type)->toBe(ContentType::Archive)
        ->and($targets->for('directory.city', ['city' => 'tehran'])->editUrl)->toStartWith('/admin/directory/landings')
        ->and($targets->for('blog.show', []))->toEqual(new ContentTarget)
        ->and($targets->for('search', ['q' => 'x']))->toEqual(new ContentTarget)
        ->and($targets->for(null, []))->toEqual(new ContentTarget)
        ->and($targets->redirectsUrl())->toBe('/admin/seo/redirects/create')
        ->and($targets->indexingUrl())->toBe('/admin/seo/indexing');
});
