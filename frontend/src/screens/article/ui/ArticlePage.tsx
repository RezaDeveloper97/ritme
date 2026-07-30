'use client';

import { useLocale, useTranslations } from 'next-intl';

import { ArticleCard, useArticle } from '@/entities/article';
import { getApiErrorStatus } from '@/shared/api';
import { Link, useRouter } from '@/shared/i18n';
import type { Locale } from '@/shared/i18n';
import { formatJalali, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { Icon, NavBack } from '@/shared/ui';

interface ArticlePageProps {
  /** Slug from the route — the article's public identifier, never an id. */
  slug: string;
}

/**
 * One article, opened from the home row or the library. Renders the cover,
 * the body an admin authored, and a few related reads.
 *
 * The body is HTML, and the ONLY place in the app that injects markup: the API
 * sanitizes it (tag allowlist, no event handlers, no `javascript:` URLs) before
 * it leaves the server. Anything else rendered here must stay plain text.
 */
export function ArticlePage({ slug }: ArticlePageProps) {
  const t = useTranslations('articles');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();

  const query = useArticle(slug);
  const article = query.data?.article;
  const related = query.data?.related ?? [];

  const loading = !mounted || query.isPending;
  // A 404 is a real answer ("no such article"), not a failure to retry.
  const missing = query.isError && getApiErrorStatus(query.error) === 404;

  return (
    <div className="view art-page">
      <div className="home-grad art-grad" />

      <div className="hdr art-hdr">
        <NavBack onClick={() => router.back()} label={t('back')} />
        <Link href="/articles" className="art-hdr-link">
          {t('detail.backToList')}
        </Link>
      </div>

      <div className="scroll art-scroll">
        {loading ? (
          <ArticleSkeleton />
        ) : missing || !article ? (
          <div className="art-missing">
            <span className="art-missing-icon">
              <Icon name="bookOpen" size={28} stroke="currentColor" />
            </span>
            <h1 className="art-missing-title">
              {missing ? t('detail.notFound.title') : t('error.title')}
            </h1>
            <p className="art-missing-text">
              {missing ? t('detail.notFound.text') : t('error.text')}
            </p>
            {missing ? (
              <Link href="/articles" className="btn btn-ghost art-missing-cta">
                {t('detail.notFound.cta')}
              </Link>
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
        ) : (
          <>
            {article.imageUrl && (
              <div className="art-cover">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={article.imageUrl} alt="" className="art-cover-img" />
              </div>
            )}

            <div className="art-head">
              {article.category && <span className="art-tag">{article.category}</span>}
              <h1 className="art-title">{article.title}</h1>

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
                      date: formatJalali(fromApiDate(article.publishedAt), locale),
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
                <h2 className="art-related-title">{t('detail.related')}</h2>
                <div className="scroll-x">
                  <div className="art-related-track">
                    {related.map((item) => (
                      <ArticleCard
                        key={item.id}
                        article={item}
                        href={`/articles/${item.slug}`}
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
        )}

        <div className="page-tail" />
      </div>
    </div>
  );
}

/** Placeholder with the loaded page's rhythm — cover, title, body lines. */
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
