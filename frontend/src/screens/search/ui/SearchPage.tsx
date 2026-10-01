'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState, type ReactNode } from 'react';

import { useCurrentUser } from '@/entities/user';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  InfoNote,
  ListGroup,
  ListRow,
  PillChip,
  SecondaryButton,
  SectionTitle,
  SearchField,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { searchableQuery, useSearch } from '../api/queries';
import { hitLook } from '../model/hit-look';
import { addRecentSearch, clearRecentSearches, readRecentSearches } from '../model/recent';
import { searchTarget, type SearchTarget } from '../model/target';
import {
  SEARCH_MAX_LENGTH,
  SEARCH_SCOPES,
  type SearchGroup,
  type SearchHit,
  type SearchScope,
} from '../model/types';
import { useDebouncedValue } from '../model/use-debounced-value';

type T = ReturnType<typeof useTranslations<'search'>>;

const DEBOUNCE_MS = 300;
const SCORE_MAX = 10;

/** A hit with the screen it opens; hits whose screen doesn't exist yet are dropped. */
interface ShownHit {
  hit: SearchHit;
  target: SearchTarget;
}

interface ShownGroup {
  group: SearchGroup;
  hits: ShownHit[];
  /** `total` minus the hidden hits we know of. */
  total: number;
}

function shownGroups(groups: readonly SearchGroup[]): ShownGroup[] {
  return groups.flatMap((group) => {
    const hits = group.items.flatMap((hit): ShownHit[] => {
      const target = searchTarget(hit.route);
      return target ? [{ hit, target }] : [];
    });
    if (hits.length === 0) return [];
    return [{ group, hits, total: Math.max(hits.length, group.total - (group.items.length - hits.length)) }];
  });
}

/**
 * `/search` — global search (CB-NAV-02, nbd_Nav_Search), opened from the
 * Today header of every mode. A flow screen: no bottom nav (IA_Nav), the
 * cancel button returns to Today. The query lives only in component state —
 * never in the URL (CLAUDE.md §11).
 */
