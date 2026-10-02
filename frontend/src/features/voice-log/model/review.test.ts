import { describe, expect, it } from 'vitest';

import type { LogCategory, LogParam } from '@/entities/health-log';

import type { VoiceResult, VoiceSuggestion } from '../api/voice';
import {
  chooseItem,
  commitItems,
  groupItems,
  hasHeavyBleeding,
  logSuggestions,
  revalueItem,
  reviewItems,
  savedRowText,
  splitLabel,
  statusCategories,
  toggleItem,
} from './review';

function param(code: string, type: LogParam['type']): LogParam {
  return {
    code,
    type,
    label: code,
    modes: ['cycle'],
    detail: false,
    alert: false,
    options: [],
    levels: [],
    score: null,
    range: null,
    scale: null,
    unit: null,
    maxLength: null,
    dynamic: false,
    source: null,
  };
}

function cat(code: string, params: LogParam[]): LogCategory {
  return { code, label: code, group: { value: 'body', label: 'body' }, modes: ['cycle'], conditions: {}, params };
}

const categories = [
  cat('pain', [param('location', 'items')]),
  cat('mood', [param('moods', 'multi')]),
  cat('symptoms', [param('general', 'items')]),
  cat('bleeding', [param('flow', 'single')]),
  cat('sleep', [param('quality', 'single')]),
];

function s(partial: Partial<VoiceSuggestion> & Pick<VoiceSuggestion, 'category' | 'param'>): VoiceSuggestion {
  return { target: 'log', item: null, value: true, confidence: 0.9, label: partial.category, options: [], ...partial };
}

const result: VoiceResult = {
  transcript: 'x',
  language: 'fa',
  mode: 'menopause',
  suggestions: [
    s({ category: 'pain', param: 'location', item: 'abdomen', value: 'moderate', label: 'Abdomen pain · Moderate' }),
    s({
      category: 'mood',
      param: 'moods',
      item: 'bored',
      label: 'Unmotivated',
      options: [
        { category: 'mood', param: 'moods', item: 'sad', value: true, label: 'Sad' },
        { category: 'symptoms', param: 'general', item: 'fatigue', value: 'yes', label: 'Fatigue' },
        { category: 'hidden', param: 'x', item: null, value: true, label: 'Hidden' },
      ],
    }),
    s({ target: 'hot_flash', category: 'hot_flash', param: 'count', value: 3, label: 'Hot flashes · 3' }),
    s({ category: 'not_in_sheet', param: 'x', label: 'Dropped' }),
    s({ category: 'bleeding', param: 'flow', value: 'heavy', label: 'Flow · Heavy' }),
  ],
};

describe('voice review', () => {
  it('keeps log items the sheet shows and every diary item; drops options it cannot show', () => {
    const items = reviewItems(result, categories);
    expect(items.map((i) => i.choice.label)).toEqual(['Abdomen pain · Moderate', 'Unmotivated', 'Hot flashes · 3', 'Flow · Heavy']);
    expect(items[1].choices.map((c) => c.label)).toEqual(['Unmotivated', 'Sad', 'Fatigue']);
    expect(groupItems(items).map((g) => g.key)).toEqual(['log:pain', 'log:mood', 'hot_flash', 'log:bleeding']);
  });

  it('splits the save between the day PUT and the diary commit', () => {
    let items = reviewItems(result, categories);
    items = chooseItem(items, items[1].id, 2);
    items = toggleItem(items, items[0].id);
    expect(logSuggestions(items).map((x) => `${x.category}.${x.param}.${x.item}`)).toEqual([
      'symptoms.general.fatigue',
      'bleeding.flow.null',
    ]);
    expect(commitItems(items)).toEqual([{ category: 'hot_flash', param: 'count', value: 3 }]);
  });

  it('flags heavy bleeding until it is re-valued or left out', () => {
    const items = reviewItems(result, categories);
    const flow = items[3];
    expect(hasHeavyBleeding(items)).toBe(true);
    expect(hasHeavyBleeding(revalueItem(items, flow.id, 'medium', 'Flow · Medium'))).toBe(false);
    expect(hasHeavyBleeding(toggleItem(items, flow.id))).toBe(false);
  });

  it('splits chip labels and picks the status rows per mode', () => {
    expect(splitLabel('Abdomen pain · Moderate')).toEqual({ title: 'Abdomen pain', sub: 'Moderate' });
    expect(splitLabel('Night-time hot flashes')).toEqual({ title: 'Night-time hot flashes', sub: null });
    expect(savedRowText({ key: 'a', category: 'mood', label: 'Bored', group: 'Mood' })).toEqual({ title: 'Mood', sub: 'Bored' });
    expect(savedRowText({ key: 'b', category: 'pain', label: 'Head · Mild', group: 'Pain' })).toEqual({ title: 'Head', sub: 'Mild' });
    expect(statusCategories('cycle', categories).map((c) => c.code)).toEqual(['bleeding', 'pain', 'sleep']);
    expect(statusCategories('menopause', categories).map((c) => c.code)).toEqual(['symptoms', 'sleep', 'bleeding']);
  });
});
