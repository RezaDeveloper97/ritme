'use client';

import { useTranslations } from 'next-intl';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { Button, CheckboxGrid, Icon, Panel, Select, Switch, TextInput, TranslatableField } from '@/shared/ui';

import { daysOf, MODES, newTask, PHASES, START_RULES, tasksOnDay, type ProgramDraft, type TaskDraft } from '../lib/program';

/**
 * Challenges v2 editor (nbl_Admin_Challenges): duration, start rule, reminder, eligibility include / exclude,
 * per-day tasks, reward (badge + Plus days) and the completion rule. Off = the legacy one-day challenge.
 */
export function ProgramSection({
  draft,
  onChange,
  error,
}: {
  draft: ProgramDraft;
  onChange: (next: ProgramDraft) => void;
  error: unknown;
}) {
  const t = useTranslations('challenges.program');
  const set = <K extends keyof ProgramDraft>(key: K, value: ProgramDraft[K]) => onChange({ ...draft, [key]: value });
  const err = (name: string) => fieldError(error, name);
  const modeOptions = MODES.map((m) => ({ value: m, label: t(`modes.${m}`) }));
  const phaseOptions = PHASES.map((p) => ({ value: p, label: t(`phases.${p}`) }));
  const setTask = (key: string, patch: Partial<TaskDraft>) =>
    set(
      'tasks',
      draft.tasks.map((x) => (x.key === key ? { ...x, ...patch } : x)),
    );

  return (
    <Panel title={t('title')}>
      <div className="flex flex-col gap-5">
        <Switch label={t('enable')} checked={draft.enabled} onChange={(v) => set('enabled', v)} />
        <p className="field-hint m-0">{t('enableHint')}</p>
        {draft.enabled ? (
          <>
            <div className="form-grid">
              <TextInput
                label={t('duration')}
                type="number"
                min={1}
                max={366}
                value={draft.duration}
                onChange={(e) => set('duration', e.target.value)}
                error={err('duration_days')}
                required
              />
              <Select
                label={t('startRule')}
                value={draft.startRule}
                onChange={(e) => set('startRule', e.target.value)}
                options={START_RULES.map((r) => ({ value: r, label: t(`startRules.${r}`) }))}
                error={err('start_rule')}
              />
              {draft.startRule === 'fixed_date' ? (
                <TextInput
                  label={t('startsOn')}
                  type="date"
                  value={draft.startsOn}
                  onChange={(e) => set('startsOn', e.target.value)}
                  error={err('starts_on')}
                />
              ) : null}
              <TextInput
                label={t('reminder')}
                hint={t('reminderHint')}
                type="time"
                value={draft.reminder}
                onChange={(e) => set('reminder', e.target.value)}
                error={err('reminder_time')}
              />
            </div>

            <section aria-labelledby="elig-title" className="flex flex-col gap-3">
              <h3 id="elig-title" className="panel-title m-0">
                {t('eligibility')}
              </h3>
              <p className="field-hint m-0">{t('eligibilityHint')}</p>
              <div className="form-grid">
                <CheckboxGrid label={t('modesInclude')} options={modeOptions} value={draft.modesInclude} onChange={(v) => set('modesInclude', v)} />
                <CheckboxGrid label={t('modesExclude')} options={modeOptions} value={draft.modesExclude} onChange={(v) => set('modesExclude', v)} />
                <CheckboxGrid label={t('phasesInclude')} options={phaseOptions} value={draft.phasesInclude} onChange={(v) => set('phasesInclude', v)} />
                <CheckboxGrid label={t('phasesExclude')} options={phaseOptions} value={draft.phasesExclude} onChange={(v) => set('phasesExclude', v)} />
              </div>
            </section>

            <section aria-labelledby="tasks-title" className="flex flex-col gap-3">
              <h3 id="tasks-title" className="panel-title m-0">
                {t('tasks')}
              </h3>
              <p className="field-hint m-0">{t('tasksHint')}</p>
              {err('tasks') ? (
                <span className="field-error" role="alert">
                  {err('tasks')}
                </span>
              ) : null}
              {draft.tasks.map((task, i) => (
                <div key={task.key} className="flex flex-col gap-2 rounded-lg border border-line p-3">
                  <div className="flex items-end gap-2">
                    <span className="field-label m-0 w-6 shrink-0 pb-2 text-muted">{i + 1}</span>
                    <TextInput
                      label={t('dayFrom')}
                      className="w-28"
                      type="number"
                      min={1}
                      value={task.dayFrom}
                      onChange={(e) => setTask(task.key, { dayFrom: e.target.value })}
                      error={err(`tasks.${i}.day_from`)}
                    />
                    <TextInput
                      label={t('dayTo')}
                      className="w-28"
                      type="number"
                      min={1}
                      value={task.dayTo}
                      onChange={(e) => setTask(task.key, { dayTo: e.target.value })}
                      error={err(`tasks.${i}.day_to`)}
                    />
                    <div className="flex-1" />
                    <Button
                      size="sm"
                      variant="ghost"
                      icon
                      aria-label={t('removeTask')}
                      onClick={() => set('tasks', draft.tasks.filter((x) => x.key !== task.key))}
                    >
                      <Icon name="close" size={15} />
                    </Button>
                  </div>
                  <TranslatableField
                    name={`tasks.${i}.title`}
                    label={t('taskTitle')}
                    value={task.title}
                    onChange={(v) => setTask(task.key, { title: v })}
                    required
                    maxLength={255}
                    errors={fieldErrorsOf(error)}
                  />
                  {err(`tasks.${i}.title`) ? (
                    <span className="field-error" role="alert">
                      {err(`tasks.${i}.title`)}
                    </span>
                  ) : null}
                </div>
              ))}
              <div>
                <Button size="sm" onClick={() => set('tasks', [...draft.tasks, newTask('1', draft.duration)])}>
                  <Icon name="plus" size={15} />
                  {t('addTask')}
                </Button>
              </div>
              <DayOverview draft={draft} />
            </section>

            <section aria-labelledby="reward-title" className="flex flex-col gap-3">
              <h3 id="reward-title" className="panel-title m-0">
                {t('reward')}
              </h3>
              <div className="form-grid">
                <TextInput
                  label={t('badgeCode')}
                  hint={t('badgeCodeHint')}
                  value={draft.badgeCode}
                  maxLength={64}
                  onChange={(e) => set('badgeCode', e.target.value)}
                  error={err('reward_badge_code')}
                />
                <TextInput
                  label={t('plusDays')}
                  type="number"
                  min={0}
                  max={365}
                  value={draft.plusDays}
                  onChange={(e) => set('plusDays', e.target.value)}
                  error={err('reward_plus_days')}
                />
              </div>
              <TranslatableField
                name="reward_badge_title"
                label={t('badgeTitle')}
                value={draft.badgeTitle}
                onChange={(v) => set('badgeTitle', v)}
                maxLength={255}
                errors={fieldErrorsOf(error)}
              />
            </section>

            <section aria-labelledby="rule-title" className="flex flex-col gap-3">
              <h3 id="rule-title" className="panel-title m-0">
                {t('completion')}
              </h3>
              <TextInput
                label={t('minDays')}
                hint={t('minDaysHint')}
                type="number"
                min={1}
                className="max-w-xs"
                value={draft.minDays}
                onChange={(e) => set('minDays', e.target.value)}
                error={err('completion_min_days')}
              />
            </section>
          </>
        ) : null}
      </div>
    </Panel>
  );
}

/** A compact day strip: how many tasks each day of the run carries (days with none can never be completed). */
function DayOverview({ draft }: { draft: ProgramDraft }) {
  const t = useTranslations('challenges.program');
  const days = daysOf(draft.duration);
  if (days.length === 0 || days.length > 60) return null;
  return (
    <div>
      <span className="field-label">{t('perDay')}</span>
      <ol className="m-0 flex list-none flex-wrap gap-1 p-0">
        {days.map((d) => {
          const n = tasksOnDay(draft.tasks, d).length;
          return (
            <li
              key={d}
              className="grid min-h-11 min-w-11 place-items-center rounded-md border border-line px-1 text-xs"
              data-empty={n === 0 || undefined}
              title={t('dayTasks', { day: d, count: n })}
            >
              <span className="font-semibold">{d}</span>
              <span className="text-muted">{n}</span>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
