<?php

declare(strict_types=1);

use App\Domain\Blog\Data\AuthorData;
use App\Domain\Blog\Data\CategoryData;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostData;
use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Data\TagData;
use App\Domain\Blog\Enums\LifeStage;
use Carbon\CarbonImmutable;

function blogPostData(): PostData
{
    $date = CarbonImmutable::parse('2026-10-01T09:30:00+03:30');

    return new PostData(
        id: 7, title: 'عنوان', slug: 'slug', excerpt: 'خلاصه', body: '<h2 id="a">الف</h2><p>متن</p>', sources: '<p>منبع</p>',
        category: new CategoryData(1, 'چرخه و پریود', 'cycle', 'چرخه', null, LifeStage::Cycle),
        lifeStage: LifeStage::Cycle, tags: [new TagData(3, 'درد پریود', 'period-pain', 4)],
        author: new AuthorData(1, 'تیم', 'team'),
        reviewer: new AuthorData(2, 'دکتر', 'dr', 'متخصص', 'متخصص زنان', null, null, ['https://example.test/dr'], true),
        reviewedAt: $date, coverMediaId: 5, mobileCoverMediaId: null, readingTime: 6, wordCount: 1100,
        publishedAt: $date, updatedContentAt: $date->addDay(), isFeatured: true,
    );
}

it('round-trips post DTOs through plain arrays (cache format)', function (): void {
    $post = blogPostData();

    expect(PostData::fromArray($post->toArray()))->toEqual($post)
        ->and(PostCardData::fromArray($post->toCard()->toArray()))->toEqual($post->toCard());
});

it('exposes the TOC outline and the reviewer as a Person', function (): void {
    $post = blogPostData();
    $person = $post->reviewer?->toPerson('https://ritme.ir/blog/author/dr');

    expect($post->outline())->toBe([['level' => 2, 'id' => 'a', 'text' => 'الف']])
        ->and($person?->credential)->toBe('متخصص زنان')
        ->and($person?->sameAs)->toBe(['https://example.test/dr']);
});

it('computes page bounds', function (): void {
    $page = new PostPage([], total: 25, page: 3, perPage: 12);

    expect($page->lastPage())->toBe(3)
        ->and($page->isOutOfRange())->toBeFalse()
        ->and((new PostPage([], 25, 4, 12))->isOutOfRange())->toBeTrue()
        ->and((new PostPage([], 0, 1, 12))->isOutOfRange())->toBeFalse()
        ->and(PostPage::fromArray($page->toArray()))->toEqual($page);
});
