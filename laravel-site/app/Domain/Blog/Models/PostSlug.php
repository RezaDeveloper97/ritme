<?php

declare(strict_types=1);

namespace App\Domain\Blog\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A previous slug of a post (written by PostSlugger when the slug changes).
 *
 * @property int $id
 * @property int $post_id
 * @property string $slug
 * @property Carbon|null $created_at
 */
final class PostSlug extends Model
{
    public const UPDATED_AT = null;

    protected $table = 'blog_post_slugs';

    protected $fillable = ['post_id', 'slug'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['post_id' => 'integer'];
    }

    /**
     * @return BelongsTo<Post, $this>
     */
    public function post(): BelongsTo
    {
        return $this->belongsTo(Post::class);
    }
}
