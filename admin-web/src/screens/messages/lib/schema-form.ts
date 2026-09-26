import type { SchemaField } from '../api/messages';

/**
 * Helpers of the schema-driven payload editor (registered groups, admin-api.md §13):
 * an empty value per field, and the draft → request body normalisation (blank
 * nullable texts → null, list lines trimmed, integers parsed).
 */
export type Draft = Record<string, unknown>;

export function emptyValue(field: SchemaField): unknown {
  switch (field.kind) {
    case 'text':
    case 'url':
      return field.nullable ? null : '';
    case 'integer':
      return field.nullable ? null : (field.min ?? 0);
    case 'boolean':
      return false;
    case 'enum':
      return field.nullable ? null : (field.values?.[0] ?? '');
    case 'text_list':
    case 'enum_list':
    case 'object_list':
      return [];
    case 'object':
      return emptyObject(field.fields ?? []);
  }
}

export function emptyObject(fields: readonly SchemaField[]): Draft {
  return Object.fromEntries(fields.map((f) => [f.key, emptyValue(f)]));
}

/** Seed a draft from a stored / template payload: every schema key present, extra keys dropped. */
export function seedDraft(fields: readonly SchemaField[], value: unknown): Draft {
  const src = value && typeof value === 'object' && !Array.isArray(value) ? (value as Draft) : {};
  const out: Draft = {};
  for (const f of fields) {
    const v = src[f.key];
    if (v === undefined) out[f.key] = emptyValue(f);
    else if (f.kind === 'object') out[f.key] = seedDraft(f.fields ?? [], v);
    else if (f.kind === 'object_list') out[f.key] = Array.isArray(v) ? v.map((item) => seedDraft(f.fields ?? [], item)) : [];
    else out[f.key] = v;
  }
  return out;
}

function normalizeValue(field: SchemaField, value: unknown): unknown {
  switch (field.kind) {
    case 'text':
    case 'url': {
      const text = typeof value === 'string' ? value : value === null || value === undefined ? '' : String(value);
      return field.nullable && text.trim() === '' ? null : text;
    }
    case 'integer': {
      if (value === null || value === undefined || value === '') return field.nullable ? null : value;
      const n = Number(value);
      return Number.isInteger(n) ? n : value;
    }
    case 'text_list':
      return (Array.isArray(value) ? value : []).map((v) => (typeof v === 'string' ? v.trim() : v)).filter((v) => v !== '');
    case 'object':
      return normalizeDraft(field.fields ?? [], value);
    case 'object_list':
      return (Array.isArray(value) ? value : []).map((item) => normalizeDraft(field.fields ?? [], item));
    default:
      return value;
  }
}

export function normalizeDraft(fields: readonly SchemaField[], value: unknown): Draft {
  const src = value && typeof value === 'object' && !Array.isArray(value) ? (value as Draft) : {};
  return Object.fromEntries(fields.map((f) => [f.key, normalizeValue(f, src[f.key])]));
}
