import {
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
} from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import {
  Card as ShadCard,
  CardContent as ShadCardContent,
  CardFooter as ShadCardFooter,
} from '@/components/ui/card';
import { cn } from '@/lib/utils';
import { useSxClassName, type SxProp } from './theme';

interface CommonProps {
  [key: string]: any;
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
  sx?: SxProp;
}

function Card({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement>) {
  const sxClassName = useSxClassName(sx);
  return (
    <ShadCard className={cn(sxClassName, className)} style={style} {...props}>
      {children}
    </ShadCard>
  );
}

function CardContent({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement>) {
  const sxClassName = useSxClassName(sx);
  return (
    <ShadCardContent className={cn(sxClassName, className)} style={style} {...props}>
      {children}
    </ShadCardContent>
  );
}

function CardActions({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement>) {
  const sxClassName = useSxClassName(sx);
  return (
    <ShadCardFooter className={cn('gap-2', sxClassName, className)} style={style} {...props}>
      {children}
    </ShadCardFooter>
  );
}

function List({
  children,
  className,
  dense,
  disablePadding,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLUListElement> & {
  dense?: boolean;
  disablePadding?: boolean;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <ul
      className={cn('space-y-1', !disablePadding && 'py-1', dense && 'space-y-0.5', sxClassName, className)}
      style={style}
      {...props}
    >
      {children}
    </ul>
  );
}

function ListItem({
  children,
  className,
  dense,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLLIElement> & {
  dense?: boolean;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <li className={cn('relative', dense && 'py-0.5', sxClassName, className)} style={style} {...props}>
      {children}
    </li>
  );
}

function ListItemButton({
  children,
  className,
  selected,
  sx,
  style,
  ...props
}: CommonProps &
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    selected?: boolean;
  }) {
  const sxClassName = useSxClassName(sx);
  return (
    <button
      type="button"
      role="button"
      className={cn(
        'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm hover:bg-accent',
        selected && 'bg-accent text-accent-foreground',
        sxClassName,
        className,
      )}
      style={style}
      {...props}
    >
      {children}
    </button>
  );
}

function ListItemIcon({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLSpanElement>) {
  const sxClassName = useSxClassName(sx);
  return (
    <span className={cn('inline-flex min-w-8 items-center justify-center text-muted-foreground', sxClassName, className)} style={style} {...props}>
      {children}
    </span>
  );
}

function ListItemText({
  primary,
  secondary,
  className,
  children,
  primaryTypographyProps,
  secondaryTypographyProps,
  sx,
  style,
}: {
  children?: ReactNode;
  className?: string;
  primary?: ReactNode;
  primaryTypographyProps?: Record<string, any>;
  secondary?: ReactNode;
  secondaryTypographyProps?: Record<string, any>;
  sx?: SxProp;
  style?: CSSProperties;
}) {
  const sxClassName = useSxClassName(sx);
  const primaryContent = primary ?? children;
  return (
    <span className={cn('min-w-0 flex-1', sxClassName, className)} style={style}>
      {primaryContent ? <span className={cn('block truncate text-sm', primaryTypographyProps?.className)}>{primaryContent}</span> : null}
      {secondary ? (
        <span className={cn('block truncate text-xs text-muted-foreground', secondaryTypographyProps?.className)}>
          {secondary}
        </span>
      ) : null}
    </span>
  );
}

function ListItemSecondaryAction({
  children,
  className,
}: {
  children?: ReactNode;
  className?: string;
}) {
  return <span className={cn('ml-auto inline-flex items-center gap-2', className)}>{children}</span>;
}

function Table({
  children,
  className,
  sx,
  style,
  size,
  ...props
}: CommonProps &
  HTMLAttributes<HTMLTableElement> & {
    size?: 'small' | 'medium';
  }) {
  const sxClassName = useSxClassName(sx);
  return (
    <table className={cn('w-full border-collapse text-sm', size === 'small' && 'text-xs', sxClassName, className)} style={style} {...props}>
      {children}
    </table>
  );
}

function TableContainer({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement>) {
  const sxClassName = useSxClassName(sx);
  return (
    <div className={cn('overflow-x-auto rounded-xl border', sxClassName, className)} style={style} {...props}>
      {children}
    </div>
  );
}

function TableHead(props: HTMLAttributes<HTMLTableSectionElement>) {
  return <thead {...props} />;
}

function TableBody(props: HTMLAttributes<HTMLTableSectionElement>) {
  return <tbody {...props} />;
}

function TableRow({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLTableRowElement>) {
  const sxClassName = useSxClassName(sx);
  return (
    <tr className={cn('border-b last:border-b-0', sxClassName, className)} style={style} {...props}>
      {children}
    </tr>
  );
}

function TableCell({
  children,
  className,
  sx,
  style,
  align,
  ...props
}: CommonProps &
  HTMLAttributes<HTMLTableCellElement> & {
    align?: 'left' | 'center' | 'right';
  }) {
  const sxClassName = useSxClassName(sx);
  return (
    <td
      className={cn(
        'px-3 py-2 align-middle',
        align === 'center' && 'text-center',
        align === 'right' && 'text-right',
        sxClassName,
        className,
      )}
      style={style}
      {...props}
    >
      {children}
    </td>
  );
}

function TablePagination({
  count,
  onPageChange,
  onRowsPerPageChange,
  page,
  rowsPerPage,
  rowsPerPageOptions = [10, 25, 50],
}: {
  count: number;
  component?: any;
  onPageChange: (_event: unknown, page: number) => void;
  onRowsPerPageChange?: (event: any) => void;
  page: number;
  rowsPerPage: number;
  rowsPerPageOptions?: number[];
}) {
  const from = count === 0 ? 0 : page * rowsPerPage + 1;
  const to = Math.min(count, (page + 1) * rowsPerPage);

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t px-3 py-3 text-sm text-muted-foreground">
      <div>
        {from}-{to} of {count}
      </div>
      <div className="flex items-center gap-2">
        {onRowsPerPageChange ? (
          <select className="rounded-md border bg-background px-2 py-1 text-sm" value={rowsPerPage} onChange={onRowsPerPageChange}>
            {rowsPerPageOptions.map((option) => (
              <option key={option} value={option}>
                {option} / page
              </option>
            ))}
          </select>
        ) : null}
        <button type="button" className="rounded-md border px-2 py-1 disabled:opacity-50" disabled={page === 0} onClick={() => onPageChange({}, page - 1)}>
          Prev
        </button>
        <button
          type="button"
          className="rounded-md border px-2 py-1 disabled:opacity-50"
          disabled={(page + 1) * rowsPerPage >= count}
          onClick={() => onPageChange({}, page + 1)}
        >
          Next
        </button>
      </div>
    </div>
  );
}

function TableSortLabel({
  active,
  direction = 'asc',
  children,
  onClick,
}: {
  active?: boolean;
  children?: ReactNode;
  direction?: 'asc' | 'desc';
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
}) {
  return (
    <button type="button" className={cn('inline-flex items-center gap-1 font-medium', active && 'text-foreground')} onClick={onClick}>
      <span>{children}</span>
      {active ? direction === 'asc' ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" /> : null}
    </button>
  );
}

export {
  Card,
  CardActions,
  CardContent,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemSecondaryAction,
  ListItemText,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  TableSortLabel,
};
