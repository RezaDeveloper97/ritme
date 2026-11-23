import type { LogCategory, LogParam, LogTaxonomy } from '@/entities/health-log';
import type { ManifestTarget, PilotValueType } from '@/shared/pilot';

import { isGraded, pickableOptions } from './draft';

/**
 * The log sheet's autopilot names (frontend/CLAUDE.md §11.6). The ids are built from taxonomy codes —
 * the same `category` / `param` / option `value` the API and the voice parser already speak — so a field
 * the registry gains tomorrow is addressable without a change here.
 */
export const logPilotId = {
  /** The «ذخیره» button of the footer. */
  save: 'log.save',
  /** A category's accordion. */
  category: (category: string) => `log.cat:${category}`,
  /** «جزئیات بیشتر» inside a category. */
  more: (category: string) => `log.more:${category}`,
  /** One param's control (a chip row, a number, a note, a switch). */
  param: (category: string, param: string) => `log.param:${category}.${param}`,
  /** The severity row of a graded `items` param. */
  level: (category: string, param: string) => `log.level:${category}.${param}`,
  /** The link that opens a detail panel (`pain`). */
  panel: (kind: string) => `log.panel:${kind}`,
  /** One control inside a detail panel, or of the inline bleeding form. */
  panelParam: (category: string, param: string) => `log.panel-param:${category}.${param}`,
  /** The footer button of a detail panel («تأیید» / «ذخیره»). */
  panelConfirm: 'log.panel-confirm',
  /** A quick tile (`tileKey` from /logs/preferences). */
  tile: (key: string) => `log.tile:${key}`,
  /** «دستی» / «صوتی». */
  tab: (tab: string) => `log.tab:${tab}`,
  /** The search box above the category list. */
  search: 'log.search',
  /** A day of the date strip (`YYYY-MM-DD`). */
  date: (date: string) => `log.date:${date}`,
};

const VALUE_TYPE: Partial<Record<LogParam['type'], PilotValueType>> = {
  single: 'single',
  multi: 'multi',
  items: 'multi',
  number: 'number',
  integer: 'integer',
  text: 'text',
  bool: 'bool',
};

/**
 * Params without a `log.param` control: pain details live in the pain panel. Bleeding and measurements
 * have no panel — their accordion is the whole form — except the bleeding clots row, one choice over
 * `clots` + `clot_size` catalogued as `log.panel-param:bleeding.clots`.
 */
function inPanel(category: LogCategory, param: LogParam): boolean {
  if (category.code === 'bleeding') return param.code === 'clots' || param.code === 'clot_size';
  if (category.code === 'pain') return param.detail || param.code === 'relief';
  return false;
}

/**
 * What the agent may set in the log sheet, for the manifest: one target per param the accordions render,
 * with its option codes and bounds. `text_items` and `link` params are left out (no single value to set).
 */
export function logTargetsOf(taxonomy: LogTaxonomy, mode: string | null = taxonomy.mode): ManifestTarget[] {
  const targets: ManifestTarget[] = [];
  for (const category of taxonomy.categories) {
    targets.push({ id: logPilotId.category(category.code), kind: 'section', label: category.label });
    for (const param of category.params) {
      const kind = VALUE_TYPE[param.type];
      if (!kind || inPanel(category, param)) continue;
      const options = pickableOptions(param, mode).map((o) => ({ value: o.value, label: o.label }));
      const choice = kind === 'single' || kind === 'multi';
      // A chip row with nothing to pick (custom items not created yet) is not something to set.
      if (choice && options.length === 0) continue;
      targets.push({
        id: logPilotId.param(category.code, param.code),
        kind,
        label: param.label,
        group: category.label,
        ...(choice ? { options } : {}),
        ...(param.range ? { min: param.range.min, max: param.range.max } : {}),
        ...(param.unit ? { unit: param.unit.label } : {}),
        ...(kind === 'text' && param.maxLength ? { maxLength: param.maxLength } : {}),
      });
      if (isGraded(param)) {
        targets.push({
          id: logPilotId.level(category.code, param.code),
          kind: 'single',
          label: param.label,
          group: category.label,
          options: param.levels.filter((l) => l.value !== 'no').map((l) => ({ value: l.value, label: l.label })),
        });
      }
    }
  }
  return targets;
}
