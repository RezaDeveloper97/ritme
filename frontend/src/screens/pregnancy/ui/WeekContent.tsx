'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { CONTENT_MODULE_ORDER, type ContentModuleKey, type WeeklyContent } from '@/entities/pregnancy';
import { Icon, type IconName } from '@/shared/ui';

type T = ReturnType<typeof useTranslations>;

// Presentation-only icon + accent per content module (a UI concern, kept out of
// the entity model).
const MODULE_STYLE: Record<ContentModuleKey, { icon: IconName; color: string }> = {
  fetalDevelopment: { icon: 'heart', color: 'var(--rose)' },
  motherBodyChanges: { icon: 'sparkle', color: 'var(--violet)' },
  bodyAdaptation: { icon: 'refresh', color: 'var(--indigo-deep)' },
  emotionalStatus: { icon: 'smile', color: 'var(--orange)' },
  keyNutrition: { icon: 'glass', color: 'var(--green)' },
  physicalActivity: { icon: 'walk', color: 'var(--teal)' },
  dosAndDonts: { icon: 'checkCircle', color: 'var(--blue)' },
  carePlan: { icon: 'stetho', color: 'var(--brand)' },
  testsAndCheckups: { icon: 'chart', color: 'var(--muted)' },
};

function AccordionItem({
  title,
  icon,
  color,
  body,
  open,
  onToggle,
}: {
  title: string;
  icon: IconName;
  color: string;
  body: string;
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
      <button
        type="button"
        onClick={onToggle}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 11,
          width: '100%',
          padding: '13px 14px',
          background: 'transparent',
          border: 0,
          cursor: 'pointer',
          textAlign: 'start',
          fontFamily: 'inherit',
        }}
      >
        <span className="dot" style={{ width: 34, height: 34, background: color + '1A', color }}>
          <Icon name={icon} size={17} />
        </span>
        <span style={{ flex: 1, fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>{title}</span>
        <Icon name="chevronDown" size={18} style={{ color: 'var(--muted)', transform: open ? 'rotate(180deg)' : 'none', transition: 'transform .15s' }} />
      </button>
      {open && (
        <div style={{ padding: '0 14px 14px', fontSize: 13.5, lineHeight: 2, color: 'var(--ink)', whiteSpace: 'pre-line', textAlign: 'start' }}>
          {body}
        </div>
      )}
    </div>
  );
}

/** Renders a week's educational modules as an accordion plus its FAQ. Degrades
 *  gracefully to an empty-state note when the week hasn't been authored yet. */
export function WeekContent({ content, t }: { content: WeeklyContent | null; t: T }) {
  const [openKey, setOpenKey] = useState<ContentModuleKey | null>('fetalDevelopment');
  const [openFaq, setOpenFaq] = useState<number | null>(null);

  const modules = CONTENT_MODULE_ORDER.map((key) => ({ key, body: content?.[key] ?? null })).filter(
    (m): m is { key: ContentModuleKey; body: string } => !!m.body,
  );

  if (!content || modules.length === 0) {
    return (
      <div className="card" style={{ padding: '18px 14px', textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
        {t('content.noContent')}
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {modules.map(({ key, body }) => {
        const style = MODULE_STYLE[key];
        return (
          <AccordionItem
            key={key}
            title={t(`content.modules.${key}`)}
            icon={style.icon}
            color={style.color}
            body={body}
            open={openKey === key}
            onToggle={() => setOpenKey((k) => (k === key ? null : key))}
          />
        );
      })}

      {content.faq.length > 0 && (
        <div style={{ marginTop: 4 }}>
          <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', margin: '4px 2px 8px', textAlign: 'start' }}>
            {t('content.faqTitle')}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {content.faq.map((item, i) => (
              <div key={i} className="card" style={{ padding: 0, overflow: 'hidden' }}>
                <button
                  type="button"
                  onClick={() => setOpenFaq((f) => (f === i ? null : i))}
                  style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%', padding: '12px 14px', background: 'transparent', border: 0, cursor: 'pointer', textAlign: 'start', fontFamily: 'inherit' }}
                >
                  <Icon name="info" size={16} style={{ color: 'var(--brand)' }} />
                  <span style={{ flex: 1, fontSize: 13.5, fontWeight: 700, color: 'var(--ink)' }}>{item.question}</span>
                  <Icon name="chevronDown" size={16} style={{ color: 'var(--muted)', transform: openFaq === i ? 'rotate(180deg)' : 'none' }} />
                </button>
                {openFaq === i && (
                  <div style={{ padding: '0 14px 13px', fontSize: 13, lineHeight: 1.9, color: 'var(--muted)', textAlign: 'start' }}>
                    {item.answer}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
