'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatDateTime, formatNumber, toIntOrNull } from '@/shared/lib';
import { Badge, Button, LoadGate, PageHeader, Panel, Select, TextArea, TextInput, confirm, toast, useNotifyError } from '@/shared/ui';

import { useCityServiceOptions, type CityServiceOptions } from '../api/city-services';
import { applicationsApi, useDocumentLink, type Application } from '../api/directory';

import { APP_TONE } from './CityServiceApplicationsScreen';

/**
 * /city-services/applications/:id — one business application (CB-DIR-05): every step the applicant filled, the
 * photos, private documents (each view = a fresh 5-minute signed link, audited), and the decision: start review, ask
 * for more information, decline (note required; photos are deleted) or approve → a published, verified place.
 */
export function CityServiceApplicationScreen({ id }: { id: number }) {
  const t = useTranslations('cityServices.dir.apps');
  const detail = applicationsApi.useDetail(id);
  const options = useCityServiceOptions();
  return (
    <LoadGate queries={[detail, options]} header={<PageHeader title={t('detailTitle')} backHref="/city-services/applications" backLabel={t('backToList')} />}>
      {() => (detail.data && options.data ? <ApplicationView app={detail.data.application} options={options.data} /> : null)}
    </LoadGate>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-muted">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

function ApplicationView({ app, options }: { app: Application; options: CityServiceOptions }) {
  const t = useTranslations('cityServices.dir.apps');
  const tw = useTranslations('cityServices.dir');
  const tm = useTranslations('cityServices');
  const locale = useLocale();
  const localize = useLocalized();
  const chip = (list: CityServiceOptions['age_ranges'], code: string) => {
    const it = list.find((x) => x.code === code);
    return it ? localize(it.name) || code : code;
  };
  const city = options.cities.find((c) => c.id === app.city_id);
  const cats = app.category_slugs.map((s) => {
    const c = options.categories.find((x) => x.slug === s);
    return c ? localize(c.name) || s : s;
  });

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={app.business_name ?? t('detailTitle')} backHref="/city-services/applications" backLabel={t('backToList')} />
      <Panel
        title={
          <span className="flex items-center gap-2">
            {t('detailTitle')} <Badge tone={APP_TONE[app.status] ?? 'neutral'}>{t(`statusName.${app.status}` as 'statusName.submitted')}</Badge>
          </span>
        }
        actions={
          app.listing_id ? (
            <Link href={`/city-services/${app.listing_id}`} className="btn btn-sm">
              {t('openPlace')}
            </Link>
          ) : null
        }
      >
        <dl className="form-grid">
          <Row label={t('trackingCode')}>
            <span dir="ltr">{app.tracking_code ?? '—'}</span>
          </Row>
          <Row label={t('submittedAt')}>{app.submitted_at ? formatDateTime(app.submitted_at, locale) : '—'}</Row>
          <Row label={t('categories')}>{cats.join('، ') || '—'}</Row>
          <Row label={t('city')}>{[city ? localize(city.name) || city.slug : '—', app.district].filter(Boolean).join(' · ')}</Row>
          <Row label={t('contact')}>
            {app.contact_hidden ? (
              <span className="text-muted">{t('contactHidden')}</span>
            ) : (
              <span className="flex flex-col">
                <span>{app.contact_name ?? '—'}</span>
                <span dir="ltr" className="text-xs text-muted">
                  {app.contact_phone ?? ''}
                </span>
              </span>
            )}
          </Row>
          <Row label={t('bookingMode')}>{app.booking_mode ? tm(`modes.${app.booking_mode}` as 'modes.link') : '—'}</Row>
          {app.review_note ? <Row label={t('reviewNote')}>{app.review_note}</Row> : null}
        </dl>
      </Panel>

      <Panel title={t('place')}>
        <dl className="flex flex-col gap-3">
          <Row label={t('address')}>{app.address ?? '—'}</Row>
          <Row label={t('map')}>
            <span dir="ltr">{app.lat && app.lng ? `${app.lat}, ${app.lng}` : '—'}</span>
          </Row>
          <Row label={t('about')}>{app.about ?? '—'}</Row>
          <Row label={t('ageRanges')}>{(app.age_ranges ?? []).map((c) => chip(options.age_ranges, c)).join('، ') || '—'}</Row>
          <Row label={t('amenities')}>{(app.amenities ?? []).map((c) => chip(options.amenities, c)).join('، ') || '—'}</Row>
          <Row label={t('hours')}>
            {(app.weekly_hours ?? []).length === 0
              ? '—'
              : (app.weekly_hours ?? []).map((h) => (
                  <span key={`${h.day}-${h.open}`} className="block">
                    {tw(`weekday.${h.day}` as 'weekday.0')}: <span dir="ltr">{h.open}–{h.close}</span>
                  </span>
                ))}
          </Row>
        </dl>
      </Panel>

      <Panel title={t('services')}>
        {(app.services ?? []).length === 0 ? (
          <p className="panel-note">—</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {(app.services ?? []).map((s, i) => (
              <li key={`${s.name}-${i}`} className="flex flex-col">
                <span>
                  {s.name} <Badge tone="brand">{tw(`services.kindName.${s.kind}` as 'services.kindName.session')}</Badge>
                </span>
                <span className="text-xs text-muted">
                  {[
                    s.duration_minutes ? t('duration', { minutes: formatNumber(s.duration_minutes, locale) }) : null,
                    s.age_min_months != null || s.age_max_months != null
                      ? t('ageMonths', { min: formatNumber(s.age_min_months ?? 0, locale), max: formatNumber(s.age_max_months ?? 0, locale) })
                      : null,
                    s.price_toman != null ? t('price', { price: formatNumber(s.price_toman, locale) }) : null,
                  ]
                    .filter(Boolean)
                    .join(' · ')}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Panel>

      <Panel title={t('photos')}>
        {(app.photos ?? []).length === 0 ? (
          <p className="panel-note">{t('noPhotos')}</p>
        ) : (
          <ul className="flex flex-wrap gap-3" aria-label={t('photos')}>
            {(app.photos ?? []).map((p) => (
              <li key={p.id}>
                {/* eslint-disable-next-line @next/next/no-img-element -- API-hosted public file */}
                <img src={p.url} alt="" className="thumb" loading="lazy" />
              </li>
            ))}
          </ul>
        )}
      </Panel>

      <DocumentsPanel app={app} />
      {app.allowed_moves.length > 0 || app.can_approve ? <DecisionPanel app={app} options={options} /> : null}
    </div>
  );
}

function DocumentsPanel({ app }: { app: Application }) {
  const t = useTranslations('cityServices.dir.apps');
  const notifyError = useNotifyError();
  const link = useDocumentLink(app.id);
  const open = (slot: string) =>
    link.mutate(slot, {
      onSuccess: (r) => {
        window.open(r.url, '_blank', 'noopener,noreferrer');
      },
      onError: notifyError,
    });
  // Without the support capability the API lists no documents and refuses links (403): say so instead of buttons.
  if (app.contact_hidden || !app.documents?.length) {
    return (
      <Panel title={t('documents')}>
        <p className="panel-note">{app.contact_hidden ? t('documentsHidden') : t('documentsNote')}</p>
      </Panel>
    );
  }
  return (
    <Panel title={t('documents')}>
      <p className="panel-note">{t('documentsNote')}</p>
      <ul className="flex flex-col gap-2">
        {app.documents.map((d) => (
          <li key={d.slot} className="flex items-center justify-between gap-3">
            <span>{t(`slot.${d.slot}` as 'slot.licence')}</span>
            {d.present ? (
              <Button size="sm" loading={link.isPending && link.variables === d.slot} onClick={() => open(d.slot)}>
                {t('viewDocument')}
              </Button>
            ) : (
              <span className="text-xs text-muted">{t('missingDocument')}</span>
            )}
          </li>
        ))}
      </ul>
    </Panel>
  );
}

function DecisionPanel({ app, options }: { app: Application; options: CityServiceOptions }) {
  const t = useTranslations('cityServices.dir.apps');
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const review = applicationsApi.useAction('review');
  const approve = applicationsApi.useAction('approve');
  const [note, setNote] = useState('');
  const firstCat = options.categories.find((c) => app.category_slugs.includes(c.slug));
  const [category, setCategory] = useState(String(firstCat?.id ?? options.categories[0]?.id ?? ''));
  const [phone, setPhone] = useState('');

  const move = async (to: string) => {
    if (to === 'rejected' && !(await confirm({ message: t('rejectConfirm'), tone: 'danger', confirmLabel: t('moveTo.rejected') }))) return;
    review.mutate(
      { id: app.id, body: { status: to, note: blankToNull(note) } },
      { onSuccess: () => { setNote(''); toast.success(t('moved')); }, onError: notifyError },
    );
  };
  const publish = () =>
    approve.mutate(
      { id: app.id, body: { category_id: toIntOrNull(category), phone: blankToNull(phone), note: blankToNull(note) } },
      { onSuccess: () => toast.success(t('approved')), onError: notifyError },
    );

  return (
    <Panel title={t('decision')}>
      <div className="flex flex-col gap-4">
        <TextArea label={t('note')} hint={t('noteHint')} value={note} onChange={(e) => setNote(e.target.value)} rows={3} maxLength={500} error={fieldError(review.error, 'note')} />
        {app.allowed_moves.length > 0 ? (
          <div className="row-actions">
            {app.allowed_moves.map((to) => (
              <Button key={to} size="sm" variant={to === 'rejected' ? 'danger' : 'default'} loading={review.isPending} onClick={() => void move(to)}>
                {t(`moveTo.${to}` as 'moveTo.in_review')}
              </Button>
            ))}
          </div>
        ) : null}
        {app.can_approve ? (
          <div className="flex flex-col gap-3">
            <p className="panel-note">{t('approveHint')}</p>
            <div className="form-grid">
              <Select
                label={t('approveCategory')}
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                options={options.categories.map((c) => ({ value: String(c.id), label: localize(c.name) || c.slug }))}
                error={fieldError(approve.error, 'category_id')}
              />
              <TextInput label={t('approvePhone')} hint={t('approvePhoneHint')} value={phone} onChange={(e) => setPhone(e.target.value)} dir="ltr" maxLength={32} error={fieldError(approve.error, 'phone')} />
            </div>
            <div>
              <Button variant="primary" loading={approve.isPending} onClick={publish}>
                {t('approve')}
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </Panel>
  );
}
