import type { ReactElement } from 'react';

export interface TransitionProps {
  children?: ReactElement;
  direction?: 'down' | 'left' | 'right' | 'up';
  in?: boolean;
}
