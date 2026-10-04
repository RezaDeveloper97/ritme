import {
  BIRTH_RANGES,
  type BirthField,
  type Child,
  type ChildDeliveryType,
  type ChildInput,
  type ChildSex,
  MAX_CHILD_AGE_YEARS,
  MAX_CHILD_NAME,
  parseDecimalInput,
} from '@/entities/child';
import { diffInDays, fromApiDate } from '@/shared/lib/date';

/** Form state: decimals stay as typed text (canonical ASCII) until submit. */
export interface ChildFormState {
  name: string;
  birthDate: string | null;
  sex: ChildSex | null;
  deliveryType: ChildDeliveryType | null;
  weightKg: string;
  lengthCm: string;
  headCm: string;
}

export const EMPTY_FORM: ChildFormState = {
  name: '',
  birthDate: null,
  sex: null,
  deliveryType: null,
  weightKg: '',
  lengthCm: '',
  headCm: '',
};

export type FormProblem =
  | { field: 'name'; key: 'nameRequired' | 'nameTooLong' }
  | { field: 'birthDate'; key: 'dateRequired' | 'future' | 'tooOld' }
  | { field: BirthField; key: 'range' };

const FIELD_OF: Record<BirthField, 'weightKg' | 'lengthCm' | 'headCm'> = {
  weightKg: 'weightKg',
  lengthCm: 'lengthCm',
  headCm: 'headCm',
};

function numberOf(text: string): number | null {
  return parseDecimalInput(text).value ?? null;
}

/** Client mirror of `validateChild` (the API decides; its 422 is shown per field too). */
export function validateChildForm(state: ChildFormState, today: string): FormProblem[] {
  const problems: FormProblem[] = [];
  const name = state.name.trim();
  if (name === '') problems.push({ field: 'name', key: 'nameRequired' });
  else if (Array.from(name).length > MAX_CHILD_NAME) problems.push({ field: 'name', key: 'nameTooLong' });

  if (!state.birthDate) problems.push({ field: 'birthDate', key: 'dateRequired' });
  else {
    const age = diffInDays(fromApiDate(today), fromApiDate(state.birthDate));
    if (age < 0) problems.push({ field: 'birthDate', key: 'future' });
    else if (age > MAX_CHILD_AGE_YEARS * 366) problems.push({ field: 'birthDate', key: 'tooOld' });
  }

  for (const field of Object.keys(BIRTH_RANGES) as BirthField[]) {
    const v = numberOf(state[FIELD_OF[field]]);
    const { min, max } = BIRTH_RANGES[field];
    if (v !== null && (v < min || v > max)) problems.push({ field, key: 'range' });
  }
  return problems;
}

export function toChildInput(state: ChildFormState): ChildInput {
  return {
    name: state.name.trim(),
    birthDate: state.birthDate ?? '',
    sex: state.sex,
    deliveryType: state.deliveryType,
    birthWeightKg: numberOf(state.weightKg),
    birthLengthCm: numberOf(state.lengthCm),
    birthHeadCm: numberOf(state.headCm),
  };
}

const text = (v: number | null): string => (v == null ? '' : String(v));

/** Edit mode: the stored child as form state. */
export function formFromChild(child: Pick<Child, 'name' | 'birthDate' | 'sex' | 'deliveryType' | 'birth'>): ChildFormState {
  return {
    name: child.name,
    birthDate: child.birthDate,
    sex: child.sex,
    deliveryType: child.deliveryType,
    weightKg: text(child.birth.weightKg),
    lengthCm: text(child.birth.lengthCm),
    headCm: text(child.birth.headCm),
  };
}
