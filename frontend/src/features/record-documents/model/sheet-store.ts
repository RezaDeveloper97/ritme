import { create } from 'zustand';

/**
 * Which record-home sheet is open (CB-REC-04). The cards live inside bloom's scroller, but `AppSheet` positions
 * against its nearest positioned ancestor, so the sheets render outside it (`RecordHomeSheets`) and the two halves
 * share this bit of UI state.
 */
export type RecordHomeSheet = 'upload' | 'surgeries' | 'family_history';

interface SheetState {
  sheet: RecordHomeSheet | null;
  open: (sheet: RecordHomeSheet) => void;
  close: () => void;
}

export const useRecordHomeSheet = create<SheetState>((set) => ({
  sheet: null,
  open: (sheet) => set({ sheet }),
  close: () => set({ sheet: null }),
}));
