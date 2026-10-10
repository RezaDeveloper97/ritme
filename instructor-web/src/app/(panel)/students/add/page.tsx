import { Suspense } from 'react';

import { AddStudentsScreen } from '@/screens/add-students';

export default function Page() {
  return (
    <Suspense>
      <AddStudentsScreen />
    </Suspense>
  );
}
