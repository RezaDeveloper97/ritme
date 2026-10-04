<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Database\Eloquent\ModelNotFoundException;
use InvalidArgumentException;

/**
 * Stores a visitor review of a published product as `pending` (moderated in L6-06; nothing is shown or counted before
 * approval). Text is trimmed and length-capped, the variant label kept only when given; the IP is stored only as a
 * keyed hash for spam checks. `is_verified_purchase` stays false (set from orders, L6-05). Rate limiting and the
 * honeypot belong to the delivery layer.
 */
final class SubmitProductReview
{
    public const MAX_NAME = 80;

    public const MAX_BODY = 2000;

    public const MAX_VARIANT_LABEL = 120;

    public function __construct(private readonly Config $config) {}

    /**
     * @throws ModelNotFoundException when the product is not published
     * @throws InvalidArgumentException on an invalid rating or empty text
     */
    public function handle(string $productSlug, string $authorName, int $rating, string $body, ?string $variantLabel = null, ?string $ip = null): ProductReview
    {
        if ($rating < 1 || $rating > 5) {
            throw new InvalidArgumentException('Rating must be between 1 and 5.');
        }

        $name = mb_substr(trim(preg_replace('/\s+/u', ' ', $authorName) ?? ''), 0, self::MAX_NAME);
        $text = mb_substr(trim(strip_tags($body)), 0, self::MAX_BODY);
        if ($name === '' || $text === '') {
            throw new InvalidArgumentException('Name and text are required.');
        }
        $label = $variantLabel === null ? '' : mb_substr(trim(strip_tags($variantLabel)), 0, self::MAX_VARIANT_LABEL);

        $product = Product::query()->published()->where('slug', $productSlug)->firstOrFail();

        $review = new ProductReview([
            'product_id' => $product->id,
            'author_name' => $name,
            'rating' => $rating,
            'body' => $text,
            'variant_label' => $label === '' ? null : $label,
            'status' => ReviewStatus::Pending,
            'ip_hash' => $ip === null ? null : hash_hmac('sha256', $ip, (string) $this->config->get('app.key')),
        ]);
        $review->save();

        return $review;
    }
}
