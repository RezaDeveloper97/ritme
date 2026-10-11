'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatDateTime, formatNumber, toIntOrNull } from '@/shared/lib';
import { Badge, Button, CheckboxGrid, Panel, Select, Switch, TextArea, TextInput, confirm, toast, useNotifyError } from '@/shared/ui';

import type { CityServiceOptions, Listing } from '../api/city-services';
import { useRemoveFilePhoto, useSaveDirectory, useServiceMutations, useServices, useSlotMutations, useSlots, type PlaceService, type PlaceSlot } from '../api/directory';

const DAYS = [0, 1, 2, 3, 4, 5, 6] as const;

/** CB-DIR-05 panels of a saved place: directory fields, services / packages and slots. */
export function ListingDirectoryPanels({ listing, options }: { listing: Listing; options: CityServiceOptions }) {
  return (
    <>
      <DirectoryPanel listing={listing} options={options} />
      <ServicesPanel listing={listing} options={options} />
      <SlotsPanel listing={listing} />
    </>
  );
}

function DirectoryPanel({ listing, options }: { listing: Listing; options: CityServiceOptions }) {
  const t = useTranslations('cityServices.dir.directory');
  const td = useTranslations('cityServices.dir');
  const locale = useLocale();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = useSaveDirectory(listing.id);
  const removePhoto = useRemoveFilePhoto(listing.id);
  const err = (n: string) => fieldError(save.error, n);

  const [status, setStatus] = useState(listing.status);
  const [verified, setVerified] = useState(listing.verified);
  const [lat, setLat] = useState(listing.lat == null ? '' : String(listing.lat));
  const [lng, setLng] = useState(listing.lng == null ? '' : String(listing.lng));
  const [about, setAbout] = useState(listing.about ?? '');
  const [ages, setAges] = useState<string[]>(listing.age_ranges);
  const [amenities, setAmenities] = useState<string[]>(listing.amenities);
  const [hours, setHours] = useState(() => DAYS.map((d) => listing.weekly_hours.find((h) => h.day === d) ?? null));
  const [website, setWebsite] = useState(listing.website_url ?? '');
  const [cancelHours, setCancelHours] = useState(listing.cancel_free_hours == null ? '' : String(listing.cancel_free_hours));
  const [policy, setPolicy] = useState(listing.cancel_policy ?? '');
  const [rules, setRules] = useState(listing.rules ?? '');

  const setDay = (d: number, next: { day: number; open: string; close: string } | null) => setHours((cur) => cur.map((h, i) => (i === d ? next : h)));
  const submit = () =>
    save.mutate(
      {
        status,
        verified,
        lat: blankToNull(lat) === null ? null : Number(lat),
        lng: blankToNull(lng) === null ? null : Number(lng),
        about: blankToNull(about),
        age_ranges: ages,
        amenities,
        weekly_hours: hours.filter((h): h is { day: number; open: string; close: string } => h !== null),
        website_url: blankToNull(website),
        cancel_free_hours: toIntOrNull(cancelHours),
        cancel_policy: blankToNull(policy),
        rules: blankToNull(rules),
      },
      { onSuccess: () => toast.success(t('saved')), onError: notifyError },
    );
  const chips = (list: CityServiceOptions['age_ranges']) => list.map((c) => ({ value: c.code, label: localize(c.name) || c.code }));

  return (
    <Panel
      title={t('title')}
      actions={
        <span className="text-xs text-muted">
          {listing.rating_count > 0 && listing.rating_avg != null
            ? t('rating', { avg: formatNumber(listing.rating_avg, locale), count: formatNumber(listing.rating_count, locale) })
            : t('noRating')}
        </span>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="form-grid">
          <Select
            label={t('status')}
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            options={options.listing_statuses.map((s) => ({ value: s, label: td(`placeStatus.${s}` as 'placeStatus.published') }))}
            error={err('status')}
          />
          <Switch label={t('verified')} checked={verified} onChange={setVerified} />
          <TextInput label={t('lat')} value={lat} onChange={(e) => setLat(e.target.value)} dir="ltr" inputMode="decimal" error={err('lat')} />
          <TextInput label={t('lng')} value={lng} onChange={(e) => setLng(e.target.value)} dir="ltr" inputMode="decimal" error={err('lng')} />
          <TextInput label={t('website')} value={website} onChange={(e) => setWebsite(e.target.value)} dir="ltr" maxLength={1000} error={err('website_url')} />
          <TextInput label={t('cancelFreeHours')} type="number" value={cancelHours} onChange={(e) => setCancelHours(e.target.value)} error={err('cancel_free_hours')} />
        </div>
        <TextArea label={t('about')} value={about} onChange={(e) => setAbout(e.target.value)} rows={3} maxLength={5000} error={err('about')} />
        <CheckboxGrid label={t('ageRanges')} options={chips(options.age_ranges)} value={ages} onChange={setAges} error={err('age_ranges')} />
        <CheckboxGrid label={t('amenities')} options={chips(options.amenities)} value={amenities} onChange={setAmenities} error={err('amenities')} />
        <fieldset className="flex flex-col gap-2">
          <legend className="text-sm">{t('hours')}</legend>
          {DAYS.map((d) => {
            const h = hours[d];
            return (
              <div key={d} className="flex flex-wrap items-end gap-3">
                <Switch label={td(`weekday.${d}` as 'weekday.0')} checked={h !== null} onChange={(on) => setDay(d, on ? { day: d, open: '09:00', close: '18:00' } : null)} />
                {h ? (
                  <>
                    <TextInput label={t('open')} type="time" value={h.open} onChange={(e) => setDay(d, { ...h, open: e.target.value })} dir="ltr" />
                    <TextInput label={t('close')} type="time" value={h.close} onChange={(e) => setDay(d, { ...h, close: e.target.value })} dir="ltr" />
                  </>
                ) : null}
              </div>
            );
          })}
          {err('weekly_hours') ? <p className="field-error">{err('weekly_hours')}</p> : null}
        </fieldset>
        <TextInput label={t('cancelPolicy')} value={policy} onChange={(e) => setPolicy(e.target.value)} maxLength={300} error={err('cancel_policy')} />
        <TextArea label={t('rules')} value={rules} onChange={(e) => setRules(e.target.value)} rows={3} maxLength={5000} error={err('rules')} />
        {listing.file_photo_urls.length > 0 ? (
          <div className="flex flex-col gap-2">
            <span className="text-sm">{t('filePhotos')}</span>
            <ul className="flex flex-wrap gap-3" aria-label={t('filePhotos')}>
              {listing.file_photo_urls.map((u, i) => {
                const fid = listing.photo_file_ids[i];
                return (
                  <li key={u} className="flex flex-col items-center gap-1">
                    {/* eslint-disable-next-line @next/next/no-img-element -- API-hosted public file */}
                    <img src={u} alt="" className="thumb" loading="lazy" />
                    {fid !== undefined ? (
                      <Button size="sm" loading={removePhoto.isPending && removePhoto.variables === fid} onClick={() => removePhoto.mutate(fid, { onError: notifyError })}>
                        {t('removeFilePhoto')}
                      </Button>
                    ) : null}
                  </li>
                );
              })}
            </ul>
          </div>
        ) : null}
        <div>
          <Button variant="primary" loading={save.isPending} onClick={submit}>
            {t('save')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}

type ServiceDraft = {
  kind: string;
  name: string;
  format: string;
  duration: string;
  ageMin: string;
  ageMax: string;
  capacity: string;
  price: string;
  active: boolean;
};

const draftOf = (s: PlaceService | null): ServiceDraft => ({
  kind: s?.kind ?? 'session',
  name: s?.name ?? '',
  format: s?.format ?? '',
  duration: s?.duration_minutes == null ? '' : String(s.duration_minutes),
  ageMin: s?.age_min_months == null ? '' : String(s.age_min_months),
  ageMax: s?.age_max_months == null ? '' : String(s.age_max_months),
  capacity: s?.capacity == null ? '' : String(s.capacity),
  price: s?.price_toman == null ? '' : String(s.price_toman),
  active: s?.is_active ?? true,
});

function ServicesPanel({ listing, options }: { listing: Listing; options: CityServiceOptions }) {
  const t = useTranslations('cityServices.dir.services');
  const tc = useTranslations('common');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const query = useServices(listing.id);
  const { save, remove } = useServiceMutations(listing.id);
  const [editing, setEditing] = useState<number | null | 'new'>(null);
  const [draft, setDraft] = useState<ServiceDraft>(draftOf(null));
  const err = (n: string) => fieldError(save.error, n);
  const set = (k: keyof ServiceDraft, v: string | boolean) => setDraft((d) => ({ ...d, [k]: v }));

  const startEdit = (s: PlaceService | null) => {
    setDraft(draftOf(s));
    setEditing(s ? s.id : 'new');
    save.reset();
  };
  const submit = () =>
    save.mutate(
      {
        id: editing === 'new' || editing === null ? null : editing,
        body: {
          kind: draft.kind,
          name: draft.name.trim(),
          format: blankToNull(draft.format),
          duration_minutes: toIntOrNull(draft.duration),
          age_min_months: toIntOrNull(draft.ageMin),
          age_max_months: toIntOrNull(draft.ageMax),
          capacity: toIntOrNull(draft.capacity),
          price_toman: toIntOrNull(draft.price),
          is_active: draft.active,
        },
      },
      { onSuccess: () => { setEditing(null); toast.success(t('saved')); }, onError: notifyError },
    );
  const destroy = async (s: PlaceService) => {
    if (!(await confirm({ message: t('deleteConfirm'), tone: 'danger' }))) return;
    remove.mutate(s.id, { onSuccess: () => toast.success(t('deleted')), onError: notifyError });
  };

  return (
    <Panel
      title={t('title')}
      actions={
        <Button size="sm" onClick={() => startEdit(null)}>
          {t('new')}
        </Button>
      }
    >
      <div className="flex flex-col gap-3">
        {(query.data?.items ?? []).length === 0 && !query.isPending ? <p className="panel-note">{t('empty')}</p> : null}
        <ul className="flex flex-col gap-2">
          {(query.data?.items ?? []).map((s) => (
            <li key={s.id} className="flex flex-wrap items-center justify-between gap-2">
              <span className="flex flex-col">
                <span>
                  {s.name} <Badge tone="brand">{t(`kindName.${s.kind}` as 'kindName.session')}</Badge> {s.is_active ? null : <Badge tone="amber">✕</Badge>}
                </span>
                <span className="text-xs text-muted">{t('usageText', { slots: formatNumber(s.slots_count, locale), bookings: formatNumber(s.bookings_count, locale) })}</span>
              </span>
              <span className="row-actions">
                <Button size="sm" onClick={() => startEdit(s)}>
                  {t('edit')}
                </Button>
                <Button size="sm" variant="danger" aria-label={tc('delete')} loading={remove.isPending && remove.variables === s.id} onClick={() => void destroy(s)}>
                  ✕
                </Button>
              </span>
            </li>
          ))}
        </ul>
        {editing !== null ? (
          <div className="flex flex-col gap-3">
            <div className="form-grid">
              <TextInput label={t('name')} value={draft.name} onChange={(e) => set('name', e.target.value)} maxLength={160} required error={err('name')} />
              <Select label={t('kind')} value={draft.kind} onChange={(e) => set('kind', e.target.value)} options={options.service_kinds.map((k) => ({ value: k, label: t(`kindName.${k}` as 'kindName.session') }))} />
              <Select
                label={t('format')}
                value={draft.format}
                onChange={(e) => set('format', e.target.value)}
                options={[{ value: '', label: '—' }, ...options.service_formats.map((f) => ({ value: f, label: t(`formatName.${f}` as 'formatName.group') }))]}
              />
              <TextInput label={t('duration')} type="number" value={draft.duration} onChange={(e) => set('duration', e.target.value)} error={err('duration_minutes')} />
              <TextInput label={t('ageMin')} type="number" value={draft.ageMin} onChange={(e) => set('ageMin', e.target.value)} error={err('age_min_months')} />
              <TextInput label={t('ageMax')} type="number" value={draft.ageMax} onChange={(e) => set('ageMax', e.target.value)} error={err('age_max_months')} />
              <TextInput label={t('capacity')} type="number" value={draft.capacity} onChange={(e) => set('capacity', e.target.value)} error={err('capacity')} />
              <TextInput label={t('price')} type="number" value={draft.price} onChange={(e) => set('price', e.target.value)} error={err('price_toman')} />
              <Switch label={t('active')} checked={draft.active} onChange={(v) => set('active', v)} />
            </div>
            <div className="row-actions">
              <Button variant="primary" size="sm" loading={save.isPending} onClick={submit}>
                {t('save')}
              </Button>
              <Button size="sm" aria-label={tc('cancel')} onClick={() => setEditing(null)}>
                ✕
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </Panel>
  );
}

const todayTehran = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Tehran' }).format(new Date());

function SlotsPanel({ listing }: { listing: Listing }) {
  const t = useTranslations('cityServices.dir.slots');
  const td = useTranslations('cityServices.dir');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const [from, setFrom] = useState(todayTehran);
  const query = useSlots(listing.id, from);
  const services = useServices(listing.id);
  const m = useSlotMutations(listing.id);
  const serviceOptions = [{ value: '', label: t('anyService') }, ...(services.data?.items ?? []).map((s) => ({ value: String(s.id), label: s.name }))];

  // single slot
  const [service, setService] = useState('');
  const [startsAt, setStartsAt] = useState('');
  const [capacity, setCapacity] = useState('1');
  // generator
  const [gFrom, setGFrom] = useState(todayTehran);
  const [gTo, setGTo] = useState(todayTehran);
  const [days, setDays] = useState<string[]>([]);
  const [times, setTimes] = useState('');
  const [duration, setDuration] = useState('');

  const createOne = () =>
    m.create.mutate(
      { service_id: toIntOrNull(service), starts_at: startsAt.trim(), capacity: toIntOrNull(capacity) },
      { onSuccess: () => toast.success(t('created', { count: formatNumber(1, locale) })), onError: notifyError },
    );
  const generate = () =>
    m.generate.mutate(
      {
        service_id: toIntOrNull(service),
        from: gFrom,
        to: gTo,
        weekdays: days.map(Number),
        times: times.split(/[,،]/).map((s) => s.trim()).filter(Boolean),
        duration_minutes: toIntOrNull(duration),
        capacity: toIntOrNull(capacity),
      },
      {
        onSuccess: (r) => toast.success(t('generated', { created: formatNumber(r.created, locale), skipped: formatNumber(r.skipped, locale) })),
        onError: notifyError,
      },
    );
  const svcName = (id: number | null) => services.data?.items.find((s) => s.id === id)?.name ?? t('anyService');

  return (
    <Panel title={t('title')}>
      <div className="flex flex-col gap-4">
        <TextInput label={t('from')} type="date" value={from} onChange={(e) => setFrom(e.target.value)} dir="ltr" />
        {(query.data?.items ?? []).length === 0 && !query.isPending ? <p className="panel-note">{t('empty')}</p> : null}
        <ul className="flex flex-col gap-2">
          {(query.data?.items ?? []).map((s) => (
            <SlotRow key={s.id} slot={s} serviceName={svcName(s.service_id)} update={m.update} remove={m.remove} />
          ))}
        </ul>

        <div className="form-grid">
          <Select label={t('service')} value={service} onChange={(e) => setService(e.target.value)} options={serviceOptions} />
          <TextInput label={t('capacity')} type="number" value={capacity} onChange={(e) => setCapacity(e.target.value)} error={fieldError(m.create.error ?? m.generate.error, 'capacity')} />
        </div>
        <fieldset className="flex flex-col gap-2">
          <legend className="text-sm">{t('single')}</legend>
          <div className="flex flex-wrap items-end gap-3">
            <TextInput label={t('startsAtInput')} value={startsAt} onChange={(e) => setStartsAt(e.target.value)} dir="ltr" placeholder="2026-10-20 10:00" error={fieldError(m.create.error, 'starts_at')} />
            <Button size="sm" variant="primary" loading={m.create.isPending} onClick={createOne}>
              {t('add')}
            </Button>
          </div>
        </fieldset>
        <fieldset className="flex flex-col gap-2">
          <legend className="text-sm">{t('generator')}</legend>
          <div className="form-grid">
            <TextInput label={t('genFrom')} type="date" value={gFrom} onChange={(e) => setGFrom(e.target.value)} dir="ltr" error={fieldError(m.generate.error, 'from')} />
            <TextInput label={t('genTo')} type="date" value={gTo} onChange={(e) => setGTo(e.target.value)} dir="ltr" error={fieldError(m.generate.error, 'to')} />
            <TextInput label={t('times')} hint={t('timesHint')} value={times} onChange={(e) => setTimes(e.target.value)} dir="ltr" error={fieldError(m.generate.error, 'times')} />
            <TextInput label={t('durationOptional')} type="number" value={duration} onChange={(e) => setDuration(e.target.value)} error={fieldError(m.generate.error, 'duration_minutes')} />
          </div>
          <CheckboxGrid
            label={t('weekdays')}
            options={DAYS.map((d) => ({ value: String(d), label: td(`weekday.${d}` as 'weekday.0') }))}
            value={days}
            onChange={setDays}
            error={fieldError(m.generate.error, 'weekdays')}
          />
          <div>
            <Button size="sm" variant="primary" loading={m.generate.isPending} onClick={generate}>
              {t('generate')}
            </Button>
          </div>
        </fieldset>
      </div>
    </Panel>
  );
}

function SlotRow({
  slot,
  serviceName,
  update,
  remove,
}: {
  slot: PlaceSlot;
  serviceName: string;
  update: ReturnType<typeof useSlotMutations>['update'];
  remove: ReturnType<typeof useSlotMutations>['remove'];
}) {
  const t = useTranslations('cityServices.dir.slots');
  const tc = useTranslations('common');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const [cap, setCap] = useState(String(slot.capacity));
  const save = (status: string) =>
    update.mutate({ id: slot.id, body: { capacity: toIntOrNull(cap), status } }, { onSuccess: () => toast.success(t('saved')), onError: notifyError });
  return (
    <li className="flex flex-wrap items-end justify-between gap-3">
      <span className="flex flex-col">
        <span>{formatDateTime(slot.starts_at, locale)}</span>
        <span className="text-xs text-muted">
          {serviceName} · {formatNumber(slot.booked_count, locale)} / {formatNumber(slot.capacity, locale)}{' '}
          <Badge tone={slot.status === 'open' ? 'green' : 'neutral'}>{t(`statusName.${slot.status}` as 'statusName.open')}</Badge>
        </span>
      </span>
      <span className="flex flex-wrap items-end gap-2">
        <TextInput label={t('capacity')} hint={t('capacityHint')} type="number" value={cap} onChange={(e) => setCap(e.target.value)} />
        <Button size="sm" onClick={() => save(slot.status)}>
          {t('save')}
        </Button>
        <Button size="sm" onClick={() => save(slot.status === 'open' ? 'closed' : 'open')}>
          {slot.status === 'open' ? t('close') : t('reopen')}
        </Button>
        {slot.booked_count === 0 ? (
          <Button size="sm" variant="danger" aria-label={tc('delete')} onClick={() => remove.mutate(slot.id, { onSuccess: () => toast.success(t('deleted')), onError: notifyError })}>
            ✕
          </Button>
        ) : null}
      </span>
    </li>
  );
}
