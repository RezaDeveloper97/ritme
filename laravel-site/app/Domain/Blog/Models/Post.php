<?php

declare(strict_types=1);

namespace App\Domain\Blog\Models;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Seo\Concerns\HasSeo;
use Database\Factories\Blog\PostFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * A magazine article. Read through PostRepository (DTOs). On save (PostObserver): slug assigned/unique + history,
 * body/sources sanitised with heading ids, reading time counted, status/published_at normalised.
 *
 * @property int $id
 * @property string $title
 * @property string $slug
 * @property string|null $excerpt
 * @property string $body
 * @property string|null $sources
 * @property int|null $cover_media_id
 * @property int|null $cover_mobile_media_id
 * @property int|null $category_id
 * @property int|null $author_id
 * @property int|null $reviewer_id
 * @property Carbon|null $reviewed_at
 * @property LifeStage|null $life_stage
 * @property PostStatus $status
 * @property Carbon|null $published_at
 * @property Carbon|null $updated_content_at
 * @property int $reading_time
 * @property int $word_count
 * @property bool $is_featured
 * @property int $views
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Category|null $category
 * @property-read Author|null $author
 * @property-read Author|null $reviewer
 * @property-read Collection<int, Tag> $tags
 */
final class Post extends Model
{
    /** @use HasFactory<PostFactory> */
    use HasFactory;

    use HasSeo;

    protected $table = 'blog_posts';

    protected $fillable = [
        'title', 'slug', 'excerpt', 'body', 'sources',
        'cover_media_id', 'cover_mobile_media_id', 'category_id', 'author_id', 'reviewer_id', 'reviewed_at',
        'life_stage', 'status', 'published_at', 'updated_content_at', 'is_featured',
    ];

    protected $attributes = [
        'status' => 'draft',
        'body' => '',
        'reading_time' => 1,
        'word_count' => 0,
        'is_featured' => false,
        'views' => 0,
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'status' => PostStatus::class,
            'life_stage' => LifeStage::class,
            'published_at' => 'datetime',
            'updated_content_at' => 'datetime',
            'reviewed_at' => 'datetime',
            'reading_time' => 'integer',
            'word_count' => 'integer',
            'is_featured' => 'boolean',
            'views' => 'integer',
            'cover_media_id' => 'integer',
            'cover_mobile_media_id' => 'integer',
            'category_id' => 'integer',
            'author_id' => 'integer',
            'reviewer_id' => 'integer',
        ];
    }

    protected static function newFactory(): PostFactory
    {
        return PostFactory::new();
    }

    /**
     * Visible on the site: published and not dated in the future.
     *
     * @param  Builder<self>  $query
     */
    public function scopePublished(Builder $query): void
    {
        $query->where($this->qualifyColumn('status'), PostStatus::Published->value)
            ->where($this->qualifyColumn('published_at'), '<=', now());
    }

    public function isPublished(): bool
    {
        return $this->status === PostStatus::Published && $this->published_at !== null && ! $this->published_at->isFuture();
    }

    /**
     * @return BelongsTo<Category, $this>
     */
    public function category(): BelongsTo
    {
        return $this->belongsTo(Category::class);
    }

    /**
     * @return BelongsTo<Author, $this>
     */
    public function author(): BelongsTo
    {
        return $this->belongsTo(Author::class);
    }

    /**
     * @return BelongsTo<Author, $this>
     */
    public function reviewer(): BelongsTo
    {
        return $this->belongsTo(Author::class, 'reviewer_id');
    }

    /**
     * @return BelongsToMany<Tag, $this>
     */
    public function tags(): BelongsToMany
    {
        return $this->belongsToMany(Tag::class, 'blog_post_tag', 'post_id', 'tag_id');
    }

    /**
     * @return HasMany<PostSlug, $this>
     */
    public function previousSlugs(): HasMany
    {
        return $this->hasMany(PostSlug::class);
    }
}
