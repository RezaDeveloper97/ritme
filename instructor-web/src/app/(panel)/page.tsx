'use client';

import { DashboardScreen } from '@/screens/dashboard';

import { usePanelInstructor } from './panel-context';

export default function DashboardPage() {
  return <DashboardScreen instructor={usePanelInstructor()} />;
}
