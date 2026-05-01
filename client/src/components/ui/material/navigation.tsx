import {
  Children,
  cloneElement,
  createContext,
  isValidElement,
  useContext,
  useState,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
} from 'react';
import { ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useSxClassName, type SxProp } from './theme';

interface CommonProps {
  [key: string]: any;
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
  sx?: SxProp;
}

const TabsContext = createContext<{
  onChange?: (_event: unknown, value: any) => void;
  value?: any;
}>({});

function Tabs({
  children,
  value,
  onChange,
  role,
  sx,
  style,
  className,
}: CommonProps & {
  onChange?: (_event: unknown, value: any) => void;
  role?: string;
  value?: any;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <TabsContext.Provider value={{ onChange, value }}>
      <div
        role={role ?? 'tablist'}
        className={cn('inline-flex items-center gap-1 rounded-lg bg-muted p-1', sxClassName, className)}
        style={style}
      >
        {children}
      </div>
    </TabsContext.Provider>
  );
}

function Tab({
  label,
  value,
  icon,
  iconPosition,
  sx,
  style,
  className,
  ...props
}: {
  className?: string;
  icon?: ReactNode;
  iconPosition?: 'bottom' | 'end' | 'start' | 'top';
  label?: ReactNode;
  style?: CSSProperties;
  sx?: SxProp;
  value?: any;
} & CommonProps) {
  const tabs = useContext(TabsContext);
  const selected = tabs.value === value;
  const sxClassName = useSxClassName(sx);
  const content = (
    <>
      {icon}
      {label}
    </>
  );

  return (
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      className={cn(
        'inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
        selected ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
        sxClassName,
        className,
      )}
      style={style}
      onClick={() => tabs.onChange?.({}, value)}
      {...props}
    >
      {iconPosition === 'bottom' || iconPosition === 'end'
        ? (
          <>
            {label}
            {icon}
          </>
        )
        : content}
    </button>
  );
}

const AccordionContext = createContext<{
  expanded: boolean;
  toggle: () => void;
}>({
  expanded: false,
  toggle: () => {},
});

function Accordion({
  children,
  expanded,
  defaultExpanded,
  onChange,
  sx,
  style,
  className,
}: CommonProps & {
  defaultExpanded?: boolean;
  expanded?: boolean;
  onChange?: (_event: unknown, expanded: boolean) => void;
}) {
  const [localExpanded, setLocalExpanded] = useState(Boolean(defaultExpanded));
  const resolvedExpanded = expanded ?? localExpanded;
  const sxClassName = useSxClassName(sx);

  const toggle = () => {
    const next = !resolvedExpanded;
    if (expanded == null) {
      setLocalExpanded(next);
    }
    onChange?.({}, next);
  };

  return (
    <AccordionContext.Provider value={{ expanded: resolvedExpanded, toggle }}>
      <div className={cn('rounded-xl border bg-card text-card-foreground', sxClassName, className)} style={style}>
        {children}
      </div>
    </AccordionContext.Provider>
  );
}

function AccordionSummary({
  children,
  expandIcon,
  className,
  ...props
}: CommonProps & HTMLAttributes<HTMLButtonElement> & {
  expandIcon?: ReactNode;
}) {
  const accordion = useContext(AccordionContext);
  return (
    <button
      type="button"
      className={cn('flex w-full items-center justify-between gap-4 px-4 py-3 text-left text-sm font-medium', className)}
      onClick={accordion.toggle}
      {...props}
    >
      <span>{children}</span>
      {expandIcon ?? <ChevronDown className={cn('size-4 transition-transform', accordion.expanded && 'rotate-180')} />}
    </button>
  );
}

function AccordionDetails({
  children,
  className,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement>) {
  const accordion = useContext(AccordionContext);
  const sxClassName = useSxClassName(sx);
  if (!accordion.expanded) {
    return null;
  }
  return (
    <div className={cn('border-t px-4 py-4', sxClassName, className)} style={style} {...props}>
      {children}
    </div>
  );
}

function AppBar({
  children,
  className,
  position,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement> & {
  position?: string;
}) {
  const sxClassName = useSxClassName(sx);
  void position;
  return (
    <header className={cn('border-b bg-background/90 backdrop-blur-xl', sxClassName, className)} style={style} {...props}>
      {children}
    </header>
  );
}

function Toolbar({
  children,
  className,
  variant,
  sx,
  style,
  ...props
}: CommonProps & HTMLAttributes<HTMLDivElement> & {
  variant?: string;
}) {
  const sxClassName = useSxClassName(sx);
  void variant;
  return (
    <div className={cn('flex min-h-14 items-center gap-3 px-4', sxClassName, className)} style={style} {...props}>
      {children}
    </div>
  );
}

const StepperContext = createContext<{
  activeStep: number;
}>({
  activeStep: 0,
});

function Stepper({
  children,
  activeStep = 0,
  alternativeLabel,
  className,
  sx,
  style,
}: CommonProps & {
  activeStep?: number;
  alternativeLabel?: boolean;
}) {
  const sxClassName = useSxClassName(sx);
  const items = Children.toArray(children);
  return (
    <StepperContext.Provider value={{ activeStep }}>
      <div
        className={cn('grid gap-3', sxClassName, className)}
        style={{
          gridTemplateColumns: alternativeLabel ? `repeat(${Math.max(items.length, 1)}, minmax(0, 1fr))` : undefined,
          ...style,
        }}
      >
        {items.map((child, index) => (
          isValidElement(child)
            ? cloneElement(child, { stepIndex: index } as { stepIndex: number })
            : child
        ))}
      </div>
    </StepperContext.Provider>
  );
}

function Step({
  children,
  className,
  stepIndex,
}: {
  children?: ReactNode;
  className?: string;
  stepIndex?: number;
}) {
  const content = Children.map(children, (child) => (
    isValidElement(child)
      ? cloneElement(child, { index: stepIndex } as { index?: number })
      : child
  ));
  return <div className={cn('flex items-center gap-3', className)}>{content}</div>;
}

function StepLabel({
  children,
  className,
  index = 0,
}: {
  children?: ReactNode;
  className?: string;
  index?: number;
}) {
  const stepper = useContext(StepperContext);
  const active = stepper.activeStep >= index;
  return (
    <div className={cn('flex items-center gap-3 text-sm', className)}>
      <span className={cn('inline-flex size-6 items-center justify-center rounded-full border text-xs font-semibold', active ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground')}>
        {index + 1}
      </span>
      <span className={cn(active ? 'text-foreground' : 'text-muted-foreground')}>{children}</span>
    </div>
  );
}

export {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  AppBar,
  Step,
  StepLabel,
  Stepper,
  Tab,
  Tabs,
  Toolbar,
};
