import { z } from 'zod';

/** `{success, message?, data}` — admin-api.md §2. */
export function envelopeSchema<T extends z.ZodTypeAny>(data: T) {
  return z.object({
    success: z.literal(true),
    message: z.string().optional(),
    data,
  });
}

export const errorBodySchema = z.object({
  success: z.literal(false).optional(),
  message: z.string().optional(),
  error_code: z.string().optional(),
  errors: z.record(z.string(), z.array(z.string())).optional(),
  retry_after: z.number().optional(),
});

export const pageMetaSchema = z.object({
  current_page: z.number().int(),
  last_page: z.number().int(),
  per_page: z.number().int(),
  total: z.number().int(),
});
export type PageMeta = z.infer<typeof pageMetaSchema>;

/** `{items, meta, filters}` — admin-api.md §5. */
export function listSchema<I extends z.ZodTypeAny, F extends z.ZodTypeAny>(
  item: I,
  filters: F,
) {
  return z.object({ items: z.array(item), meta: pageMetaSchema, filters });
}

export type ListResult<I, F = Record<string, unknown>> = {
  items: I[];
  meta: PageMeta;
  filters: F;
};

/** `{value, label}` options (labels in the default language). */
export const optionSchema = z.object({ value: z.string(), label: z.string() });
export type Option = z.infer<typeof optionSchema>;
