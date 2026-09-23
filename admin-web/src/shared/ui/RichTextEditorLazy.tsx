'use client';

import dynamic from 'next/dynamic';

import { Skeleton } from './Skeleton';

/**
 * TipTap is ~180 kB; load it only on screens that render an editor, never in
 * the shared bundle. Client-only (ProseMirror needs the DOM).
 */
export const RichTextEditor = dynamic(() => import('./RichTextEditor').then((m) => m.RichTextEditor), {
  ssr: false,
  loading: () => <Skeleton className="h-48 w-full" />,
});
