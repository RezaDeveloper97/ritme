<?php

declare(strict_types=1);

namespace Database\Factories\Blog;

use App\Domain\Blog\Models\Author;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Author>
 */
final class AuthorFactory extends Factory
{
    protected $model = Author::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return [
            'name' => "نویسنده {$n}",
            'slug' => "author-{$n}",
            'job_title' => 'نویسنده محتوای سلامت',
            'bio' => 'نویسنده مجله ریتمی.',
            'same_as' => [],
            'is_medical_reviewer' => false,
        ];
    }

    public function reviewer(): self
    {
        return $this->state(fn (): array => [
            'job_title' => 'متخصص زنان و زایمان',
            'credentials' => 'متخصص زنان و زایمان',
            'is_medical_reviewer' => true,
        ]);
    }
}
