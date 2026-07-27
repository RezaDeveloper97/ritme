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
    <div className="card acc">
      <button
        type="button"
        className="acc-head"
        onClick={onToggle}
        aria-expanded={open}
      >
        <span className="dot acc-dot" style={{ background: color + '1A', color }}>
          <Icon name={icon} size={17} />
        </span>
        <span className="acc-title">{title}</span>
        <Icon name="chevronDown" size={18} className="acc-chev" />
      </button>
      {open && (
        <div className="acc-body">
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
      <div className="card wc-empty">
        {t('content.noContent')}
      </div>
    );
  }

  return (
    <div className="wc-list">
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
        <div className="wc-faq">
          <div className="wc-faq-title">
            {t('content.faqTitle')}
          </div>
          <div className="wc-faq-list">
            {content.faq.map((item, i) => (
              <div key={i} className="card acc">
                <button
                  type="button"
                  className="acc-head wc-faq-head"
                  onClick={() => setOpenFaq((f) => (f === i ? null : i))}
                  aria-expanded={openFaq === i}
                >
                  <Icon name="info" size={16} className="wc-faq-icon" />
                  <span className="wc-faq-q">{item.question}</span>
                  <Icon name="chevronDown" size={16} className="acc-chev" />
                </button>
                {openFaq === i && (
                  <div className="wc-faq-a">
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
