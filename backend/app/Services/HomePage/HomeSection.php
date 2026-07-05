<?php

namespace App\Services\HomePage;

/**
 * @OA\Schema(
 *     schema="HomeSection",
 *     type="object",
 *     description="A single home-page widget. The frontend renders by `type` and iterates in `order`.",
 *     @OA\Property(property="key", type="string", example="next_period"),
 *     @OA\Property(property="type", type="string", example="next_period"),
 *     @OA\Property(property="title", type="string", nullable=true, example="۱ روز تا پریود بعدی"),
 *     @OA\Property(property="subtitle", type="string", nullable=true),
 *     @OA\Property(property="order", type="integer", example=3),
 *     @OA\Property(property="action", type="object", nullable=true,
 *         @OA\Property(property="key", type="string", example="view_all"),
 *         @OA\Property(property="label", type="string", example="مشاهده کامل")
 *     ),
 *     @OA\Property(property="data", type="object", description="Section-specific payload"),
 *     @OA\Property(property="meta", type="object", nullable=true)
 * )
 *
 * Immutable transfer object describing one rendered section.
 */
final class HomeSection
{
    public function __construct(
        public readonly string $key,
        public readonly string $type,
        public readonly ?string $title,
        public readonly array $data,
        public readonly int $order,
        public readonly ?string $subtitle = null,
        public readonly ?array $action = null,
        public readonly array $meta = [],
    ) {}

    public function toArray(): array
    {
        return [
            'key' => $this->key,
            'type' => $this->type,
            'title' => $this->title,
            'subtitle' => $this->subtitle,
            'order' => $this->order,
            'action' => $this->action,
            'data' => $this->data,
            'meta' => empty($this->meta) ? null : $this->meta,
        ];
    }
}