export function SearchPage() {
  const t = useTranslations('search');
  const locale = useLocale() as Locale;
  const router = useRouter();

  const [input, setInput] = useState('');
  const [scope, setScope] = useState<SearchScope>('all');
  // Recent searches belong to the signed-in account (model/recent.ts); none until it is known.
  const me = useCurrentUser();
  const owner = me.data ? String(me.data.id) : null;
  const [recent, setRecent] = useState<string[]>([]);
  useEffect(() => setRecent(owner ? readRecentSearches(owner) : []), [owner]);

  const debounced = useDebouncedValue(input, DEBOUNCE_MS);
  const query = searchableQuery(debounced);
  const search = useSearch(debounced, scope, locale);
  const groups = useMemo(() => shownGroups(search.data?.groups ?? []), [search.data]);

  const remember = (q: string) => {
    const s = searchableQuery(q);
    if (s && owner) setRecent(addRecentSearch(owner, s));
  };

  const open = (target: SearchTarget) => {
    remember(input);
    if (target.kind === 'sheet') openSheet(target.id, target.arg);
    else router.push(target.href);
  };

  const typedTooShort = input.trim() !== '' && searchableQuery(input) === null;

  let body: ReactNode;
  if (!query) {
    body = typedTooShort ? (
      <p className="srch-hint">{t('short')}</p>
    ) : recent.length === 0 ? (
      <EmptyState icon="search" title={t('start.title')} body={t('start.body')} className="srch-empty" />
    ) : null;
  } else if (search.isError && !search.data) {
    body = (
      <EmptyState
        icon="info"
        title={t('error.title')}
        body={t('error.body')}
        className="srch-empty"
        action={
          <SecondaryButton block={false} icon="refresh" onClick={() => void search.refetch()}>
            {t('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (!search.data) {
    body = (
      <SkeletonGroup label={t('loading')} className="srch-skel">
        <Skeleton shape="line" width="short" />
        <Skeleton shape="card" />
        <Skeleton shape="line" width="short" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (groups.length === 0) {
    body = (
      <EmptyState
        icon="search"
        title={t('empty.title')}
        body={t('empty.body', { query: search.data.query })}
        className="srch-empty"
      />
    );
  } else {
    body = (
      <div className="srch-groups" aria-busy={search.isFetching || undefined}>
        {groups.map((g) => (
          <ResultGroup
            key={g.group.key}
            shown={g}
            showAll={scope === 'all' && g.total > g.hits.length}
            onShowAll={() => setScope(g.group.key)}
            onOpen={open}
            t={t}
            locale={locale}
          />
        ))}
      </div>
    );
  }

  return (
    <div className="view srch-page">
      <SkyLayer />
      <div className="scroll srch-scroll">
        <div className="srch-bar">
          <SearchField
            className="srch-field"
            value={input}
            onValueChange={setInput}
            onSubmit={remember}
            label={t('field.label')}
            clearLabel={t('field.clear')}
            placeholder={t('field.placeholder')}
            maxLength={SEARCH_MAX_LENGTH}
            autoFocus
            autoComplete="off"
          />
          <button type="button" className="srch-cancel" onClick={() => router.push('/home')}>
            {t('cancel')}
          </button>
        </div>

        <div role="group" aria-label={t('scopes.label')} className="srch-scopes">
          {SEARCH_SCOPES.map((s) => (
            <PillChip key={s} className="srch-chip" pressed={scope === s} onPressedChange={() => setScope(s)}>
              {t(`scopes.${s}`)}
            </PillChip>
          ))}
        </div>

        {body}

        {query || recent.length > 0 ? <InfoNote className="srch-note">{t('shopNote')}</InfoNote> : null}

        {recent.length > 0 ? (
          <section className="srch-recent" aria-labelledby="srch-recent-title">
            <SectionTitle
              id="srch-recent-title"
              title={t('recent.title')}
              actionLabel={t('recent.clear')}
              onAction={() => {
                clearRecentSearches();
                setRecent([]);
              }}
            />
            <ul className="srch-recent-list">
              {recent.map((q) => (
                <li key={q}>
                  <button
                    type="button"
                    className="srch-recent-chip"
                    aria-label={t('recent.again', { query: q })}
                    onClick={() => setInput(q)}
                  >
                    <Icon name="clock" size={14} />
                    <span>{q}</span>
                  </button>
                </li>
              ))}
            </ul>
          </section>
        ) : null}
      </div>
    </div>
  );
}

function ResultGroup({
  shown,
  showAll,
  onShowAll,
  onOpen,
  t,
  locale,
}: {
  shown: ShownGroup;
  showAll: boolean;
  onShowAll: () => void;
  onOpen: (target: SearchTarget) => void;
  t: T;
  locale: Locale;
}) {
  const key = shown.group.key;
  return (
    <section className="srch-group" aria-labelledby={`srch-g-${key}`}>
      <SectionTitle
        id={`srch-g-${key}`}
        title={t(`groups.${key}`)}
        actionLabel={showAll ? t('seeAll', { count: formatNumber(shown.total, locale) }) : undefined}
        onAction={showAll ? onShowAll : undefined}
      />
      <ListGroup className="srch-list">
        {shown.hits.map(({ hit, target }) => {
          const look = hitLook(hit);
          const text = hitText(hit, t, locale);
          return (
            <ListRow
              key={`${hit.type}:${hit.id}`}
              icon={look.icon}
              iconTone={look.tone}
              title={text.title}
              description={text.subtitle ?? undefined}
              onClick={() => onOpen(target)}
            />
          );
        })}
      </ListGroup>
    </section>
  );
}

/** Title + caption of a row: the API gives labels, the screen words them like the board. */
function hitText(hit: SearchHit, t: T, locale: Locale): { title: string; subtitle: string | null } {
  const n = (v: number) => formatNumber(v, locale);
  const m = hit.meta;
  switch (hit.type) {
    case 'log_insight': {
      const title =
        m.days === null
          ? hit.title
          : m.windowKind === 'cycle'
            ? t('hit.insightCycle', { count: m.days, days: n(m.days), label: hit.title })
            : t('hit.insightDays', {
                count: m.days,
                days: n(m.days),
                label: hit.title,
                window: n(m.windowDays ?? 30),
              });
      const subtitle = m.peak
        ? m.peak.cycleDay !== null
          ? t('hit.peak', { day: n(m.peak.cycleDay), score: n(m.peak.score), max: n(SCORE_MAX) })
          : t('hit.peakNoDay', { score: n(m.peak.score), max: n(SCORE_MAX) })
        : hit.subtitle;
      return { title, subtitle };
    }
    case 'log_analysis':
      return { title: t('hit.analysisTitle', { label: hit.title }), subtitle: t('hit.analysisSub') };
    case 'article':
      return {
        title: hit.title,
        subtitle:
          m.readTimeMinutes !== null ? t('hit.articleTime', { minutes: n(m.readTimeMinutes) }) : t('hit.article'),
      };
    case 'reminder':
      return {
        title: hit.title,
        subtitle: hit.subtitle ?? (m.kind === 'appointment' ? t('hit.appointment') : t('hit.medication')),
      };
    case 'program':
      return { title: hit.title, subtitle: hit.subtitle ?? t('hit.program') };
    case 'checkup':
      return { title: hit.title, subtitle: hit.subtitle ?? t('hit.checkup') };
    case 'service':
      return { title: hit.title, subtitle: hit.subtitle ?? t('hit.service') };
  }
}
