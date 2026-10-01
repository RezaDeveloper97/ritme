import { describe, expect, it } from 'vitest';

import type { LogCategory, LogParam } from '@/entities/health-log';

import type { VoiceSuggestion } from '../api/voice';
import { mergeSuggestions, mergeValue, paramKey } from './merge';

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
  cat('sleep', [param('quality', 'single')]),
  cat('measurements', [param('weight', 'number')]),
  cat('pain2', [param('none', 'bool')]),
];

function s(category: string, p: string, item: string | null, value: VoiceSuggestion['value']): VoiceSuggestion {
  return { category, param: p, item, value, confidence: 0.9, label: `${category}.${p}` };
}

describe('voice merge', () => {
  it('adds to what is already logged and replaces scalars', () => {
    const values = {
      pain: { location: { head: { level: 'mild', score: 2 } } },
      mood: { moods: ['calm'] },
      sleep: { quality: 'poor' },
    };
    const merged = mergeSuggestions(
      values,
      [
        s('pain', 'location', 'abdomen', 'moderate'),
        s('mood', 'moods', 'bored', true),
        s('mood', 'moods', 'calm', true),
        s('mood', 'moods', 'sad', true),
        s('sleep', 'quality', null, 'good'),
        s('measurements', 'weight', null, 58.5),
        s('pain2', 'none', null, true),
      ],
      categories,
    );
    expect(merged).toEqual([
      { category: 'pain', param: 'location', value: { head: { level: 'mild', score: 2 }, abdomen: { level: 'moderate', score: null } } },
      { category: 'mood', param: 'moods', value: ['calm', 'bored', 'sad'] },
      { category: 'sleep', param: 'quality', value: 'good' },
      { category: 'measurements', param: 'weight', value: 58.5 },
      { category: 'pain2', param: 'none', value: true },
    ]);
  });

  it('keeps a pain score when the level agrees and skips what the sheet does not show', () => {
    expect(mergeValue('items', s('pain', 'location', 'head', 'mild'), { head: { level: 'mild', score: 3 } })).toEqual({
      head: { level: 'mild', score: 3 },
    });
    expect(mergeSuggestions({}, [s('bleeding', 'flow', null, 'heavy')], categories)).toEqual([]);
    expect(mergeValue('multi', s('mood', 'moods', null, true), undefined)).toBeNull();
    expect(mergeValue('number', s('measurements', 'weight', null, 'x'), undefined)).toBeNull();
    expect(mergeValue('text', s('note', 'text', null, 'hi'), undefined)).toBeNull();
  });

  it('builds the voice_params key', () => {
    expect(paramKey({ category: 'pain', param: 'location' })).toBe('pain.location');
  });
});
