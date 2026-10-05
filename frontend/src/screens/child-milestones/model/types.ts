/** One age band of the month chips. */
export interface MilestoneBandRef {
  months: number;
  label: string;
  current: boolean;
}

export interface MilestoneItem {
  code: string;
  title: string;
  domain: string;
  checked: boolean;
  checkedOn: string | null;
}

export interface MilestoneActivity {
  code: string;
  title: string;
  body: string | null;
}

/** GET /children/{id}/milestones?month= (and the PUT response). */
export interface MilestonesView {
  bands: MilestoneBandRef[];
  band: {
    months: number;
    label: string;
    checked: number;
    total: number;
    items: MilestoneItem[];
    activities: MilestoneActivity[];
    doctorNote: string | null;
  };
  intro: string | null;
}
