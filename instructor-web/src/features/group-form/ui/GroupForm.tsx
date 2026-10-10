'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useState, type FormEvent } from 'react';

import { GROUP_NAME_MAX, createGroup, studentKeys, updateGroup, type GroupDetail } from '@/entities/student';
import { useErrorMessage } from '@/shared/i18n';
import { Button, TextField } from '@/shared/ui';

export interface CourseOption {
  id: number;
  title: string;
  published: boolean;
}

interface Props {
  /** Editing when set, creating otherwise. */
  groupId?: number;
  initialName?: string;
  initialCourseIds?: number[];
  courses: CourseOption[];
  onSaved: (detail: GroupDetail) => void;
  onCancel: () => void;
}

/** Create / edit a group: name and the courses it opens (PUT replaces the set). */
export function GroupForm({ groupId, initialName = '', initialCourseIds = [], courses, onSaved, onCancel }: Props) {
  const t = useTranslations('groupForm');
  const errorMessage = useErrorMessage();
  const qc = useQueryClient();
  const [name, setName] = useState(initialName);
  const [ids, setIds] = useState<number[]>(initialCourseIds);
  const [invalid, setInvalid] = useState(false);

  const mutation = useMutation({
    mutationFn: () => {
      const input = { name: name.trim(), course_ids: ids };
      return groupId ? updateGroup(groupId, input) : createGroup(input);
    },
    onSuccess: (detail) => {
      void qc.invalidateQueries({ queryKey: studentKeys.all });
      void qc.invalidateQueries({ queryKey: ['courses'] });
      onSaved(detail);
    },
  });

  function toggle(id: number) {
    setIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]));
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) return setInvalid(true);
    mutation.mutate();
  }

  return (
    <form className="panel-card form-card" onSubmit={submit} noValidate>
      <TextField
        label={t('nameLabel')}
        value={name}
        maxLength={GROUP_NAME_MAX}
        onChange={(e) => {
          setName(e.target.value);
          setInvalid(false);
        }}
        error={invalid ? t('nameRequired') : undefined}
        autoFocus
      />
      <fieldset className="check-group">
        <legend className="field__label">{t('coursesLabel')}</legend>
        {courses.length === 0 ? (
          <p className="field__hint">{t('noCourses')}</p>
        ) : (
          courses.map((c) => (
            <label key={c.id} className="check-row">
              <input type="checkbox" checked={ids.includes(c.id)} onChange={() => toggle(c.id)} />
              <span>{c.title}</span>
              {c.published ? null : <span className="field__hint">{t('draft')}</span>}
            </label>
          ))
        )}
        <p className="field__hint">{t('coursesHint')}</p>
      </fieldset>
      {mutation.isError ? <p className="form-error">{errorMessage(mutation.error)}</p> : null}
      <div className="form-card__actions">
        <Button type="submit" loading={mutation.isPending}>
          {groupId ? t('save') : t('create')}
        </Button>
        <Button variant="text" onClick={onCancel}>
          {t('cancel')}
        </Button>
      </div>
    </form>
  );
}
