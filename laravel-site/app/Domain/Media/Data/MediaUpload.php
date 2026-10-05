<?php

declare(strict_types=1);

namespace App\Domain\Media\Data;

/**
 * Input of StoreMedia: a local file (e.g. an UploadedFile's real path — the delivery layer converts) plus the
 * editable details. Keeps the domain free of Illuminate\Http. `disk` overrides `media.disk` for uploads that must not be
 * public yet (e.g. the private `pending` disk; PromoteMedia moves them to the media disk later).
 */
final readonly class MediaUpload
{
    public function __construct(
        public string $path,
        public string $originalName,
        public ?string $alt = null,
        public ?string $title = null,
        public ?string $caption = null,
        public float $focalX = 0.5,
        public float $focalY = 0.5,
        public ?int $uploadedBy = null,
        public ?string $disk = null,
    ) {}
}
