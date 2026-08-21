'use client';

import { useLocale, useTranslations } from 'next-intl';

import { ArticleCard, useArticle } from '@/entities/article';
import { getApiErrorStatus } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet, type SheetContentProps } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

/**
 * One article, opened from the home row or the library. Renders the cover, the
 * body an admin authored, and a few related reads — each of which opens as a
 * further sheet, so a reader can follow a thread and still back out one step
 * at a time.
 *
 * The body is HTML, and the ONLY place in the app that injects markup: the API
 * sanitizes it (tag allowlist, no event handlers, no `javascript:` URLs) before
 * it leaves the server. Anything else rendered here must stay plain text.
 *
 * `arg` is the article's slug — its public identifier, never an id.
 */
export function ArticleSheet({ arg }: SheetContentProps) {
  const t = useTranslations('articles');
  const locale = useLocale() as Locale;
  const mounted = useMounted();

  const query = useArticle(arg ?? '');
  const article = query.data?.article;
  const related = query.data?.related ?? [];

  const loading = !mounted || query.isPending;
  // A 404 is a real answer ("no such article"), not a failure to retry.
  const missing = query.isError && getApiErrorStatus(query.error) === 404;

  if (loading) return <ArticleSkeleton />;

  if (missing || !article) {
    return (
      <div className="art-missing">
        <span className="art-missing-icon">
          <Icon name="bookOpen" size={28} stroke="currentColor" />
        </span>
        <h3 className="art-missing-title">
          {missing ? t('detail.notFound.title') : t('error.title')}
        </h3>
        <p className="art-missing-text">
          {missing ? t('detail.notFound.text') : t('error.text')}
        </p>
        {missing ? (
          <button
            type="button"
            className="btn btn-ghost art-missing-cta"
            onClick={() => openSheet('articles')}
          >
            {t('detail.notFound.cta')}
          </button>
        ) : (
          <button
            type="button"
            className="btn btn-ghost art-missing-cta"
            onClick={() => void query.refetch()}
          >
            {t('error.retry')}
          </button>
        )}
      </div>
    );
  }

  return (
    <>
      {article.imageUrl && (
        <div className="art-cover">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={article.imageUrl} alt="" className="art-cover-img" />
        </div>
      )}

      <div className="art-head">
        {article.category && <span className="art-tag">{article.category}</span>}
        <h3 className="art-title">{article.title}</h3>

        <div className="art-meta">
          {article.readTimeMinutes !== null && (
            <span className="art-meta-item">
              <Icon name="bookOpen" size={15} stroke="currentColor" />
              {t('min', { n: article.readTimeMinutes })}
            </span>
          )}
          {article.publishedAt && (
            <span className="art-meta-item">
              <Icon name="calendar" size={15} stroke="currentColor" />
              {t('detail.publishedOn', {
                date: formatLongDate(fromApiDate(article.publishedAt), locale),
              })}
            </span>
          )}
        </div>

        {article.excerpt && <p className="art-excerpt">{article.excerpt}</p>}
      </div>

      {article.body ? (
        <div
          className="art-body"
          // Sanitized server-side — see the note on this component.
          dangerouslySetInnerHTML={{ __html: article.body }}
        />
      ) : (
        <p className="art-nobody">{t('detail.noBody')}</p>
      )}

      {/* §11: educational content is never presented as medical advice. */}
      <p className="art-disclaimer">
        <Icon name="info" size={15} stroke="currentColor" />
        <span>{t('detail.disclaimer')}</span>
      </p>

      {related.length > 0 && (
        <section className="art-related">
          <h4 className="art-related-title">{t('detail.related')}</h4>
          <div className="scroll-x">
            <div className="art-related-track">
              {related.map((item) => (
                <ArticleCard
                  key={item.id}
                  article={item}
                  onSelect={() => openSheet('article', item.slug)}
                  readTimeLabel={
                    item.readTimeMinutes === null
                      ? null
                      : t('min', { n: item.readTimeMinutes })
                  }
                  compact
                />
              ))}
            </div>
          </div>
        </section>
      )}
    </>
  );
}

/** Placeholder with the loaded sheet's rhythm — cover, title, body lines. */
function ArticleSkeleton() {
  return (
    <div aria-hidden className="art-skel">
      <span className="skeleton-line art-skel-cover" />
      <span className="skeleton-line art-skel-title" />
      <span className="skeleton-line art-skel-line" />
      <span className="skeleton-line art-skel-line" />
      <span className="skeleton-line art-skel-line is-short" />
    </div>
  );
}
