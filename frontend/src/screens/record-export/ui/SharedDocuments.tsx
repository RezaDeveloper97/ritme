'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { DOCUMENT_KINDS, downloadSignedFile, KIND_LOOK, type SharedDocument } from '@/entities/health-record';
import type { Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { Card, IconCircle } from '@/shared/ui';
import './rx.css';

const isKind = (k: string): k is (typeof DOCUMENT_KINDS)[number] => (DOCUMENT_KINDS as readonly string[]).includes(k);

/**
 * CB-REC-05: the documents the owner added to a 24h doctor summary, on the public page. Files are signed for five
 * minutes and vetted to the API's /files/ path; they download through a blob (no signed URL in a new tab, CB-REC-04
 * L2), with no credentials and no referrer.
 */
export function SharedDocuments({ documents }: { documents: SharedDocument[] }) {
  const t = useTranslations('record.doctor');
  const tk = useTranslations('record.kinds');
  const loc = useLocale() as Locale;
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);

  const download = async (url: string, name: string) => {
    setBusy(url);
    setFailed(false);
    try {
      await downloadSignedFile(url, name);
    } catch {
      setFailed(true);
    } finally {
      setBusy(null);
    }
  };

  return (
    <section className="rx-docs" aria-labelledby="rx-docs-title">
      <h2 id="rx-docs-title" className="rx-docs-title">
        {t('docsTitle')}
      </h2>
      <ul className="rx-docs-list">
        {documents.map((d, i) => {
          const kind = isKind(d.kind) ? d.kind : 'other';
          const look = KIND_LOOK[kind];
          const meta = [tk(kind), d.date ? formatDayMonth(fromApiDate(d.date), loc) : null, d.centre, d.doctor].filter(Boolean);
          return (
            <li key={`${d.kind}-${i}`}>
              <Card className="rx-doc">
                <div className="rx-doc-head">
                  <IconCircle icon={look.icon} tone={look.tone} size="sm" />
                  <div className="rx-doc-text">
                    <p className="rx-doc-title">{d.title || tk(kind)}</p>
                    <p className="rx-doc-meta">{meta.join(' · ')}</p>
                  </div>
                </div>
                {d.files.length ? (
                  <div className="rx-doc-files">
                    {d.files.map((f, j) => {
                      const label = t('docFile', { n: formatNumber(j + 1, loc) });
                      return (
                        <button
                          key={f.url}
                          type="button"
                          className="rx-doc-file"
                          disabled={busy === f.url}
                          onClick={() => void download(f.url, `${kind}-${i + 1}-${j + 1}${extensionOf(f.mime)}`)}
                        >
                          {label} · {t('download')}
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <p className="rx-doc-meta">{t('noFiles')}</p>
                )}
              </Card>
            </li>
          );
        })}
      </ul>
      {failed ? (
        <p className="rx-error" role="alert">
          {t('downloadError')}
        </p>
      ) : null}
    </section>
  );
}

function extensionOf(mime: string): string {
  if (mime === 'application/pdf') return '.pdf';
  if (mime === 'image/png') return '.png';
  if (mime === 'image/webp') return '.webp';
  if (mime.startsWith('image/')) return '.jpg';
  return '';
}
