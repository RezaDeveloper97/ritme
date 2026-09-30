import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { setCsrfToken } from '@/shared/api';

import { catalogRequests } from './catalog';

type Call = { url: string; method: string; body: unknown };
let calls: Call[];

const item = (over: Record<string, unknown> = {}) => ({
  id: 7,
  group: 'teen_faq',
  code: 'first_period',
  sort_order: 1,
  is_active: true,
  audiences: ['teen'],
  title: { fa: 'اولین پریود', en: 'First period' },
  body: null,
  meta: [],
  needs_review: true,
  created_at: '2026-09-30 10:00:00',
  updated_at: '2026-09-30 10:00:00',
  ...over,
});

beforeEach(() => {
  calls = [];
  setCsrfToken('tok');
  vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
    const body = init.body ? JSON.parse(String(init.body)) : undefined;
    calls.push({ url, method: init.method ?? 'GET', body });
    const data = url.endsWith('/catalog')
      ? { items: [{ group: 'teen_faq', items_count: 2, active_count: 1 }] }
      : init.method === 'DELETE'
        ? { id: 7 }
        : init.method === 'GET' && !/\/\d+$/.test(url.split('?')[0] ?? '')
          ? { items: [item()], meta: { current_page: 1, last_page: 1, per_page: 100, total: 1 }, filters: { group: 'teen_faq', q: '', status: 'all' } }
          : { catalog_item: item(body ? { ...(body as object) } : {}) };
    return new Response(JSON.stringify({ success: true, data }), {
      status: init.method === 'POST' ? 201 : 200,
      headers: { 'content-type': 'application/json' },
    });
  });
});
afterEach(() => vi.unstubAllGlobals());

describe('catalog admin requests (catalog.md §3)', () => {
  it('lists groups and items; decodes null/[] columns', async () => {
    const groups = await catalogRequests.groups();
    expect(groups.items[0]).toEqual({ group: 'teen_faq', items_count: 2, active_count: 1 });
    const page = await catalogRequests.list('teen_faq', { status: 'active', q: '', per_page: 100 });
    expect(calls[1]!.url).toBe('/api/admin/v1/catalog/teen_faq?status=active&per_page=100');
    expect(page.items[0]).toMatchObject({ body: {}, meta: null, audiences: ['teen'] });
  });

  it('creates, reads, updates and deletes one item with CSRF', async () => {
    const created = await catalogRequests.create('teen_faq', { code: 'first_period', title: { fa: 'x' } });
    expect(created.catalog_item.id).toBe(7);
    await catalogRequests.detail('teen_faq', 7);
    await catalogRequests.update('teen_faq', 7, { title: { fa: 'y' }, is_active: false });
    await catalogRequests.remove('teen_faq', 7);
    expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual([
      'POST /api/admin/v1/catalog/teen_faq',
      'GET /api/admin/v1/catalog/teen_faq/7',
      'PUT /api/admin/v1/catalog/teen_faq/7',
      'DELETE /api/admin/v1/catalog/teen_faq/7',
    ]);
    expect(calls[2]!.body).toEqual({ title: { fa: 'y' }, is_active: false });
  });

  it('reorders with one partial PUT per moved row, title included', async () => {
    const rows = [
      { id: 2, sort_order: 2, title: { fa: 'ب' }, is_active: true },
      { id: 1, sort_order: 1, title: { fa: 'الف' }, is_active: false },
      { id: 3, sort_order: 3, title: { fa: 'پ' }, is_active: true },
    ];
    await expect(catalogRequests.reorder('teen_faq', rows)).resolves.toBe(2);
    expect(calls).toEqual([
      { url: '/api/admin/v1/catalog/teen_faq/2', method: 'PUT', body: { title: { fa: 'ب' }, sort_order: 1 } },
      { url: '/api/admin/v1/catalog/teen_faq/1', method: 'PUT', body: { title: { fa: 'الف' }, sort_order: 2 } },
    ]);
  });
});
