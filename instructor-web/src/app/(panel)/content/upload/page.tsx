import { Suspense } from 'react';

import { UploadScreen } from '@/screens/upload';

export default function Page() {
  return (
    <Suspense>
      <UploadScreen />
    </Suspense>
  );
}
