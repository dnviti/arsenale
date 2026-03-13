import { useState, useEffect, useCallback, type RefObject } from 'react';

/**
 * Tracks fullscreen state scoped to a specific container element.
 * Attaches a single `fullscreenchange` listener per component instance, but only
 * reports `isFullscreen = true` when the element that entered fullscreen is the
 * provided container (or a descendant). This prevents background tabs from
 * receiving spurious `isFullscreen = true` updates when another tab goes fullscreen.
 */
export function useFullscreen(containerRef: RefObject<HTMLElement | null>): [boolean, () => void] {
  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    const onFsChange = () => {
      const el = document.fullscreenElement;
      setIsFullscreen(
        !!el &&
        containerRef.current != null &&
        (containerRef.current === el || containerRef.current.contains(el)),
      );
    };
    document.addEventListener('fullscreenchange', onFsChange);
    return () => document.removeEventListener('fullscreenchange', onFsChange);
  }, [containerRef]);

  const toggleFullscreen = useCallback(() => {
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {});
    } else {
      containerRef.current?.requestFullscreen().catch(() => {});
    }
  }, [containerRef]);

  return [isFullscreen, toggleFullscreen];
}
