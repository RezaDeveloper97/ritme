import { describe, expect, it } from 'vitest';

import { infoSectionsSchema } from './schema';

/** A box shaped like the backend's, with a fully filled contact button. */
const box = {
  id: 5,
  heading: 'پشتیبانی',
  body: 'اگه سؤالی داری برامون بنویس.',
  link_label: 'ارسال ایمیل',
  link_url: 'mailto:support@ritmesalamat.com',
};

describe('infoSectionsSchema', () => {
  it('maps boxes onto the camelCase domain shape', () => {
    expect(infoSectionsSchema.parse({ group: 'help', sections: [box] })).toEqual([
      {
        id: 5,
        heading: 'پشتیبانی',
        body: 'اگه سؤالی داری برامون بنویس.',
        linkLabel: 'ارسال ایمیل',
        linkUrl: 'mailto:support@ritmesalamat.com',
      },
    ]);
  });

  it('drops a half-filled link so no unusable button is rendered', () => {
    const [urlOnly, labelOnly] = infoSectionsSchema.parse({
      sections: [
        { ...box, link_label: null },
        { ...box, id: 6, link_url: null },
      ],
    });

    expect(urlOnly).toMatchObject({ linkLabel: null, linkUrl: null });
    expect(labelOnly).toMatchObject({ linkLabel: null, linkUrl: null });
  });

  it('tolerates absent link fields and missing copy', () => {
    expect(infoSectionsSchema.parse({ sections: [{ id: 1 }] })).toEqual([
      { id: 1, heading: '', body: '', linkLabel: null, linkUrl: null },
    ]);
  });

  it('reads an empty screen as an empty list', () => {
    expect(infoSectionsSchema.parse({ group: 'about' })).toEqual([]);
  });
});
