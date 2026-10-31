// Public API of features/rule-conditions (reused by the challenge editor, T-M10-13).
export { ConditionBuilder } from './ui/ConditionBuilder';
export { useFacts, factDefSchema } from './api/facts';
export type { FactDef, FactRegistry, FactType } from './api/facts';
export {
  emptyConditions,
  emptyCond,
  fromStored,
  toStored,
  opsFor,
  inputKind,
  parseValue,
  formatValue,
  factsUsed,
  usesFertilityFact,
  onFactChange,
  summarize,
} from './model/conditions';
export type { CondForm, ConditionsForm, Condition, Group } from './model/conditions';
