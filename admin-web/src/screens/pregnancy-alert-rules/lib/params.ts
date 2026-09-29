import type { ParamField } from '../api/alert-rules';

/**
 * The `params` body of PUT /pregnancy-alert-rules/:key, in schema order:
 * integers typed; an empty optional integer (e.g. `weight_missing_week.from_weekday`)
 * is null, so the engine default applies; everything else as edited.
 */
export function paramsBody(schema: ParamField[], params: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    schema.map((f) => {
      const v = params[f.key];
      if (f.kind !== 'integer') return [f.key, v];
      if (v === '' || v === null || v === undefined) return [f.key, f.nullable ? null : v];
      return [f.key, Number(v)];
    }),
  );
}
