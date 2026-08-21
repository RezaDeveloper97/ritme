'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type InfoGroup, isInfoGroup, useInfoSections } from '@/entities/info';
import type { SheetContentProps } from '@/shared/sheet';

interface StaticSection {
  heading: string;
  body: string;
}

/**
 * The app's text screens — راهنما و پشتیبانی, حریم خصوصی, قوانین, درباره ما —
 * presented as a full sheet over whatever screen the reader came from.
 *
 * All four are maintained box-by-box in the admin panel and fetched from the
 * API, so copy can be corrected without shipping a release. The copy bundled
 * with the app stays as a fallback for when the request can't be made.
 *
 * The topic arrives as the sheet's URL argument, so it is validated here rather
 * than trusted: an unknown value renders nothing instead of throwing.
 */
export function InfoSheet({ arg }: SheetContentProps) {
  if (!arg || !isInfoGroup(arg)) return null;

  return <InfoBody group={arg} />;
}

/**
 * The admin-managed boxes of one screen. While they load, placeholder lines
 * keep the sheet from jumping; if the request fails (offline, server down) the
 * copy bundled with the app is shown instead — a policy or a support address
 * the reader can't read is worse than a slightly stale one.
 */
function InfoBody({ group }: { group: InfoGroup }) {
  const t = useTranslations('profileInfo');
  const locale = useLocale();
  const { data, isPending, isError } = useInfoSections(group, locale);

  if (isPending) {
    return (
      <>
        {[0, 1, 2].map((i) => (
          <section
            key={i}
            className="card pd-section pd-skel"
            aria-hidden="true"
          >
            <span className="skeleton-line pd-skel-head" />
            <span className="skeleton-line" />
            <span className="skeleton-line" />
            <span className="skeleton-line pd-skel-last" />
          </section>
        ))}
      </>
    );
  }

  if (isError || !data?.length) {
    // `t.raw` is typed per-namespace, so the group has to be narrowed for
    // next-intl to resolve the key.
    const sections = t.raw(`${group}.sections`) as StaticSection[];
    return (
      <>
        {sections.map((section) => (
          <section key={section.heading} className="card pd-section">
            <h3 className="pd-h2">{section.heading}</h3>
            <p className="pd-body">{section.body}</p>
          </section>
        ))}
      </>
    );
  }

  return (
    <>
      {data.map((section) => (
        <section key={section.id} className="card pd-section">
          <h3 className="pd-h2">{section.heading}</h3>
          <p className="pd-body">{section.body}</p>
          {section.linkUrl && section.linkLabel ? (
            <a
              className="pd-cta"
              href={section.linkUrl}
              // mailto:/tel: hand off to another app, so only real web links
              // open a tab — and those get the usual opener protection.
              {...(section.linkUrl.startsWith('http')
                ? { target: '_blank', rel: 'noopener noreferrer' }
                : {})}
            >
              {section.linkLabel}
            </a>
          ) : null}
        </section>
      ))}
    </>
  );
}
