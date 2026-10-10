'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { formatDateTime, localInputToApi } from '@/shared/lib';
import { Badge, Button, Panel, Select, Switch, TextInput, TranslatableField, toast, useNotifyError } from '@/shared/ui';

import { useDoctorWrite, type Doctor, type DoctorOptions, type Rule, type VisitType } from '../api/telemed';
import { rialsToToman, tomanToRials } from '../lib/format';

/** Visit types, weekly availability and time off of one saved doctor (each panel saves on its own). */
export function DoctorSchedule({ doctor, options }: { doctor: Doctor; options: DoctorOptions }) {
  return (
    <div className="flex flex-col gap-5">
      <VisitTypesPanel doctor={doctor} options={options} />
      <AvailabilityPanel doctor={doctor} options={options} />
      <TimeOffPanel doctor={doctor} />
    </div>
  );
}

interface VtDraft {
  enabled: boolean;
  duration: string;
  toman: string;
  active: boolean;
  note: Record<string, string>;
  address: Record<string, string>;
}

function VisitTypesPanel({ doctor, options }: { doctor: Doctor; options: DoctorOptions }) {
  const t = useTranslations('telemed.schedule');
  const tm = useTranslations('telemed.doctors');
  const notifyError = useNotifyError();
  const write = useDoctorWrite(doctor.id);
  const initial = (mode: string): VtDraft => {
    const vt: VisitType | undefined = doctor.visit_types.find((v) => v.mode === mode);
    return {
      enabled: Boolean(vt),
      duration: vt ? String(vt.duration_minutes) : '30',
      toman: vt ? String(rialsToToman(vt.price_rials)) : '0',
      active: vt?.is_active ?? true,
      note: vt?.note ?? {},
      address: vt?.address ?? {},
    };
  };
  const [drafts, setDrafts] = useState<Record<string, VtDraft>>(() => Object.fromEntries(options.modes.map((m) => [m, initial(m)])));
  const draftOf = (mode: string): VtDraft => drafts[mode] ?? initial(mode);
  const patch = (mode: string, p: Partial<VtDraft>) => setDrafts((d) => ({ ...d, [mode]: { ...(d[mode] ?? initial(mode)), ...p } }));

  const submit = () =>
    write.mutate(
      {
        method: 'put',
        path: 'visit-types',
        body: {
          visit_types: options.modes
            .filter((m) => draftOf(m).enabled)
            .map((m) => ({
              mode: m,
              duration_minutes: Number(draftOf(m).duration) || 0,
              price_rials: tomanToRials(draftOf(m).toman),
              is_active: draftOf(m).active,
              note: Object.keys(draftOf(m).note).length ? draftOf(m).note : null,
              address: Object.keys(draftOf(m).address).length ? draftOf(m).address : null,
            })),
        },
      },
      { onSuccess: () => toast.success(t('saved')), onError: notifyError },
    );

  return (
    <Panel title={t('visitTypes')}>
      <div className="flex flex-col gap-4">
        <p className="m-0 text-sm text-ink-3">{t('visitTypesHint')}</p>
        {options.modes.map((m) => {
          const d = draftOf(m);
          return (
            <fieldset key={m} className="flex flex-col gap-3 rounded-lg border border-line p-3">
              <legend className="px-1 font-semibold">{tm(`mode.${m}` as 'mode.video')}</legend>
              <Switch label={t('offerMode')} checked={d.enabled} onChange={(v) => patch(m, { enabled: v })} />
              {d.enabled ? (
                <>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <TextInput label={t('duration')} type="number" value={d.duration} onChange={(e) => patch(m, { duration: e.target.value })} />
                    <TextInput label={t('price')} hint={t('priceHint')} type="number" value={d.toman} onChange={(e) => patch(m, { toman: e.target.value })} />
                  </div>
                  {m === 'in_person' ? (
                    <TranslatableField name={`visit_types.${options.modes.filter((x) => draftOf(x).enabled).indexOf(m)}.address`} label={t('address')} value={d.address} onChange={(v) => patch(m, { address: v })} kind="textarea" />
                  ) : null}
                  <Switch label={t('modeActive')} checked={d.active} onChange={(v) => patch(m, { active: v })} />
                </>
              ) : null}
            </fieldset>
          );
        })}
        <div>
          <Button variant="primary" loading={write.isPending} onClick={submit}>
            {t('save')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}

interface RuleDraft {
  weekday: string;
  start: string;
  end: string;
  slot: string;
  modes: string[] | null;
}

function AvailabilityPanel({ doctor, options }: { doctor: Doctor; options: DoctorOptions }) {
  const t = useTranslations('telemed.schedule');
  const notifyError = useNotifyError();
  const write = useDoctorWrite(doctor.id);
  const toDraft = (r: Rule): RuleDraft => ({ weekday: String(r.weekday), start: r.start_time, end: r.end_time, slot: String(r.slot_minutes), modes: r.modes });
  const [rules, setRules] = useState<RuleDraft[]>(doctor.availability.map(toDraft));
  const patch = (i: number, p: Partial<RuleDraft>) => setRules((rs) => rs.map((r, j) => (i === j ? { ...r, ...p } : r)));

  const submit = () =>
    write.mutate(
      {
        method: 'put',
        path: 'availability',
        body: {
          rules: rules.map((r) => ({ weekday: Number(r.weekday), start_time: r.start, end_time: r.end, slot_minutes: Number(r.slot) || 0, modes: r.modes })),
        },
      },
      { onSuccess: () => toast.success(t('saved')), onError: notifyError },
    );

  const weekdays = Array.from({ length: 7 }, (_, d) => ({ value: String(d), label: t(`weekdays.${d}` as 'weekdays.0') }));

  return (
    <Panel
      title={t('availability')}
      actions={
        <Button size="sm" disabled={rules.length >= (options.limits.rules ?? 50)} onClick={() => setRules((rs) => [...rs, { weekday: '6', start: '09:00', end: '12:00', slot: '30', modes: null }])}>
          {t('addRule')}
        </Button>
      }
    >
      <div className="flex flex-col gap-3">
        <p className="m-0 text-sm text-ink-3">{t('availabilityHint')}</p>
        {rules.length === 0 ? <p className="m-0 text-sm text-muted">{t('noRules')}</p> : null}
        {rules.map((r, i) => (
          <div key={i} className="grid items-end gap-3 sm:grid-cols-[1.4fr_1fr_1fr_1fr_auto]">
            <Select label={t('weekday')} value={r.weekday} onChange={(e) => patch(i, { weekday: e.target.value })} options={weekdays} />
            <TextInput label={t('from')} type="time" value={r.start} dir="ltr" onChange={(e) => patch(i, { start: e.target.value })} />
            <TextInput label={t('to')} type="time" value={r.end} dir="ltr" onChange={(e) => patch(i, { end: e.target.value })} />
            <TextInput label={t('slot')} type="number" value={r.slot} onChange={(e) => patch(i, { slot: e.target.value })} />
            <Button variant="ghost" onClick={() => setRules((rs) => rs.filter((_, j) => j !== i))}>
              {t('remove')}
            </Button>
          </div>
        ))}
        <div>
          <Button variant="primary" loading={write.isPending} onClick={submit}>
            {t('save')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}

function TimeOffPanel({ doctor }: { doctor: Doctor }) {
  const t = useTranslations('telemed.schedule');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const write = useDoctorWrite(doctor.id);
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [note, setNote] = useState('');

  const add = () =>
    write.mutate(
      { method: 'post', path: 'time-off', body: { starts_at: localInputToApi(from), ends_at: localInputToApi(to), note: note.trim() || null } },
      {
        onSuccess: () => {
          setFrom('');
          setTo('');
          setNote('');
          toast.success(t('saved'));
        },
        onError: notifyError,
      },
    );

  return (
    <Panel title={t('timeOff')}>
      <div className="flex flex-col gap-3">
        {doctor.time_off.length === 0 ? <p className="m-0 text-sm text-muted">{t('noTimeOff')}</p> : null}
        <ul className="m-0 flex list-none flex-col gap-2 p-0">
          {doctor.time_off.map((o) => (
            <li key={o.id} className="flex flex-wrap items-center gap-2">
              <Badge>{formatDateTime(o.starts_at, locale)}</Badge>
              <span aria-hidden="true">→</span>
              <Badge>{formatDateTime(o.ends_at, locale)}</Badge>
              {o.note ? <span className="text-ink-3" dir="auto">{o.note}</span> : null}
              <Button size="sm" variant="ghost" onClick={() => write.mutate({ method: 'delete', path: `time-off/${o.id}` }, { onError: notifyError })}>
                {t('remove')}
              </Button>
            </li>
          ))}
        </ul>
        <div className="grid items-end gap-3 sm:grid-cols-[1fr_1fr_1.4fr_auto]">
          <TextInput label={t('from')} type="datetime-local" value={from} dir="ltr" onChange={(e) => setFrom(e.target.value)} />
          <TextInput label={t('to')} type="datetime-local" value={to} dir="ltr" onChange={(e) => setTo(e.target.value)} />
          <TextInput label={t('note')} value={note} maxLength={190} onChange={(e) => setNote(e.target.value)} />
          <Button variant="primary" disabled={!from || !to} loading={write.isPending} onClick={add}>
            {t('addTimeOff')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}
