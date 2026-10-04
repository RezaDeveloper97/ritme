<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Data;

/**
 * One entry of the manifest `icons` array (also used for shortcut icons).
 */
final readonly class ManifestIcon
{
    public function __construct(
        public string $src,
        public int $size,
        public string $purpose = 'any',
        public string $type = 'image/png',
    ) {}

    /**
     * @return array{src: string, sizes: string, type: string, purpose: string}
     */
    public function toArray(): array
    {
        return ['src' => $this->src, 'sizes' => "{$this->size}x{$this->size}", 'type' => $this->type, 'purpose' => $this->purpose];
    }
}
