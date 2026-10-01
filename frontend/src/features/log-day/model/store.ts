import { create } from 'zustand';

/**
 * The day the open log sheet is editing (API date, `YYYY-MM-DD`). Client UI state only: the sheet's
 * heading (rendered by the sheet host, outside the content) reads it to show «شنبه ۱۲ مهر · روز ۲۵ سیکل».
 */
interface LogDayState {
  date: string | null;
  setDate: (date: string | null) => void;
}

export const useLogDayStore = create<LogDayState>((set) => ({
  date: null,
  setDate: (date) => set({ date }),
}));
