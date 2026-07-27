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
    <div className="view pd-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="pd-title">
          {t(`${topic}.title`)}
        </span>
      </div>

      <div className="scroll pd-scroll">
        {sections.map((section) => (
          <section key={section.heading} className="card pd-section">
            <h2 className="pd-h2">
              {section.heading}
            </h2>
            <p className="pd-body">
              {section.body}
            </p>
          </section>
        ))}
      </div>
    </div>
  );
}
