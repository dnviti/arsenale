import { useEffect, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, ChevronRight, Search, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

interface SidebarIconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean;
}

export function SidebarIconButton({
  active,
  className,
  type = 'button',
  title,
  ...props
}: SidebarIconButtonProps) {
  const ariaLabel = props['aria-label'] ?? (typeof title === 'string' ? title : undefined);

  return (
    <button
      type={type}
      title={title}
      aria-label={ariaLabel}
      className={cn(
        'inline-flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        active && 'bg-accent text-foreground',
        className,
      )}
      {...props}
    />
  );
}

interface SidebarSectionHeaderProps {
  open: boolean;
  label: string;
  icon: ReactNode;
  onToggle: () => void;
  actions?: ReactNode;
}

export function SidebarSectionHeader({
  open,
  label,
  icon,
  onToggle,
  actions,
}: SidebarSectionHeaderProps) {
  return (
    <div className="flex items-center gap-1 px-2">
      <button
        type="button"
        onClick={onToggle}
        className="flex min-w-0 flex-1 items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-accent/70"
      >
        {open ? (
          <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
        )}
        <span className="shrink-0 text-muted-foreground">{icon}</span>
        <span className="truncate text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          {label}
        </span>
      </button>
      {actions}
    </div>
  );
}

interface SidebarSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function SidebarSearchInput({
  value,
  onChange,
  placeholder = 'Search connections...',
}: SidebarSearchInputProps) {
  return (
    <div className="relative px-2">
      <Search className="pointer-events-none absolute left-5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            onChange('');
          }
        }}
        placeholder={placeholder}
        className="h-9 pl-9 pr-9 text-sm"
      />
      {value ? (
        <button
          type="button"
          aria-label="Clear search"
          onClick={() => onChange('')}
          className="absolute right-4 top-1/2 inline-flex size-6 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <X className="size-3.5" />
        </button>
      ) : null}
    </div>
  );
}

interface SidebarContextMenuAction {
  label: string;
  icon?: ReactNode;
  onSelect: () => void;
  disabled?: boolean;
  destructive?: boolean;
  separatorBefore?: boolean;
}

interface SidebarContextMenuProps {
  open: boolean;
  position: { x: number; y: number } | null;
  onClose: () => void;
  label?: string;
  actions: SidebarContextMenuAction[];
}

export function SidebarContextMenu({
  open,
  position,
  onClose,
  label,
  actions,
}: SidebarContextMenuProps) {
  useEffect(() => {
    if (!open) {
      return undefined;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    const handleWindowBlur = () => onClose();

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('blur', handleWindowBlur);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('blur', handleWindowBlur);
    };
  }, [onClose, open]);

  if (!open || !position || typeof document === 'undefined') {
    return null;
  }

  const maxWidth = 240;
  const maxHeight = actions.length * 36 + (label ? 32 : 12);
  const left = Math.min(position.x, window.innerWidth - maxWidth - 12);
  const top = Math.min(position.y, window.innerHeight - maxHeight - 12);

  return createPortal(
    <div
      className="fixed inset-0 z-[80]"
      onMouseDown={onClose}
      onContextMenu={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <div
        className="absolute min-w-[14rem] overflow-hidden rounded-xl border bg-popover p-1 text-popover-foreground shadow-lg animate-in fade-in-0 zoom-in-95"
        style={{ left, top }}
        onMouseDown={(event) => event.stopPropagation()}
        onContextMenu={(event) => event.preventDefault()}
      >
        {label ? (
          <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
            {label}
          </div>
        ) : null}
        {actions.map((action, index) => (
          <div key={`${action.label}-${index}`}>
            {action.separatorBefore ? <div className="my-1 h-px bg-border" /> : null}
            <button
              type="button"
              disabled={action.disabled}
              onClick={() => {
                action.onSelect();
                onClose();
              }}
              className={cn(
                'flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50',
                action.destructive && 'text-destructive hover:bg-destructive/10 hover:text-destructive',
              )}
            >
              {action.icon ? (
                <span className="inline-flex size-4 shrink-0 items-center justify-center">
                  {action.icon}
                </span>
              ) : null}
              <span className="truncate">{action.label}</span>
            </button>
          </div>
        ))}
      </div>
    </div>,
    document.body,
  );
}

interface SidebarConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  destructive?: boolean;
}

export function SidebarConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  onConfirm,
  destructive = false,
}: SidebarConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription asChild>
            <div>{description}</div>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            variant={destructive ? 'destructive' : 'default'}
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
