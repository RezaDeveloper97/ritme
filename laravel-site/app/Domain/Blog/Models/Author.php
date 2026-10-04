<?php

declare(strict_types=1);

namespace App\Domain\Blog\Models;

use App\Domain\Seo\Concerns\HasSeo;
use Database\Factories\Blog\AuthorFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Support\Carbon;

/**
 * An author or medical reviewer (E-E-A-T): credentials and sameAs feed the Person JSON-LD node.
 *
 * @property int $id
 * @property string $name
 * @property string $slug
 * @property string|null $job_title
 * @property string|null $credentials
 * @property string|null $bio
 * @property int|null $avatar_media_id
 * @property list<string>|null $same_as
 * @property bool $is_medical_reviewer
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Author extends Model
{
    /** @use HasFactory<AuthorFactory> */
    use HasFactory;

    use HasSeo;

    protected $table = 'blog_authors';

    protected $fillable = ['name', 'slug', 'job_title', 'credentials', 'bio', 'avatar_media_id', 'same_as', 'is_medical_reviewer'];

    protected $attributes = ['is_medical_reviewer' => false];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'avatar_media_id' => 'integer',
            'same_as' => 'json:unicode',
            'is_medical_reviewer' => 'boolean',
        ];
    }

    protected static function newFactory(): AuthorFactory
    {
        return AuthorFactory::new();
    }

    /**
     * @return HasMany<Post, $this>
     */
    public function posts(): HasMany
    {
        return $this->hasMany(Post::class);
    }

    /**
     * @return HasMany<Post, $this>
     */
    public function reviewedPosts(): HasMany
    {
        return $this->hasMany(Post::class, 'reviewer_id');
    }
}
