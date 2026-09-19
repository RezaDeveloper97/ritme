import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import type { ReactNode } from 'react';

import { pickNamespaces } from '@/shared/i18n';

import { type MessageRoute, ROUTE_NAMESPACES } from './message-scopes';

/**
 * Server component: gives a route's screen only the message namespaces it
 * uses (`ROUTE_NAMESPACES[route]`), instead of the whole bundle.
 *
 * It nests inside the layout's shell provider and *replaces* its messages
 * within the subtree — next-intl providers don't merge — which is why each
 * route list names shared namespaces such as `common` and `nav` itself.
 * Locale, time zone and `now` are filled in by next-intl's server provider,
 * exactly as in the layout.
 */
export async function RouteMessages({
  route,
  children,
}: {
  route: MessageRoute;
  children: ReactNode;
}) {
  const messages = await getMessages();
  return (
    <NextIntlClientProvider messages={pickNamespaces(messages, ROUTE_NAMESPACES[route])}>
      {children}
    </NextIntlClientProvider>
  );
}
