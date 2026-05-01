import * as React from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { cn } from '@/lib/utils';

interface CounterBadgeProps {
  className?: string;
  count: number;
  max?: number;
}

export function CounterBadge({ className, count, max = 99 }: CounterBadgeProps) {
  if (count <= 0) {
    return null;
  }

  return (
    <span
      className={cn(
        'absolute -right-1 -top-1 inline-flex min-h-5 min-w-5 items-center justify-center rounded-full bg-destructive px-1 text-[11px] font-semibold text-destructive-foreground shadow-sm',
        className,
      )}
    >
      {count > max ? `${max}+` : count}
    </span>
  );
}

export const HeaderIconButton = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(function HeaderIconButton({ className, type = 'button', title, ...props }, ref) {
  const ariaLabel = props['aria-label'] ?? (typeof title === 'string' ? title : undefined);

  return (
    <button
      ref={ref}
      type={type}
      title={title}
      aria-label={ariaLabel}
      className={cn(
        'relative inline-flex size-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
        className,
      )}
      {...props}
    />
  );
});

interface NotificationToastProps {
  message: string;
  severity: 'error' | 'warning' | 'info' | 'success';
  onClose?: () => void;
}

const ALERT_VARIANTS: Record<NotificationToastProps['severity'], 'destructive' | 'info' | 'success' | 'warning'> = {
  error: 'destructive',
  warning: 'warning',
  info: 'info',
  success: 'success',
};

export function StatusPill({
  children,
  tone = 'neutral',
  className,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  children: React.ReactNode;
  tone?: 'danger' | 'neutral' | 'primary';
}) {
  const toneClasses = {
    neutral: 'border-border bg-muted/50 text-foreground',
    primary: 'border-primary/30 bg-primary/10 text-primary',
    danger: 'border-destructive/30 bg-destructive/10 text-destructive',
  } as const;

  return (
    <button
      type="button"
      className={cn(
        'inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors hover:bg-accent',
        toneClasses[tone],
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export function NotificationToast({ message, severity, onClose }: NotificationToastProps) {
  return (
    <div className="pointer-events-none fixed bottom-4 left-1/2 z-50 w-[min(36rem,calc(100vw-2rem))] -translate-x-1/2">
      <Alert variant={ALERT_VARIANTS[severity]} className="pointer-events-auto shadow-lg">
        <div className="flex items-start justify-between gap-3">
          <AlertDescription className="text-foreground">{message}</AlertDescription>
          {onClose ? (
            <button
              type="button"
              aria-label="Dismiss notification"
              className="inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-background/60 hover:text-foreground"
              onClick={onClose}
            >
              ×
            </button>
          ) : null}
        </div>
      </Alert>
    </div>
  );
}
