'use client';

import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, ApiError, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { TodoBoard, TodoItem, TodoTask, TodoTaskDetail, TodoTaskInput } from '../model/types';
import { todoKeys } from './keys';
import { acceptedSchema, boardSchema, detailSchema, savedItemSchema, savedTaskSchema } from './schema';

/*
 * `/api/v1/todo*` (B-N6-08), Go only. Titles may carry health data (CLAUDE.md
 * §11): never log a payload or a response; ids are the only thing in a URL.
 */

export async function fetchTodoBoard(): Promise<TodoBoard> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/todo');
  return boardSchema.parse(data.data);
}

/** GET /todo — today / tomorrow / later, the recently done and the cycle suggestion. */
export function useTodoBoard() {
  return useQuery({ queryKey: todoKeys.board(), queryFn: fetchTodoBoard, enabled: isAuthenticated(), staleTime: 30_000, retry: 1 });
}

export async function fetchTodoTask(id: number): Promise<TodoTaskDetail> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/todo/tasks/${id}`);
  return detailSchema.parse(data.data);
}

/** GET /todo/tasks/{id} — one task with its list items (Todo_List). */
export function useTodoTask(id: number | null) {
  return useQuery({
    queryKey: todoKeys.task(id ?? 0),
    queryFn: () => fetchTodoTask(id ?? 0),
    enabled: isAuthenticated() && !!id,
    staleTime: 30_000,
    retry: (count, error) => !(error instanceof ApiError && error.response?.status === 404) && count < 1,
  });
}

/** Snake-case body of POST / PUT /todo/tasks. */
export function toTaskBody(input: TodoTaskInput): Record<string, unknown> {
  return {
    title: input.title.trim(),
    note: input.note?.trim() ? input.note.trim() : null,
    category: input.category,
    due_date: input.dueDate,
    due_time: input.dueTime,
    remind: input.remind,
  };
}

function invalidateAll(qc: QueryClient) {
  void qc.invalidateQueries({ queryKey: todoKeys.all });
}

/** POST /todo/tasks. */
export function useCreateTodoTask() {
  const qc = useQueryClient();
  return useMutation<TodoTask, unknown, TodoTaskInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/todo/tasks', toTaskBody(input));
      return savedTaskSchema.parse(data.data);
    },
    onSuccess: () => invalidateAll(qc),
  });
}

/** PUT /todo/tasks/{id} with the whole form. */
export function useUpdateTodoTask() {
  const qc = useQueryClient();
  return useMutation<TodoTask, unknown, { id: number; input: TodoTaskInput }>({
    mutationFn: async ({ id, input }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/todo/tasks/${id}`, toTaskBody(input));
      return savedTaskSchema.parse(data.data);
    },
    onSuccess: () => invalidateAll(qc),
  });
}

function patchBoardTask(board: TodoBoard | undefined, id: number, done: boolean): TodoBoard | undefined {
  if (!board) return board;
  const patch = (t: TodoTask) => (t.id === id ? { ...t, done, overdue: done ? false : t.overdue } : t);
  return { ...board, groups: board.groups.map((g) => ({ ...g, tasks: g.tasks.map(patch) })), done: board.done.map(patch) };
}

/** PUT /todo/tasks/{id} {done} — ticked at once on the board (rolled back on failure). */
export function useToggleTodoTask() {
  const qc = useQueryClient();
  return useMutation<TodoTask, unknown, { id: number; done: boolean }, { prev?: TodoBoard }>({
    mutationFn: async ({ id, done }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/todo/tasks/${id}`, { done });
      return savedTaskSchema.parse(data.data);
    },
    onMutate: async ({ id, done }) => {
      await qc.cancelQueries({ queryKey: todoKeys.board() });
      const prev = qc.getQueryData<TodoBoard>(todoKeys.board());
      qc.setQueryData<TodoBoard | undefined>(todoKeys.board(), (b) => patchBoardTask(b, id, done));
      return { prev };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(todoKeys.board(), ctx.prev);
    },
    onSettled: () => invalidateAll(qc),
  });
}

/** DELETE /todo/tasks/{id} (its items go with it). */
export function useDeleteTodoTask() {
  const qc = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/todo/tasks/${id}`);
    },
    onSuccess: (_d, id) => {
      qc.removeQueries({ queryKey: todoKeys.task(id) });
      void qc.invalidateQueries({ queryKey: todoKeys.board() });
    },
  });
}

