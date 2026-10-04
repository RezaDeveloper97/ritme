<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Data\ReviewSubmission;
use App\Domain\Directory\Enums\ReviewAspect;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceReview;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Database\Eloquent\ModelNotFoundException;
use InvalidArgumentException;

/**
 * Stores a visitor review of a published place as `pending` (moderated in L5-06; nothing is shown before approval).
 * Text is trimmed and length-capped, unknown aspects dropped; the IP is kept only as a keyed hash for spam checks.
 * Rate limiting and the honeypot belong to the delivery layer.
 */
final class SubmitPlaceReview
{
    public const MAX_NAME = 80;

    public const MAX_BODY = 2000;

    public function __construct(private readonly Config $config) {}

    /**
     * @throws ModelNotFoundException when the place is not published
     * @throws InvalidArgumentException on an invalid rating or empty text
     */
    public function handle(ReviewSubmission $submission): PlaceReview
    {
        if ($submission->rating < 1 || $submission->rating > 5) {
            throw new InvalidArgumentException('Rating must be between 1 and 5.');
        }

        $name = mb_substr(trim(preg_replace('/\s+/u', ' ', $submission->authorName) ?? ''), 0, self::MAX_NAME);
        $body = mb_substr(trim(strip_tags($submission->body)), 0, self::MAX_BODY);
        if ($name === '' || $body === '') {
            throw new InvalidArgumentException('Name and text are required.');
        }

        $place = Place::query()->published()->where('slug', $submission->placeSlug)->firstOrFail();

        $aspects = [];
        foreach (ReviewAspect::cases() as $aspect) {
            $score = $submission->aspects[$aspect->value] ?? null;
            if (is_int($score) && $score >= 1 && $score <= 5) {
                $aspects[$aspect->value] = $score;
            }
        }

        $review = new PlaceReview([
            'place_id' => $place->id,
            'author_name' => $name,
            'rating' => $submission->rating,
            'aspects' => $aspects === [] ? null : $aspects,
            'body' => $body,
            'status' => ReviewStatus::Pending,
            'ip_hash' => $submission->ip === null ? null : hash_hmac('sha256', $submission->ip, (string) $this->config->get('app.key')),
        ]);
        $review->save();

        return $review;
    }
}
