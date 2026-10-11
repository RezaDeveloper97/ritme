import { describe, expect, it } from 'vitest';

import { centresSchema } from './schema';

describe('centresSchema', () => {
  it('parses string coordinates and nulls', () => {
    const rows = centresSchema.parse({
      centres: [
        { id: 1, name: 'A', type: 'lab', services: null, city: 'Tehran', address: null, phone: '021', lat: '35.7001', lng: '51.4', is_direct: true, insurer_name: null },
        { id: 2, name: 'B', type: 'gyn', lat: null, lng: 'x', is_direct: false },
      ],
    });
    expect(rows[0]).toMatchObject({ lat: 35.7001, lng: 51.4, isDirect: true, services: null });
    expect(rows[1]).toMatchObject({ lat: null, lng: null, isDirect: false });
  });
});
