'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack } from '@/shared/ui';

import type { InfoTopic } from '../model/topics';

interface InfoSection {
  heading: string;
  body: string;
}

// Static localized copy screen for privacy / terms / about / help. All content
// lives in the `profileInfo` namespace so both locales stay in sync; the
// component just maps over the topic's sections.
export function InfoPage({ topic }: { topic: InfoTopic }) {
  const t = useTranslations('profileInfo');
  const router = useRouter();

  // The sections array comes through next-intl's raw accessor; its shape is
  // fixed by the message files (heading + body per section).
  const sections = t.raw(`${topic}.sections`) as InfoSection[];

  return (
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span style={{ flex: 1, fontSize: 16, fontWeight: 800, color: 'var(--ink)', textAlign: 'start' }}>
          {t(`${topic}.title`)}
        </span>
      </div>

      <div className="scroll" style={{ padding: '4px 18px 24px' }}>
        {sections.map((section) => (
          <section key={section.heading} className="card" style={{ padding: 18, marginTop: 14 }}>
            <h2 style={{ fontSize: 14.5, fontWeight: 800, color: 'var(--ink)', margin: 0, textAlign: 'start' }}>
              {section.heading}
            </h2>
            <p style={{ fontSize: 13.5, lineHeight: 2, color: 'var(--muted)', margin: '8px 0 0', textAlign: 'start' }}>
              {section.body}
            </p>
          </section>
        ))}
      </div>
    </div>
  );
}
