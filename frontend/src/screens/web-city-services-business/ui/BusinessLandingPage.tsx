import { useTranslations } from 'next-intl';

import type { BusinessTerm } from '@/entities/city-service';
import { Link } from '@/shared/i18n';
import { Icon, type IconName } from '@/shared/ui';

import { BusinessShell } from './BusinessShell';

const WHY: { key: 'reach' | 'book' | 'reviews'; icon: IconName }[] = [
  { key: 'reach', icon: 'mapPin' },
  { key: 'book', icon: 'calendar' },
  { key: 'reviews', icon: 'star' },
];
const PILLS = ['pill1', 'pill2', 'pill3'] as const;
const STEPS = ['s1', 's2', 's3', 's4'] as const;
const REQUIREMENTS = ['license', 'coach', 'health', 'photos'] as const;
const FAQ = ['time', 'rank', 'phone', 'privacy'] as const;

function StartLink({ label }: { label: string }) {
  return (
    <Link href="/business/join" className="wbz-start">
      {label}
    </Link>
  );
}

/**
 * CB-DIR-12 — `W_Dir_Business`: the public for-businesses landing (hero, why, 4 steps, requirements, cooperation terms,
 * FAQ). The cooperation terms are the admin-edited `business_terms` text when published (server-fetched by the route),
 * else a neutral placeholder: no fee is ever stated by the code itself. Under 1024px a signed-in visitor is sent to the app's join flow.
 */
export function BusinessLandingPage({ terms = [] }: { terms?: BusinessTerm[] }) {
  const t = useTranslations('webBusiness');
  return (
    <BusinessShell appHref="/city-services/join">
      <div className="wbz-page">
        <section className="wbz-hero" aria-labelledby="wbz-h1">
          <div className="wbz-hero-copy">
            <span className="wbz-eyebrow">{t('hero.eyebrow')}</span>
            <h1 id="wbz-h1" className="wbz-h1">
              {t('hero.title')}
            </h1>
            <p className="wbz-lead">{t('hero.body')}</p>
            <div className="wbz-hero-actions">
              <StartLink label={t('hero.start')} />
              <a href="#wbz-faq" className="wbz-ghost">
                {t('hero.faq')}
              </a>
            </div>
          </div>
          <aside className="wbz-panel" aria-label={t('hero.panelTitle')}>
            <h2 className="wbz-panel-t">{t('hero.panelTitle')}</h2>
            <p className="wbz-panel-b">{t('hero.panelBody')}</p>
            <ul className="wbz-pills">
              {PILLS.map((k, i) => (
                <li key={k} className="wbz-pill">
                  <Icon name={WHY[i]!.icon} size={16} />
                  {t(`hero.${k}`)}
                </li>
              ))}
            </ul>
          </aside>
        </section>

        <section className="wbz-section" aria-labelledby="wbz-why">
          <h2 id="wbz-why" className="wbz-h2">
            {t('why.title')}
          </h2>
          <div className="wbz-grid3">
            {WHY.map((w) => (
              <article key={w.key} className="wbz-card">
                <span className="wbz-ic">
                  <Icon name={w.icon} size={20} />
                </span>
                <h3 className="wbz-h3">{t(`why.${w.key}.title`)}</h3>
                <p className="wbz-p">{t(`why.${w.key}.body`)}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="wbz-section" aria-labelledby="wbz-steps">
          <h2 id="wbz-steps" className="wbz-h2">
            {t('steps.title')}
          </h2>
          <ol className="wbz-grid4">
            {STEPS.map((s, i) => (
              <li key={s} className="wbz-card wbz-step">
                <span className="wbz-num" aria-hidden="true">
                  {t('steps.number', { n: i + 1 })}
                </span>
                <h3 className="wbz-h3">{t(`steps.${s}.title`)}</h3>
                <p className="wbz-p">{t(`steps.${s}.body`)}</p>
              </li>
            ))}
          </ol>
        </section>

        <div className="wbz-two">
          <section className="wbz-card wbz-block" aria-labelledby="wbz-req">
            <h2 id="wbz-req" className="wbz-h2">
              {t('req.title')}
            </h2>
            <ul className="wbz-checks">
              {REQUIREMENTS.map((r) => (
                <li key={r}>
                  <Icon name="checkCircle" size={18} />
                  <span>{t(`req.${r}`)}</span>
                </li>
              ))}
            </ul>
          </section>
          <section className="wbz-card wbz-block" aria-labelledby="wbz-terms">
            <h2 id="wbz-terms" className="wbz-h2">
              {t('terms.title')}
            </h2>
            {terms.length > 0 ? (
              terms.map((term) => (
                <div key={term.id} className="wbz-term">
                  {term.heading ? <h3 className="wbz-h3">{term.heading}</h3> : null}
                  {term.body ? <p className="wbz-p wbz-term-body">{term.body}</p> : null}
                </div>
              ))
            ) : (
              <>
                <p className="wbz-p">{t('terms.body')}</p>
                <p className="wbz-note">
                  <Icon name="info" size={16} />
                  <span>{t('terms.note')}</span>
                </p>
              </>
            )}
          </section>
        </div>

        <section className="wbz-section" id="wbz-faq" aria-labelledby="wbz-faq-h">
          <h2 id="wbz-faq-h" className="wbz-h2">
            {t('faq.title')}
          </h2>
          <div className="wbz-faq">
            {FAQ.map((k) => (
              <details key={k} className="wbz-qa">
                <summary>
                  <span>{t(`faq.${k}.q`)}</span>
                  <Icon name="chevronDown" size={18} />
                </summary>
                <p className="wbz-p">{t(`faq.${k}.a`)}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="wbz-cta" aria-labelledby="wbz-cta-h">
          <h2 id="wbz-cta-h" className="wbz-h2">
            {t('cta.title')}
          </h2>
          <p className="wbz-p">{t('cta.body')}</p>
          <StartLink label={t('cta.start')} />
        </section>
      </div>
    </BusinessShell>
  );
}
