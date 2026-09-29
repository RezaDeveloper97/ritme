import { describe, expect, it } from 'vitest';

import { ApiError } from './apiClient';
import {
  getApiErrorCode,
  getApiErrorMessage,
  getApiLimitMessage,
  getApiSaveErrorMessage,
  getApiThrottleMessage,
} from './envelope';

function failed(status: number, data: unknown): ApiError {
  return new ApiError(`Request failed with status code ${status}`, {
    response: { status, data, headers: {} },
    sentToken: null,
  });
}

const LIMIT_BODY = {
  success: false,
  message: 'حداکثر ۱۰۰ دارو می‌توانید ثبت کنید.',
  errors: { limit: ['حداکثر ۱۰۰ دارو می‌توانید ثبت کنید.'] },
  error_code: 'limit_reached',
};

describe('getApiErrorCode', () => {
  it('reads error_code from a JSON body', () => {
    expect(getApiErrorCode(failed(422, LIMIT_BODY))).toBe('limit_reached');
  });

  it('is undefined without a code, for an HTML body, or a non-API error', () => {
    expect(getApiErrorCode(failed(422, { success: false, message: 'x' }))).toBeUndefined();
    expect(getApiErrorCode(failed(502, '<html>bad gateway</html>'))).toBeUndefined();
    expect(getApiErrorCode(new Error('boom'))).toBeUndefined();
    expect(getApiErrorCode(new ApiError('Network error', { code: 'network', sentToken: null }))).toBeUndefined();
  });
});

describe('getApiErrorMessage', () => {
  it('still reads the message and ignores non-JSON bodies', () => {
    expect(getApiErrorMessage(failed(422, LIMIT_BODY))).toBe(LIMIT_BODY.message);
    expect(getApiErrorMessage(failed(502, '<html/>'))).toBeUndefined();
  });
});

describe('getApiLimitMessage', () => {
  it('returns the localized message of a 422 limit_reached', () => {
    expect(getApiLimitMessage(failed(422, LIMIT_BODY))).toBe(LIMIT_BODY.message);
  });

  it('falls back to errors.limit when message is empty', () => {
    const body = { ...LIMIT_BODY, message: '  ', errors: { limit: ['Limit reached.'] } };
    expect(getApiLimitMessage(failed(422, body))).toBe('Limit reached.');
  });

  it('ignores ordinary validation errors and other statuses', () => {
    expect(
      getApiLimitMessage(failed(422, { success: false, message: 'bad', errors: { title: ['required'] } })),
    ).toBeUndefined();
    expect(getApiLimitMessage(failed(429, { ...LIMIT_BODY }))).toBeUndefined();
    expect(getApiLimitMessage(failed(500, '<html/>'))).toBeUndefined();
    expect(getApiLimitMessage(new Error('boom'))).toBeUndefined();
  });

  it('is undefined when the body carries no usable text', () => {
    expect(getApiLimitMessage(failed(422, { success: false, error_code: 'limit_reached' }))).toBeUndefined();
  });
});

const THROTTLE_BODY = {
  success: false,
  message: 'یه لحظه صبر کن و دوباره ذخیره کن؛ توی این یک دقیقه تغییرهای زیادی فرستاده شده.',
  error_code: 'too_many_requests',
  retry_after: 42,
};

describe('getApiThrottleMessage', () => {
  it("returns the write throttle's localized message", () => {
    expect(getApiThrottleMessage(failed(429, THROTTLE_BODY))).toBe(THROTTLE_BODY.message);
  });

  it("ignores the framework 429 and every other failure", () => {
    expect(getApiThrottleMessage(failed(429, { message: 'Too Many Attempts.' }))).toBeUndefined();
    expect(getApiThrottleMessage(failed(422, { ...THROTTLE_BODY }))).toBeUndefined();
    expect(getApiThrottleMessage(failed(429, { ...THROTTLE_BODY, message: ' ' }))).toBeUndefined();
    expect(getApiThrottleMessage(new Error('boom'))).toBeUndefined();
  });
});

describe('getApiSaveErrorMessage', () => {
  it('prefers the cap, then the throttle message, then the fallback', () => {
    expect(getApiSaveErrorMessage(failed(422, LIMIT_BODY), 'x')).toBe(LIMIT_BODY.message);
    expect(getApiSaveErrorMessage(failed(429, THROTTLE_BODY), 'x')).toBe(THROTTLE_BODY.message);
    expect(getApiSaveErrorMessage(failed(429, { message: 'Too Many Attempts.' }), 'x')).toBe('x');
    expect(getApiSaveErrorMessage(failed(500, '<html/>'), 'x')).toBe('x');
  });
});
