/**
 * An educational article as a card renders it — the shape shared by the home
 * "based on your current cycle" row, the article library, and the related-reads
 * rail. Text arrives already localized by the API (the backend picks fa/en from
 * the request locale), so nothing here is a translation key.
 */
export interface Article {
  id: number;
  slug: string;
  title: string;
  /** Plain-text summary (the API strips the editor's markup). */
  excerpt: string | null;
  /** Cover image URL, or `null` when the article has none. */
  imageUrl: string | null;
  readTimeMinutes: number | null;
  category: string | null;
  /** Cycle phases the article is tagged with; empty for general content. */
  cyclePhases: string[];
  /** ISO date (`YYYY-MM-DD`) the article went live; absent on the home row. */
  publishedAt: string | null;
}

/**
 * One article opened on its own page. `body` is HTML the backend sanitized on
 * the way out — it is the only value in the app that gets rendered as markup.
 */
export interface ArticleDetail extends Article {
  body: string | null;
}

/** The home "based on your current cycle" section, as the client consumes it. */
export interface CycleArticles {
  /** Section heading, worded and localized by the backend. */
  title: string | null;
  articles: Article[];
}

/** What the reader has narrowed the library down to. */
export interface ArticleFilters {
  /** Exact category, or `null` for "all". */
  category: string | null;
  /** Free-text search over title and excerpt; `''` means no search. */
  q: string;
}

/** One page of `GET /articles`. */
export interface ArticlePage {
  articles: Article[];
  /** Every category that currently has something published, for filter chips. */
  categories: string[];
  page: number;
  lastPage: number;
  total: number;
}

/** `GET /articles/{slug}` — the article plus a few further reads. */
export interface ArticleWithRelated {
  article: ArticleDetail;
  related: Article[];
}
