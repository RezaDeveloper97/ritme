'use client';

import { Link } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import type { Article } from '../model/types';

export interface ArticleCardProps {
  article: Article;
  /** Locale-aware route to the article, e.g. `/articles/period-blood-clots`. */
  href: string;
  /**
   * Pre-formatted reading time, e.g. "۴ دقیقه". `null` when the article has no
   * reading time — the host screen owns every string (FSD: entities stay
   * locale-agnostic, like CycleValuesCard).
   */
  readTimeLabel: string | null;
  /** Compact variant for horizontal rails (related reads). */
  compact?: boolean;
}

/**
 * One article as a tappable card: cover, title, and a meta line. Shared by the
 * library grid and the related-reads rail so the two never drift apart.
 */
export function ArticleCard({
  article,
  href,
  readTimeLabel,
  compact = false,
}: ArticleCardProps) {
  const meta = readTimeLabel ?? article.category;

  return (
    <Link href={href} className={compact ? 'ac ac-compact' : 'ac'}>
      <div className="ac-cover">
        {article.imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={article.imageUrl}
            alt=""
            loading="lazy"
            draggable={false}
            className="ac-img"
          />
        ) : (
          <Icon name="bookOpen" size={28} stroke="currentColor" />
        )}
        {article.category && !compact && <span className="ac-tag">{article.category}</span>}
      </div>

      <div className="ac-body">
        <h3 className="ac-title">{article.title}</h3>
        {!compact && article.excerpt && <p className="ac-excerpt">{article.excerpt}</p>}
        {meta && (
          <div className="ac-meta">
            <Icon name="bookOpen" size={15} stroke="currentColor" />
            <span>{meta}</span>
          </div>
        )}
      </div>
    </Link>
  );
}
