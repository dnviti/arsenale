import {
  createContext,
  useEffect,
  useState,
  forwardRef,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
} from 'react';
import { createPortal } from 'react-dom';
import {
  Dialog as ShadDialog,
  DialogContent as ShadDialogContent,
  DialogDescription,
  DialogFooter as ShadDialogFooter,
  DialogHeader as ShadDialogHeader,
  DialogTitle as ShadDialogTitle,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';
import { resolveSx, useTheme, type SxProp } from './theme';

interface CommonProps {
  [key: string]: any;
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
  sx?: SxProp;
}

function useSxStyle(sx: CommonProps['sx']) {
  const theme = useTheme();
  return resolveSx(theme, sx);
}

type Origin = {
  horizontal: 'left' | 'center' | 'right';
  vertical: 'top' | 'center' | 'bottom';
};

function originOffset(origin: Origin, width: number, height: number) {
  return {
    x: origin.horizontal === 'left' ? 0 : origin.horizontal === 'center' ? width / 2 : width,
    y: origin.vertical === 'top' ? 0 : origin.vertical === 'center' ? height / 2 : height,
  };
}

function useFloatingPosition({
  anchorEl,
  anchorOrigin = { vertical: 'bottom', horizontal: 'left' },
  anchorPosition,
  anchorReference,
  open,
  transformOrigin = { vertical: 'top', horizontal: 'left' },
}: {
  anchorEl?: Element | null;
  anchorOrigin?: Origin;
  anchorPosition?: { left: number; top: number } | null;
  anchorReference?: 'anchorEl' | 'anchorPosition';
  open: boolean;
  transformOrigin?: Origin;
}) {
  const [style, setStyle] = useState<CSSProperties>({ opacity: 0 });

  useEffect(() => {
    if (!open) {
      return;
    }

    const updatePosition = () => {
      const width = 0;
      const height = 0;
      const transform = originOffset(transformOrigin, width, height);

      if (anchorReference === 'anchorPosition' && anchorPosition) {
        setStyle({
          left: anchorPosition.left - transform.x,
          opacity: 1,
          position: 'fixed',
          top: anchorPosition.top - transform.y,
        });
        return;
      }

      if (!anchorEl) {
        setStyle({ opacity: 0 });
        return;
      }

      const rect = anchorEl.getBoundingClientRect();
      const anchor = originOffset(anchorOrigin, rect.width, rect.height);
      setStyle({
        left: rect.left + anchor.x - transform.x,
        opacity: 1,
        position: 'fixed',
        top: rect.top + anchor.y - transform.y,
      });
    };

    updatePosition();
    window.addEventListener('scroll', updatePosition, true);
    window.addEventListener('resize', updatePosition);
    return () => {
      window.removeEventListener('scroll', updatePosition, true);
      window.removeEventListener('resize', updatePosition);
    };
  }, [anchorEl, anchorOrigin, anchorPosition, anchorReference, open, transformOrigin]);

  return style;
}

const DialogConfigContext = createContext<{
  fullWidth?: boolean;
  maxWidth?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | false;
  paperSx?: CommonProps['sx'];
}>({});

function dialogWidthClass(maxWidth: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | false | undefined) {
  switch (maxWidth) {
    case 'xs':
      return 'sm:max-w-md';
    case 'sm':
      return 'sm:max-w-lg';
    case 'md':
      return 'sm:max-w-2xl';
    case 'lg':
      return 'sm:max-w-4xl';
    case 'xl':
      return 'sm:max-w-6xl';
    case false:
      return 'sm:max-w-none';
    default:
      return 'sm:max-w-lg';
  }
}

function Dialog({
  children,
  open,
  onClose,
  fullWidth,
  maxWidth,
  PaperProps,
  fullScreen,
  slotProps,
  disableEscapeKeyDown,
  sx,
  TransitionComponent,
}: {
  children?: ReactNode;
  disableEscapeKeyDown?: boolean;
  fullWidth?: boolean;
  fullScreen?: boolean;
  maxWidth?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | false;
  onClose?: (_event?: unknown, _reason?: string) => void;
  open: boolean;
  PaperProps?: { sx?: any };
  slotProps?: {
    backdrop?: { sx?: SxProp; [key: string]: any };
    paper?: { sx?: SxProp; [key: string]: any };
  };
  sx?: SxProp;
  TransitionComponent?: unknown;
  transitionDuration?: number;
}) {
  const rootStyle = useSxStyle(sx);
  const paperStyle = useSxStyle(slotProps?.paper?.sx ?? PaperProps?.sx);
  const backdropStyle = useSxStyle(slotProps?.backdrop?.sx);
  const animationVariant = fullScreen || TransitionComponent ? 'slide-up' : 'default';
  const contentClassName = cn(
    fullScreen
      ? 'left-0 top-0 flex h-screen w-screen max-w-none translate-x-0 translate-y-0 flex-col gap-0 overflow-hidden rounded-none border-0 p-0 sm:rounded-none'
      : [
          fullWidth && 'w-[min(96vw,100%)]',
          dialogWidthClass(maxWidth),
        ],
  );

  return (
    <DialogConfigContext.Provider
      value={{
        fullWidth,
        maxWidth: fullScreen ? false : maxWidth,
      }}
    >
      <ShadDialog open={open} onOpenChange={(next) => { if (!next) onClose?.({}, 'backdropClick'); }}>
      <ShadDialogContent
        showCloseButton={false}
        aria-describedby={undefined}
        animationVariant={animationVariant}
        className={contentClassName}
        overlayStyle={backdropStyle}
        onEscapeKeyDown={disableEscapeKeyDown ? (event) => event.preventDefault() : undefined}
        style={{ ...paperStyle, ...rootStyle }}
      >
          <ShadDialogTitle className="sr-only">Dialog</ShadDialogTitle>
          {children}
      </ShadDialogContent>
      </ShadDialog>
    </DialogConfigContext.Provider>
  );
}

function DialogTitle({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLHeadingElement>) {
  const sxStyle = useSxStyle(sx);
  return (
    <ShadDialogHeader className="pb-2" style={{ ...sxStyle, ...style }}>
      <ShadDialogTitle className={className} {...props}>
        {children}
      </ShadDialogTitle>
    </ShadDialogHeader>
  );
}

function DialogContent({
  children,
  sx,
  style,
  className,
  dividers,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement> & { dividers?: boolean }) {
  const sxStyle = useSxStyle(sx);

  return (
    <div
      className={cn('grid gap-4', dividers && 'border-y py-4', className)}
      style={{ ...sxStyle, ...style }}
      {...props}
    >
      {children}
    </div>
  );
}

function DialogActions({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement>) {
  const sxStyle = useSxStyle(sx);
  return (
    <ShadDialogFooter className={className} style={{ ...sxStyle, ...style }} {...props}>
      {children}
    </ShadDialogFooter>
  );
}

function DialogContentText({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLParagraphElement>) {
  const sxStyle = useSxStyle(sx);
  return (
    <DialogDescription className={className} style={{ ...sxStyle, ...style }} {...props}>
      {children}
    </DialogDescription>
  );
}

function FloatingSurface({
  anchorEl,
  anchorOrigin,
  anchorPosition,
  anchorReference,
  children,
  className,
  container,
  onClose,
  open,
  transformOrigin,
}: {
  anchorEl?: Element | null;
  anchorOrigin?: Origin;
  anchorPosition?: { left: number; top: number } | null;
  anchorReference?: 'anchorEl' | 'anchorPosition';
  children: ReactNode;
  className?: string;
  container?: Element | null;
  onClose?: () => void;
  open: boolean;
  transformOrigin?: Origin;
}) {
  const positionStyle = useFloatingPosition({
    anchorEl,
    anchorOrigin,
    anchorPosition,
    anchorReference,
    open,
    transformOrigin,
  });

  useEffect(() => {
    if (!open) {
      return;
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose?.();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return createPortal(
    <div className="fixed inset-0 z-50" onMouseDown={() => onClose?.()}>
      <div
        className={cn('absolute min-w-[12rem] rounded-xl border bg-popover text-popover-foreground shadow-lg', className)}
        style={positionStyle}
        onMouseDown={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    container ?? document.body,
  );
}

function Menu({
  children,
  anchorEl,
  anchorOrigin,
  anchorPosition,
  anchorReference,
  open,
  onClose,
  slotProps,
  transformOrigin,
}: {
  anchorEl?: Element | null;
  anchorOrigin?: Origin;
  anchorPosition?: { left: number; top: number } | null;
  anchorReference?: 'anchorEl' | 'anchorPosition';
  children?: ReactNode;
  onClose?: () => void;
  open: boolean;
  slotProps?: {
    paper?: Record<string, any>;
  };
  transformOrigin?: Origin;
  transitionDuration?: number;
}) {
  const paperStyle = useSxStyle(slotProps?.paper?.sx);
  return (
    <FloatingSurface
      anchorEl={anchorEl}
      anchorOrigin={anchorOrigin}
      anchorPosition={anchorPosition}
      anchorReference={anchorReference}
      onClose={onClose}
      open={open}
      className="p-1"
      transformOrigin={transformOrigin}
    >
      <div style={paperStyle}>{children}</div>
    </FloatingSurface>
  );
}

function Popover({
  children,
  anchorEl,
  anchorOrigin,
  container,
  open,
  onClose,
  slotProps,
  transformOrigin,
}: {
  anchorEl?: Element | null;
  anchorOrigin?: Origin;
  children?: ReactNode;
  container?: Element | null;
  disablePortal?: boolean;
  onClose?: () => void;
  open: boolean;
  slotProps?: {
    paper?: { sx?: CommonProps['sx'] };
  };
  transitionDuration?: number;
  transformOrigin?: Origin;
}) {
  const paperStyle = useSxStyle(slotProps?.paper?.sx);
  return (
    <FloatingSurface
      anchorEl={anchorEl}
      anchorOrigin={anchorOrigin}
      container={container}
      onClose={onClose}
      open={open}
      transformOrigin={transformOrigin}
      className="p-0"
    >
      <div style={paperStyle}>{children}</div>
    </FloatingSurface>
  );
}

function Tooltip({
  children,
  title,
}: {
  arrow?: boolean;
  children: ReactNode;
  placement?: string;
  title?: ReactNode;
}) {
  const label = typeof title === 'string' ? title : undefined;
  return <span title={label}>{children}</span>;
}

function Snackbar({
  open,
  children,
  anchorOrigin = { vertical: 'bottom', horizontal: 'center' },
  autoHideDuration,
  onClose,
}: {
  anchorOrigin?: { horizontal: 'left' | 'center' | 'right'; vertical: 'bottom' | 'top' };
  autoHideDuration?: number;
  children?: ReactNode;
  onClose?: () => void;
  open: boolean;
}) {
  useEffect(() => {
    if (!open || !autoHideDuration) {
      return;
    }
    const timer = window.setTimeout(() => onClose?.(), autoHideDuration);
    return () => window.clearTimeout(timer);
  }, [autoHideDuration, onClose, open]);

  if (!open) {
    return null;
  }

  const horizontalClass = anchorOrigin.horizontal === 'left'
    ? 'left-4'
    : anchorOrigin.horizontal === 'right'
      ? 'right-4'
      : 'left-1/2 -translate-x-1/2';

  const verticalClass = anchorOrigin.vertical === 'top' ? 'top-4' : 'bottom-4';

  return createPortal(
    <div className={cn('fixed z-50', horizontalClass, verticalClass)} onClick={() => onClose?.()}>
      {children}
    </div>,
    document.body,
  );
}

function Drawer({
  open,
  onClose,
  anchor = 'right',
  children,
  sx,
}: {
  anchor?: 'left' | 'right';
  children?: ReactNode;
  container?: Element | null;
  onClose?: () => void;
  open: boolean;
  variant?: string;
  sx?: CommonProps['sx'];
}) {
  const paperStyle = useSxStyle(sx);
  if (!open) {
    return null;
  }

  return createPortal(
    <div className="fixed inset-0 z-50" onMouseDown={() => onClose?.()}>
      <div className="absolute inset-0 bg-black/40" />
      <div
        className={cn(
          'absolute top-0 h-full w-[min(100vw,360px)] border bg-background shadow-xl',
          anchor === 'left' ? 'left-0' : 'right-0',
        )}
        style={paperStyle}
        onMouseDown={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}

function Collapse({
  children,
  in: open,
  unmountOnExit,
  style,
  className,
}: {
  children?: ReactNode;
  className?: string;
  in: boolean;
  style?: CSSProperties;
  timeout?: 'auto' | number;
  unmountOnExit?: boolean;
}) {
  if (!open && unmountOnExit) {
    return null;
  }

  return (
    <div className={cn(!open && 'hidden', className)} style={style}>
      {children}
    </div>
  );
}

const Slide = forwardRef<unknown, { children: ReactNode; direction?: string; in?: boolean }>(
  function MaterialSlide({ children }, _ref) {
    return <>{children}</>;
  },
);

export {
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Drawer,
  Menu,
  Popover,
  Slide,
  Snackbar,
  Tooltip,
};
