import type { ReactNode } from 'react';

import { Icon } from './Icon';

interface ErrorStateProps {
  title: string;
  message?: string;
  action?: ReactNode;
}

export function ErrorState({ title, message, action }: ErrorStateProps) {
  return (
    <div className="state-card" role="alert">
      <span className="state-card__icon state-card__icon--danger">
        <Icon name="alert" />
      </span>
      <p className="state-card__title">{title}</p>
      {message ? <p className="state-card__text">{message}</p> : null}
      {action}
    </div>
  );
}
