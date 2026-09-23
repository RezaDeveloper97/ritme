// Public API of shared/api. Import only from '@/shared/api'.
export { api, request, buildUrl, setUnauthorizedHandler } from './client';
export type { Method, QueryValue, RequestOptions } from './client';
export { ApiError, isApiError, isAuthError, CLIENT_ERROR_CODES } from './errors';
export type { FieldErrors } from './errors';
export {
  envelopeSchema,
  listSchema,
  pageMetaSchema,
  optionSchema,
} from './envelope';
export type { ListResult, Option, PageMeta } from './envelope';
export { setCsrfToken, getCsrfToken } from './csrf';
export { createQueryClient } from './query-client';
