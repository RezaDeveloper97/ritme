<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;

it('aggregates only approved, non-demo reviews into the product rating', function (): void {
    $product = Product::factory()->published()->create();
    ProductReview::factory()->approved()->create(['product_id' => $product->id, 'rating' => 5]);
    ProductReview::factory()->approved()->create(['product_id' => $product->id, 'rating' => 4]);
    ProductReview::factory()->create(['product_id' => $product->id, 'rating' => 1]);               // pending
    ProductReview::factory()->rejected()->create(['product_id' => $product->id, 'rating' => 1]);
    ProductReview::factory()->approved()->demo()->create(['product_id' => $product->id, 'rating' => 1]);

    $product->refresh();
    expect($product->rating_count)->toBe(2)->and($product->rating_avg)->toBe(4.5);

    $data = app(ProductRepository::class)->findPublishedBySlug($product->slug);
    expect($data?->rating()?->ratingCount)->toBe(2)
        ->and($data?->toSchema('https://ritme.ir/shop/product/x', [])->rating?->ratingValue)->toBe(4.5);
});

it('recalculates on moderation and deletion and stamps approval', function (): void {
    $product = Product::factory()->published()->create();
    $review = ProductReview::factory()->create(['product_id' => $product->id, 'rating' => 3]);
    expect($product->refresh()->rating_count)->toBe(0)->and($review->approved_at)->toBeNull();

    $review->update(['status' => ReviewStatus::Approved]);
    expect($product->refresh()->rating_count)->toBe(1)->and($review->refresh()->approved_at)->not->toBeNull();

    $review->delete();
    expect($product->refresh()->rating_count)->toBe(0)->and($product->rating_avg)->toBe(0.0);
});

it('lists approved reviews (demo flagged) but never emits a rating from demo samples', function (): void {
    $product = Product::factory()->published()->demo()->create();
    ProductReview::factory()->approved()->demo()->create(['product_id' => $product->id, 'variant_label' => 'سایز ۳-۶ ماه']);
    ProductReview::factory()->create(['product_id' => $product->id]);
    $repo = app(ProductRepository::class);

    $data = $repo->findPublishedBySlug($product->slug);
    $reviews = $repo->reviews($product->id);

    expect($data?->rating())->toBeNull()
        ->and($data?->toSchema('https://ritme.ir/shop/product/x', [])->rating)->toBeNull()
        ->and($data?->toCard()->rating())->toBeNull()
        ->and($reviews->total)->toBe(1)
        ->and($reviews->items[0]->isDemo)->toBeTrue()
        ->and($reviews->items[0]->variantLabel)->toBe('سایز ۳-۶ ماه');
});
