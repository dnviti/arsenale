import { useRef } from 'react';

/**
 * Returns true once `trigger` has been truthy at least once.
 * Used to defer mounting lazy-loaded components until first needed,
 * while keeping them mounted afterwards to preserve exit animations.
 *
 * Uses a ref instead of useState to avoid cascading re-renders when
 * many instances are used in the same component (e.g. MainLayout).
 * The parent already re-renders when the trigger prop changes, so
 * no extra setState is needed.
 */
export function useLazyMount(trigger: unknown): boolean {
  const mounted = useRef(false);
  if (trigger && !mounted.current) {
    mounted.current = true;
  }
  return mounted.current;
}
