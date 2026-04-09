import type { ReactNode } from 'react';
import { Edit3, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';
import {
  SettingsButtonRow,
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsSectionBlock,
  SettingsStatusBadge,
} from './settings-ui';

export interface PolicyTemplateOption {
  category: string;
  name: string;
  description: string;
  summary?: string;
  badge?: string;
  badgeTone?: 'neutral' | 'success' | 'warning' | 'destructive';
}

export function PolicyTemplatePicker({
  title,
  description,
  templates,
  onApply,
}: {
  title: string;
  description: string;
  templates: PolicyTemplateOption[];
  onApply: (templateName: string) => void;
}) {
  const groupedTemplates = templates.reduce<Record<string, PolicyTemplateOption[]>>((groups, template) => {
    if (!groups[template.category]) {
      groups[template.category] = [];
    }
    groups[template.category].push(template);
    return groups;
  }, {});

  return (
    <SettingsSectionBlock title={title} description={description}>
      <div className="space-y-4">
        {Object.entries(groupedTemplates).map(([category, categoryTemplates]) => (
          <div key={category} className="space-y-3">
            <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              {category}
            </div>
            <div className="grid gap-3 xl:grid-cols-2">
              {categoryTemplates.map((template) => (
                <button
                  key={template.name}
                  type="button"
                  onClick={() => onApply(template.name)}
                  className="rounded-xl border border-border/70 bg-background/70 p-4 text-left transition-colors hover:border-primary/30 hover:bg-accent/30"
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="space-y-1">
                      <div className="text-sm font-semibold text-foreground">{template.name}</div>
                      <p className="text-sm leading-6 text-muted-foreground">{template.description}</p>
                    </div>
                    {template.badge && (
                      <SettingsStatusBadge tone={template.badgeTone ?? 'neutral'}>
                        {template.badge}
                      </SettingsStatusBadge>
                    )}
                  </div>
                  {template.summary && (
                    <div className="mt-3 text-xs text-muted-foreground">
                      {template.summary}
                    </div>
                  )}
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </SettingsSectionBlock>
  );
}

export function PolicyRecordCard({
  title,
  description,
  badges,
  metadata,
  code,
  onEdit,
  onDelete,
}: {
  title: string;
  description?: string | null;
  badges?: ReactNode;
  metadata?: ReactNode;
  code?: string | null;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="rounded-2xl border border-border/70 bg-background/70 p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-3">
          <div className="space-y-2">
            <div className="text-sm font-semibold text-foreground">{title}</div>
            {description && (
              <p className="text-sm leading-6 text-muted-foreground">{description}</p>
            )}
          </div>
          {badges && (
            <div className="flex flex-wrap gap-2">
              {badges}
            </div>
          )}
        </div>

        <SettingsButtonRow className="shrink-0">
          <Button type="button" size="sm" variant="outline" onClick={onEdit}>
            <Edit3 />
            Edit
          </Button>
          <Button type="button" size="sm" variant="outline" onClick={onDelete}>
            <Trash2 />
            Delete
          </Button>
        </SettingsButtonRow>
      </div>

      {(metadata || code) && (
        <div className="mt-4 space-y-3">
          {metadata && (
            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
              {metadata}
            </div>
          )}
          {code && (
            <pre className="overflow-x-auto rounded-xl border border-border/70 bg-muted/40 px-3 py-2 text-xs text-foreground">
              <code>{code}</code>
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

export function PolicyEmptyState({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="rounded-2xl border border-dashed border-border/70 bg-muted/20 px-4 py-8 text-center">
      <div className="text-sm font-semibold text-foreground">{title}</div>
      <p className="mx-auto mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
        {description}
      </p>
    </div>
  );
}

export function PolicyDialogShell({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  children: ReactNode;
  footer: ReactNode;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">{children}</div>
        <DialogFooter>{footer}</DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function PolicyRoleChecklist({
  label,
  description,
  options,
  selected,
  onChange,
}: {
  label: string;
  description: string;
  options: string[];
  selected: string[];
  onChange: (selected: string[]) => void;
}) {
  return (
    <SettingsFieldCard label={label} description={description}>
      <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
        {options.map((role) => {
          const checked = selected.includes(role);
          const inputId = `${label}-${role}`.replace(/\s+/g, '-').toLowerCase();

          return (
            <label
              key={role}
              htmlFor={inputId}
              className={cn(
                'flex cursor-pointer items-start gap-3 rounded-lg border border-border/70 px-3 py-3 text-sm transition-colors',
                checked ? 'bg-accent/50' : 'bg-background/70 hover:bg-accent/30',
              )}
            >
              <Checkbox
                id={inputId}
                checked={checked}
                onCheckedChange={(nextChecked) => {
                  onChange(
                    nextChecked
                      ? [...selected, role]
                      : selected.filter((entry) => entry !== role),
                  );
                }}
              />
              <span className="font-medium text-foreground">{role}</span>
            </label>
          );
        })}
      </div>
    </SettingsFieldCard>
  );
}

export function PolicyMetadataBadge({
  children,
  variant = 'outline',
}: {
  children: ReactNode;
  variant?: 'default' | 'secondary' | 'outline' | 'destructive';
}) {
  return <Badge variant={variant}>{children}</Badge>;
}

export function PolicyFormSection({
  children,
}: {
  children: ReactNode;
}) {
  return <SettingsFieldGroup>{children}</SettingsFieldGroup>;
}
