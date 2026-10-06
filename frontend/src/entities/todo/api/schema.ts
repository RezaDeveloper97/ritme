import { z } from 'zod';

import {
  TODO_CATEGORIES,
  TODO_GROUPS,
  type TodoBoard,
  type TodoItem,
  type TodoTask,
  type TodoTaskDetail,
} from '../model/types';

/*
 * Parsers of `/api/v1/todo*` (B-N6-08; schemas in backend-go/api/openapi.yaml,
 * internal/todo/model.go). Snake case → camel case. A task with an unknown
 * category is dropped rather than blanking the board.
 */

const nullableString = z.string().nullable().catch(null);

const taskRaw = z.object({
  id: z.number(),
  title: z.string(),
  note: nullableString,
  category: z.enum(TODO_CATEGORIES),
  due_date: nullableString,
  due_time: nullableString,
  remind: z.boolean().catch(false),
  done: z.boolean().catch(false),
  done_at: nullableString,
  overdue: z.boolean().catch(false),
  suggestion_key: nullableString,
  list: z.object({ total: z.number(), open: z.number() }).nullable().catch(null),
});

function toTask(r: z.infer<typeof taskRaw>): TodoTask {
  return {
    id: r.id,
    title: r.title,
    note: r.note,
    category: r.category,
    dueDate: r.due_date,
    dueTime: r.due_time,
    remind: r.remind,
    done: r.done,
    doneAt: r.done_at,
    overdue: r.overdue,
    suggestionKey: r.suggestion_key,
    list: r.list,
  };
}

/** A list of tasks; rows that do not parse are skipped. */
const tasksSchema = z.array(z.unknown()).transform((rows) =>
  rows.flatMap((row) => {
    const p = taskRaw.safeParse(row);
    return p.success ? [toTask(p.data)] : [];
  }),
);

export const taskSchema = taskRaw.transform(toTask);

const itemRaw = z.object({
  id: z.number(),
  task_id: z.number(),
  title: z.string(),
  due_date: nullableString,
  done: z.boolean().catch(false),
  done_at: nullableString,
});

function toItem(r: z.infer<typeof itemRaw>): TodoItem {
  return { id: r.id, taskId: r.task_id, title: r.title, dueDate: r.due_date, done: r.done, doneAt: r.done_at };
}

export const itemSchema = itemRaw.transform(toItem);

export const boardSchema = z
  .object({
    date: z.string(),
    suggestion: z
      .object({ key: z.string(), prompt: z.string(), action: z.string(), days: z.number() })
      .nullable()
      .catch(null),
    groups: z.array(z.object({ key: z.enum(TODO_GROUPS), date: nullableString, tasks: tasksSchema })),
    done: tasksSchema.catch([]),
    notifications: z.object({ enabled: z.boolean() }).catch({ enabled: true }),
  })
  .transform(
    (r): TodoBoard => ({
      date: r.date,
      suggestion: r.suggestion,
      groups: r.groups,
      done: r.done,
      remindersEnabled: r.notifications.enabled,
    }),
  );

export const detailSchema = z
  .object({ task: taskSchema, items: z.array(itemSchema) })
  .transform((r): TodoTaskDetail => ({ task: r.task, items: r.items }));

export const savedTaskSchema = z.object({ task: taskSchema }).transform((r) => r.task);

export const savedItemSchema = z.object({ item: itemSchema, task: taskSchema });

export const acceptedSchema = z.object({ task: taskSchema, item: itemSchema.nullable() });
