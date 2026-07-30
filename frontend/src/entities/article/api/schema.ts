import { z } from 'zod';

import type {
  Article,
  ArticlePage,
  ArticleWithRelated,
  CycleArticles,
} from '../model/types';

/**
 * The card fields every articles endpoint sends. Validated at the API boundary
 * (CLAUDE.md §10) and mapped from snake_case onto our camelCase domain shape.
 *
 * Articles are content the admin panel owns, so every optional field is treated
 * as genuinely optional: an article with no cover, no category and no reading
 * time still renders.
 */
const articleFields = z.object({
  id: z.number(),
  slug: z.string(),
  title: z.string(),
  excerpt: z.string().nullable().default(null),
  image_url: z.string().nullable().default(null),
  read_time_minutes: z.number().nullable().default(null),
  category: z.string().nullable().default(null),
  cycle_phases: z.array(z.string()).nullable().default([]),
  // The home section omits it; the library and article endpoints send it.
  published_at: z.string().nullable().default(null),
});

function toArticle(a: z.infer<typeof articleFields>): Article {
  return {
    id: a.id,
    slug: a.slug,
    title: a.title,
    excerpt: a.excerpt,
    imageUrl: a.image_url,
    readTimeMinutes: a.read_time_minutes,
    category: a.category,
    // `null` (a general article) and `[]` mean the same thing to a reader.
    cyclePhases: a.cycle_phases ?? [],
    publishedAt: a.published_at,
  };
}

const articleSchema = articleFields.transform(toArticle);

/** `GET /home/sections/articles` → `{ key, type, title, data: { items } }`. */
export const cycleArticlesSectionSchema = z
  .object({
    title: z.string().nullable().default(null),
    data: z.object({ items: z.array(articleSchema).default([]) }),
  })
  .transform(
    (s): CycleArticles => ({
      title: s.title,
      articles: s.data.items,
    }),
  );

/** `GET /articles` → one page of the library plus the category facet. */
export const articlePageSchema = z
  .object({
    items: z.array(articleSchema).default([]),
    categories: z.array(z.string()).default([]),
    meta: z
      .object({
        current_page: z.number().default(1),
        last_page: z.number().default(1),
        total: z.number().default(0),
      })
      .default({ current_page: 1, last_page: 1, total: 0 }),
  })
  .transform(
    (p): ArticlePage => ({
      articles: p.items,
      categories: p.categories,
      page: p.meta.current_page,
      lastPage: p.meta.last_page,
      total: p.meta.total,
    }),
  );

/**
 * `GET /articles/{slug}` → the article and its further reads.
 *
 * `body` is HTML: the backend allowlists its tags and strips event handlers and
 * `javascript:` URLs before it leaves the server, which is what makes the
 * article page's `dangerouslySetInnerHTML` defensible. Never point that render
 * at markup from any other source.
 */
export const articleDetailSchema = z
  .object({
    article: articleFields
      .extend({ body: z.string().nullable().default(null) })
      .transform((a) => ({ ...toArticle(a), body: a.body })),
    related: z.array(articleSchema).default([]),
  })
  .transform(
    (d): ArticleWithRelated => ({
      article: d.article,
      related: d.related,
    }),
  );