/** POST /todo/tasks/{id}/items. */
export function useAddTodoItem(taskId: number) {
  const qc = useQueryClient();
  return useMutation<TodoItem, unknown, { title: string; dueDate?: string | null }>({
    mutationFn: async ({ title, dueDate }) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/todo/tasks/${taskId}/items`, {
        title: title.trim(),
        ...(dueDate ? { due_date: dueDate } : {}),
      });
      return savedItemSchema.parse(data.data).item;
    },
    onSuccess: () => invalidateAll(qc),
  });
}

function patchDetailItem(d: TodoTaskDetail | undefined, id: number, done: boolean): TodoTaskDetail | undefined {
  if (!d) return d;
  return { ...d, items: d.items.map((it) => (it.id === id ? { ...it, done } : it)) };
}

/** PUT /todo/tasks/{id}/items/{item} {done} — ticked at once in the list (rolled back on failure). */
export function useToggleTodoItem(taskId: number) {
  const qc = useQueryClient();
  return useMutation<TodoItem, unknown, { id: number; done: boolean }, { prev?: TodoTaskDetail }>({
    mutationFn: async ({ id, done }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/todo/tasks/${taskId}/items/${id}`, { done });
      return savedItemSchema.parse(data.data).item;
    },
    onMutate: async ({ id, done }) => {
      await qc.cancelQueries({ queryKey: todoKeys.task(taskId) });
      const prev = qc.getQueryData<TodoTaskDetail>(todoKeys.task(taskId));
      qc.setQueryData<TodoTaskDetail | undefined>(todoKeys.task(taskId), (d) => patchDetailItem(d, id, done));
      return { prev };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(todoKeys.task(taskId), ctx.prev);
    },
    onSettled: () => invalidateAll(qc),
  });
}

/** DELETE /todo/tasks/{id}/items/{item}. */
export function useDeleteTodoItem(taskId: number) {
  const qc = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/todo/tasks/${taskId}/items/${id}`);
    },
    onSuccess: () => invalidateAll(qc),
  });
}

/** POST /todo/suggestions/{key}/accept — the server recomputes and adds it (task or list item). */
export function useAcceptTodoSuggestion() {
  const qc = useQueryClient();
  return useMutation<{ task: TodoTask; item: TodoItem | null }, unknown, string>({
    mutationFn: async (key) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/todo/suggestions/${encodeURIComponent(key)}/accept`);
      return acceptedSchema.parse(data.data);
    },
    onSettled: () => invalidateAll(qc),
  });
}

/** POST /todo/suggestions/{key}/dismiss — hidden at once, not offered again for this period. */
export function useDismissTodoSuggestion() {
  const qc = useQueryClient();
  return useMutation<void, unknown, string, { prev?: TodoBoard }>({
    mutationFn: async (key) => {
      await apiClient.post(`/todo/suggestions/${encodeURIComponent(key)}/dismiss`);
    },
    onMutate: async () => {
      await qc.cancelQueries({ queryKey: todoKeys.board() });
      const prev = qc.getQueryData<TodoBoard>(todoKeys.board());
      qc.setQueryData<TodoBoard | undefined>(todoKeys.board(), (b) => (b ? { ...b, suggestion: null } : b));
      return { prev };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(todoKeys.board(), ctx.prev);
    },
    onSettled: () => invalidateAll(qc),
  });
}

/** Validation messages of a 422, field → first message. */
export function todoFieldErrors(error: unknown): Record<string, string> {
  if (!(error instanceof ApiError)) return {};
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const errors = body && typeof body === 'object' ? body.errors : undefined;
  const out: Record<string, string> = {};
  if (errors && typeof errors === 'object') {
    for (const [k, v] of Object.entries(errors as Record<string, unknown>)) {
      if (Array.isArray(v) && typeof v[0] === 'string') out[k] = v[0];
    }
  }
  return out;
}

/** Whether an error is the uniform 404 of a foreign / deleted task. */
export function isTodoNotFound(error: unknown): boolean {
  return error instanceof ApiError && error.response?.status === 404;
}
