import { describe, expect, it } from 'vitest';

import { articleDetailSchema, articlePageSchema, cycleArticlesSectionSchema } from './schema';

/** A section payload shaped like the backend's, with one fully filled article. */
const section = {
  key: 'articles',
  type: 'articles',
  title: 'بر اساس سیکل فعلی شما',
  order: 150,
  data: {
    items: [
      {
        id: 7,
        slug: 'period-blood-clots',
        title: 'لخته‌های خون در دوران پریود',
        excerpt: 'چه زمانی طبیعی است',
        read_time_minutes: 4,
        image_url: 'http://ritmeapp.ir/storage/articles/a.webp',
        category: 'سلامت',
        cycle_phases: ['menstruation', 'menstrual'],
      },
    ],
  },
};

describe('cycleArticlesSectionSchema', () => {
  it('maps the section onto the camelCase domain shape', () => {
    expect(cycleArticlesSectionSchema.parse(section)).toEqual({
      title: 'بر اساس سیکل فعلی شما',
      articles: [
        {
          id: 7,
          slug: 'period-blood-clots',
          title: 'لخته‌های خون در دوران پریود',
          excerpt: 'چه زمانی طبیعی است',
          imageUrl: 'http://ritmeapp.ir/storage/articles/a.webp',
          readTimeMinutes: 4,
          category: 'سلامت',
          cyclePhases: ['menstruation', 'menstrual'],
          // The home section doesn't carry a publish date; the library does.
          publishedAt: null,
        },
      ],
    });
  });

  it('keeps an article that only has the required fields', () => {
    const parsed = cycleArticlesSectionSchema.parse({
      title: null,
      data: { items: [{ id: 1, slug: 'a', title: 'یک' }] },
    });

    expect(parsed.articles[0]).toMatchObject({
      excerpt: null,
      imageUrl: null,
      readTimeMinutes: null,
      category: null,
      cyclePhases: [],
    });
  });

  it('reads an empty section as an empty list', () => {
    expect(cycleArticlesSectionSchema.parse({ title: null, data: {} }).articles).toEqual([]);
  });
});

describe('articlePageSchema', () => {
  it('maps a library page and its pagination meta', () => {
    const page = articlePageSchema.parse({
      items: [{ id: 3, slug: 'sleep', title: 'خواب', published_at: '2026-07-01' }],
      categories: ['تغذیه', 'سلامت'],
      meta: { current_page: 2, last_page: 4, per_page: 12, total: 40 },
    });

    expect(page.articles[0]).toMatchObject({ slug: 'sleep', publishedAt: '2026-07-01' });
    expect(page.categories).toEqual(['تغذیه', 'سلامت']);
    expect(page).toMatchObject({ page: 2, lastPage: 4, total: 40 });
  });

  it('reads a page with nothing on it as an empty first page', () => {
    expect(articlePageSchema.parse({})).toEqual({
      articles: [],
      categories: [],
      page: 1,
      lastPage: 1,
      total: 0,
    });
  });
});

describe('articleDetailSchema', () => {
  it('keeps the body markup and maps the related cards', () => {
    const detail = articleDetailSchema.parse({
      article: {
        id: 1,
        slug: 'clots',
        title: 'لخته',
        body: '<p>متن</p>',
        read_time_minutes: 4,
      },
      related: [{ id: 2, slug: 'sleep', title: 'خواب' }],
    });

    expect(detail.article).toMatchObject({
      slug: 'clots',
      body: '<p>متن</p>',
      readTimeMinutes: 4,
    });
    expect(detail.related.map((a) => a.slug)).toEqual(['sleep']);
  });

  it('treats a body-less article as one with no text', () => {
    const detail = articleDetailSchema.parse({
      article: { id: 1, slug: 'stub', title: 'بدون متن' },
    });

    expect(detail.article.body).toBeNull();
    expect(detail.related).toEqual([]);
  });
});
