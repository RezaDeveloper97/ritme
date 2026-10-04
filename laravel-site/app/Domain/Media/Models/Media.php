<?php

declare(strict_types=1);

namespace App\Domain\Media\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * One uploaded file. Read through MediaRepository (DTOs), written by the Media actions. Deleting a row removes
 * every file it owns (MediaObserver).
 *
 * @property int $id
 * @property string $disk
 * @property string $directory
 * @property string $filename
 * @property string|null $original_name
 * @property string $mime
 * @property int $size
 * @property int|null $width
 * @property int|null $height
 * @property string|null $alt
 * @property string|null $title
 * @property string|null $caption
 * @property float $focal_x
 * @property float $focal_y
 * @property string|null $dominant_color
 * @property string|null $lqip
 * @property array<string, array<string, array{w: int, h: int, path: string, size: int}>>|null $variants
 * @property Carbon|null $optimized_at
 * @property int|null $uploaded_by
 * @property string $hash
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Media extends Model
{
    protected $table = 'media';

    protected $fillable = [
        'disk', 'directory', 'filename', 'original_name', 'mime', 'size', 'width', 'height',
        'alt', 'title', 'caption', 'focal_x', 'focal_y', 'dominant_color', 'lqip', 'variants',
        'optimized_at', 'uploaded_by', 'hash',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'size' => 'integer',
            'width' => 'integer',
            'height' => 'integer',
            'focal_x' => 'float',
            'focal_y' => 'float',
            'variants' => 'array',
            'optimized_at' => 'datetime',
            'uploaded_by' => 'integer',
        ];
    }

    public function path(): string
    {
        return ltrim($this->directory.'/'.$this->filename, '/');
    }

    /**
     * Every path on the disk this row owns: the original plus all variant files.
     *
     * @return list<string>
     */
    public function allPaths(): array
    {
        $paths = [$this->path()];
        foreach ($this->variants ?? [] as $formats) {
            foreach ($formats as $file) {
                $paths[] = $file['path'];
            }
        }

        return array_values(array_unique($paths));
    }
}
