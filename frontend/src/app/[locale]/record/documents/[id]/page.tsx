import { setRequestLocale } from 'next-intl/server';

import { RecordDocumentPage } from '@/screens/record-document';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/record/documents/[id]` — a record document (CB-REC-04, nbl_Rec_Doc). Back header, no bottom nav. */
export default async function RecordDocumentRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = Number(id);
  return (
    <RouteMessages route="recordDocument">
      <RecordDocumentPage id={Number.isInteger(n) && n > 0 ? n : 0} />
    </RouteMessages>
  );
}
