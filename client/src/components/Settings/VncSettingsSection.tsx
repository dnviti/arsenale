import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { VncSettings } from '../../constants/vncDefaults';
import { CLIPBOARD_ENCODINGS, VNC_DEFAULTS } from '../../constants/vncDefaults';
import {
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsSectionBlock,
} from './settings-ui';
import { SettingsOverrideToggle, useOverrideableSettings } from './settings-overrides';

interface VncSettingsSectionProps {
  value: Partial<VncSettings>;
  onChange: (updated: Partial<VncSettings>) => void;
  mode: 'global' | 'connection';
  resolvedDefaults: VncSettings;
  enforcedFields?: Partial<VncSettings>;
}

const AUTO_VALUE = '__auto__';

export default function VncSettingsSection({
  value,
  onChange,
  mode,
  resolvedDefaults,
  enforcedFields,
}: VncSettingsSectionProps) {
  const defaults = { ...VNC_DEFAULTS, ...resolvedDefaults } as VncSettings;
  const {
    getValue,
    isOverridden,
    isEnforced,
    isDisabled,
    setField,
    toggleOverride,
  } = useOverrideableSettings<VncSettings>({
    value,
    onChange,
    defaults,
    mode,
    enforcedFields,
  });

  const overrideControl = (key: keyof VncSettings) =>
    mode === 'connection' ? (
      <SettingsOverrideToggle
        checked={isOverridden(key)}
        enforced={isEnforced(key)}
        onCheckedChange={() => toggleOverride(key)}
      />
    ) : undefined;

  return (
    <SettingsFieldGroup className="space-y-5">
      <SettingsSectionBlock
        title="Display & Clipboard"
        description="Pick the wire format for pixels, cursor rendering, and copied text."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Color Depth"
            description="Auto lets the server choose the optimal format."
            aside={overrideControl('colorDepth')}
          >
            <Select
              value={String(getValue('colorDepth') ?? AUTO_VALUE)}
              onValueChange={(nextValue) =>
                setField(
                  'colorDepth',
                  (nextValue === AUTO_VALUE ? undefined : Number(nextValue)) as VncSettings['colorDepth'],
                )}
              disabled={isDisabled('colorDepth')}
            >
              <SelectTrigger aria-label="Color Depth">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={AUTO_VALUE}>Auto</SelectItem>
                <SelectItem value="8">8-bit (256 colors)</SelectItem>
                <SelectItem value="16">16-bit (High Color)</SelectItem>
                <SelectItem value="24">24-bit (True Color)</SelectItem>
                <SelectItem value="32">32-bit (True Color + Alpha)</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Cursor Mode"
            description="Local rendering is usually smoother in the browser."
            aside={overrideControl('cursor')}
          >
            <Select
              value={getValue('cursor') ?? 'local'}
              onValueChange={(nextValue) => setField('cursor', nextValue as VncSettings['cursor'])}
              disabled={isDisabled('cursor')}
            >
              <SelectTrigger aria-label="Cursor Mode">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="local">Local (browser rendered)</SelectItem>
                <SelectItem value="remote">Remote (server rendered)</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Clipboard Encoding"
            description="Match the remote system’s expected text encoding."
            aside={overrideControl('clipboardEncoding')}
          >
            <Select
              value={getValue('clipboardEncoding') ?? 'UTF-8'}
              onValueChange={(nextValue) => setField('clipboardEncoding', nextValue as VncSettings['clipboardEncoding'])}
              disabled={isDisabled('clipboardEncoding')}
            >
              <SelectTrigger aria-label="Clipboard Encoding">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CLIPBOARD_ENCODINGS.map((encoding) => (
                  <SelectItem key={encoding.value} value={encoding.value}>
                    {encoding.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Behavior"
        description="Limit input, correct color channels, or mute audio."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Read-only Mode"
            description="Prevent keyboard and mouse input from reaching the remote desktop."
            aside={overrideControl('readOnly')}
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">View only, without sending input</span>
              <Switch
                checked={Boolean(getValue('readOnly'))}
                disabled={isDisabled('readOnly')}
                onCheckedChange={(nextValue) => setField('readOnly', nextValue)}
                aria-label="Read-only mode"
              />
            </label>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Swap Red / Blue Channels"
            description="Correct color order when the remote server reports BGR pixels."
            aside={overrideControl('swapRedBlue')}
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Swap the red and blue channels</span>
              <Switch
                checked={Boolean(getValue('swapRedBlue'))}
                disabled={isDisabled('swapRedBlue')}
                onCheckedChange={(nextValue) => setField('swapRedBlue', nextValue)}
                aria-label="Swap red and blue channels"
              />
            </label>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Disable Audio"
            description="Mute any audio stream that the VNC session exposes."
            aside={overrideControl('disableAudio')}
            className="xl:col-span-2"
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Disable session audio</span>
              <Switch
                checked={Boolean(getValue('disableAudio') ?? true)}
                disabled={isDisabled('disableAudio')}
                onCheckedChange={(nextValue) => setField('disableAudio', nextValue)}
                aria-label="Disable audio"
              />
            </label>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>
    </SettingsFieldGroup>
  );
}
