import { describe, expect, it } from 'vitest';

import { ALL_MODULES_ON, modeAvailable, moduleOfRoute } from './modules';

describe('app modules', () => {
  it('maps each module route to its switch', () => {
    expect(moduleOfRoute('/companion')).toBe('companion');
    expect(moduleOfRoute('/companions/12')).toBe('companion');
    expect(moduleOfRoute('/onboarding/partner/linked')).toBe('companion');
    expect(moduleOfRoute('/fertility/bbt')).toBe('ttc');
    expect(moduleOfRoute('/analysis/fertility')).toBe('ttc');
    expect(moduleOfRoute('/ivf/meds/3')).toBe('ivf');
    expect(moduleOfRoute('/menopause/score')).toBe('menopause');
    expect(moduleOfRoute('/teen/parent')).toBe('teen');
    expect(moduleOfRoute('/postpartum/mood')).toBe('postpartum');
    expect(moduleOfRoute('/children/feeding')).toBe('children');
    expect(moduleOfRoute('/contraception/setup/')).toBe('contraception');
    expect(moduleOfRoute('/programs/pelvic/kegel')).toBe('programs');
    expect(moduleOfRoute('/city-services/12')).toBe('mother_child');
    expect(moduleOfRoute('/places/3')).toBe('mother_child');
    expect(moduleOfRoute('/learn/downloads')).toBe('learning');
    expect(moduleOfRoute('/shop/cart')).toBe('shop');
    expect(moduleOfRoute('/profile/orders')).toBe('shop');
    expect(moduleOfRoute('/insurance/claims')).toBe('insurance');
    expect(moduleOfRoute('/record/export')).toBe('record');
    expect(moduleOfRoute('/profile/devices')).toBe('devices');
    expect(moduleOfRoute('/services/doctors/4/book')).toBe('doctors');
    expect(moduleOfRoute('/services/bookings/9/chat')).toBe('doctors');
    expect(moduleOfRoute('/assistant/chat/7/handoff')).toBe('doctors');
    expect(moduleOfRoute('/todo/lists/2')).toBe('todo');
    expect(moduleOfRoute('/reminders/')).toBe('reminders');
  });

  it('leaves every other route alone', () => {
    for (const path of ['/', '/home', '/pregnancy', '/analysis', '/analysis/cycle', '/companionship', '/childrens', '/onboarding/gender', '/record/emergency', '/profile', '/profile/mode', '/services/home-lab', '/reminders/appointment/new', '/reminders/medication/3', '/assistant/chat/7/summary']) {
      expect(moduleOfRoute(path)).toBeNull();
    }
  });

  it('hides only the switchable modes', () => {
    const off = { ...ALL_MODULES_ON, ttc: false, teen: false };
    expect(modeAvailable(off, 'ttc')).toBe(false);
    expect(modeAvailable(off, 'teen')).toBe(false);
    expect(modeAvailable(off, 'menopause')).toBe(true);
    expect(modeAvailable(off, 'cycle')).toBe(true);
    expect(modeAvailable(off, 'pregnancy')).toBe(true);
  });
});
