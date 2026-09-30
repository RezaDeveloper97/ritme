import { describe, expect, it } from 'vitest';

import { boxesSchema, contactOf, filterFaq, isReportValid } from './support';

const boxes = boxesSchema.parse({
  sections: [
    { id: 1, heading: 'پیش‌بینی پریودم دقیق نیست', body: 'سیکل بیشتری ثبت کن', link_label: null, link_url: null },
    { id: 2, heading: 'تلفن پشتیبانی', body: 'شنبه تا چهارشنبه', link_label: '021 0000', link_url: 'tel:0210000' },
    { id: 3, heading: 'ایمیل', body: 'پاسخ سریع', link_label: 'a@b.c', link_url: 'mailto:a@b.c' },
    { id: 4, heading: 'بد', body: 'x', link_label: 'x', link_url: 'javascript:alert(1)' },
  ],
});

describe('support model', () => {
  it('reads the phone and the chat target by link scheme, ignoring unsafe links', () => {
    const c = contactOf(boxes);
    expect(c.phone?.id).toBe(2);
    expect(c.chat?.id).toBe(3);
    expect(boxes[3]?.linkUrl).toBeNull();
  });

  it('filters the FAQ ignoring ZWNJ and Arabic letter variants', () => {
    expect(filterFaq(boxes, 'پیشبینی').map((b) => b.id)).toEqual([1]);
    expect(filterFaq(boxes, 'ثبت كن').map((b) => b.id)).toEqual([1]);
    expect(filterFaq(boxes, '  ')).toHaveLength(4);
  });

  it('validates the report length', () => {
    expect(isReportValid('short')).toBe(false);
    expect(isReportValid('long enough text')).toBe(true);
  });
});
