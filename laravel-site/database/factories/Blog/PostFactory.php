<?php

declare(strict_types=1);

namespace Database\Factories\Blog;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Post>
 */
final class PostFactory extends Factory
{
    protected $model = Post::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return [
            'title' => "نوشته آزمایشی شماره {$n}",
            'slug' => "test-post-{$n}",
            'excerpt' => 'خلاصه کوتاه برای فهرست مقاله‌ها.',
            'body' => '<h2>بخش اول</h2><p>'.str_repeat('متن نمونه برای آزمایش مجله ریتمی است. ', 20).'</p><h2>بخش دوم</h2><p>پاراگراف دوم.</p>',
            'life_stage' => fake()->randomElement(LifeStage::cases()),
            'status' => PostStatus::Draft,
            'published_at' => null,
            'is_featured' => false,
        ];
    }

    public function published(?\DateTimeInterface $at = null): self
    {
        return $this->state(fn (): array => [
            'status' => PostStatus::Published,
            'published_at' => $at ?? now()->subDays(fake()->numberBetween(1, 60)),
        ]);
    }

    public function scheduled(?\DateTimeInterface $at = null): self
    {
        return $this->state(fn (): array => [
            'status' => PostStatus::Scheduled,
            'published_at' => $at ?? now()->addDay(),
        ]);
    }

    public function featured(): self
    {
        return $this->state(fn (): array => ['is_featured' => true]);
    }
}
