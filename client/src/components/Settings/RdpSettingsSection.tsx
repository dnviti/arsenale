import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { Input } from '@/components/ui/input';
import type { RdpSettings } from '../../constants/rdpDefaults';
import {
  COMMON_TIMEZONES,
  KEYBOARD_LAYOUTS,
  QUALITY_PRESETS,
  RDP_DEFAULTS,
} from '../../constants/rdpDefaults';
import {
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsSectionBlock,
} from './settings-ui';
import { SettingsOverrideToggle, useOverrideableSettings } from './settings-overrides';

interface RdpSettingsSectionProps {
  value: Partial<RdpSettings>;
  onChange: (updated: Partial<RdpSettings>) => void;
  mode: 'global' | 'connection';
  resolvedDefaults?: RdpSettings;
  enforcedFields?: Partial<RdpSettings>;
}

const AUTO_VALUE = '__auto__';
const NOT_SET_VALUE = '__not_set__';

const customEffectFields: Array<[keyof RdpSettings, string]> = [
  ['enableWallpaper', 'Desktop wallpaper'],
  ['enableTheming', 'Windows theming'],
  ['enableFontSmoothing', 'Font smoothing (ClearType)'],
  ['enableFullWindowDrag', 'Full window drag'],
  ['enableDesktopComposition', 'Desktop composition (Aero)'],
  ['enableMenuAnimations', 'Menu animations'],
  ['forceLossless', 'Force lossless compression'],
];

export default function RdpSettingsSection({
  value,
  onChange,
  mode,
  resolvedDefaults,
  enforcedFields,
}: RdpSettingsSectionProps) {
  const defaults = resolvedDefaults ?? RDP_DEFAULTS;
  const {
    getValue,
    isOverridden,
    isEnforced,
    isDisabled,
    setField,
    toggleOverride,
  } = useOverrideableSettings<RdpSettings>({
    value,
    onChange,
    defaults,
    mode,
    enforcedFields,
  });

  const qualityPreset = getValue('qualityPreset') ?? 'balanced';
  const isCustomPreset = qualityPreset === 'custom';
  const overrideControl = (key: keyof RdpSettings) =>
    mode === 'connection' ? (
      <SettingsOverrideToggle
        checked={isOverridden(key)}
        enforced={isEnforced(key)}
        onCheckedChange={() => toggleOverride(key)}
      />
    ) : undefined;

  const handlePresetChange = (nextValue: string) => {
    const nextPreset = nextValue as NonNullable<RdpSettings['qualityPreset']>;
    const nextState: Partial<RdpSettings> = { ...value, qualityPreset: nextPreset };
    if (nextPreset !== 'custom' && QUALITY_PRESETS[nextPreset]) {
      Object.assign(nextState, QUALITY_PRESETS[nextPreset]);
    }
    onChange(nextState);
  };

  return (
    <SettingsFieldGroup className="space-y-5">
      <SettingsSectionBlock
        title="Quality"
        description="Pick a visual profile or fine-tune each effect."
      >
        <SettingsFieldCard
          label="Quality Preset"
          description={
            qualityPreset === 'performance'
              ? 'Minimal bandwidth with visual effects disabled.'
              : qualityPreset === 'balanced'
                ? 'Balanced visuals with theming and font smoothing.'
                : qualityPreset === 'quality'
                  ? 'Highest fidelity, with the most visual effects enabled.'
                  : 'Custom mode lets you tune each effect individually.'
          }
          aside={overrideControl('qualityPreset')}
        >
          <ToggleGroup
            type="single"
            value={qualityPreset}
            onValueChange={(nextValue) => {
              if (nextValue) {
                handlePresetChange(nextValue);
              }
            }}
            disabled={isDisabled('qualityPreset')}
            className="flex-wrap"
          >
            <ToggleGroupItem value="performance" variant="outline">Performance</ToggleGroupItem>
            <ToggleGroupItem value="balanced" variant="outline">Balanced</ToggleGroupItem>
            <ToggleGroupItem value="quality" variant="outline">Quality</ToggleGroupItem>
            <ToggleGroupItem value="custom" variant="outline">Custom</ToggleGroupItem>
          </ToggleGroup>
        </SettingsFieldCard>

        {isCustomPreset && (
          <div className="grid gap-3 xl:grid-cols-2">
            {customEffectFields.map(([key, label]) => (
              <label
                key={key}
                className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3"
              >
                <span className="text-sm text-muted-foreground">{label}</span>
                <Switch
                  checked={Boolean(getValue(key))}
                  disabled={isDisabled(key)}
                  onCheckedChange={(nextValue) => setField(key, nextValue as RdpSettings[typeof key])}
                  aria-label={label}
                />
              </label>
            ))}
          </div>
        )}
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Display & Resolution"
        description="Control the remote desktop’s size, DPI, and resize behavior."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Color Depth"
            description="Auto uses the negotiated session value."
            aside={overrideControl('colorDepth')}
          >
            <Select
              value={String(getValue('colorDepth') ?? AUTO_VALUE)}
              onValueChange={(nextValue) =>
                setField(
                  'colorDepth',
                  (nextValue === AUTO_VALUE ? undefined : Number(nextValue)) as RdpSettings['colorDepth'],
                )}
              disabled={isDisabled('colorDepth')}
            >
              <SelectTrigger aria-label="Color Depth">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={AUTO_VALUE}>Auto (negotiated)</SelectItem>
                <SelectItem value="8">8-bit (256 colors)</SelectItem>
                <SelectItem value="16">16-bit (High Color)</SelectItem>
                <SelectItem value="24">24-bit (True Color)</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="DPI"
            description={`Current DPI: ${getValue('dpi') ?? 96}.`}
            aside={overrideControl('dpi')}
          >
            <Slider
              value={[getValue('dpi') ?? 96]}
              min={48}
              max={384}
              step={12}
              disabled={isDisabled('dpi')}
              onValueChange={([nextValue]) => setField('dpi', nextValue)}
              aria-label="DPI"
            />
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Resolution"
            description="Leave width or height empty to let the server negotiate it."
            aside={overrideControl('width')}
            contentClassName="grid gap-3 sm:grid-cols-2"
          >
            <Input
              type="number"
              value={getValue('width') ?? ''}
              min={640}
              max={7680}
              placeholder="Width"
              disabled={isDisabled('width')}
              onChange={(event) => setField('width', event.target.value ? Number(event.target.value) : undefined)}
            />
            <Input
              type="number"
              value={getValue('height') ?? ''}
              min={480}
              max={4320}
              placeholder="Height"
              disabled={isDisabled('width')}
              onChange={(event) => setField('height', event.target.value ? Number(event.target.value) : undefined)}
            />
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Resize Method"
            description="Display update is smoother; reconnect is more compatible."
            aside={overrideControl('resizeMethod')}
          >
            <Select
              value={getValue('resizeMethod') ?? 'display-update'}
              onValueChange={(nextValue) => setField('resizeMethod', nextValue as RdpSettings['resizeMethod'])}
              disabled={isDisabled('resizeMethod')}
            >
              <SelectTrigger aria-label="Resize Method">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="display-update">Display Update</SelectItem>
                <SelectItem value="reconnect">Reconnect</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Audio"
        description="Choose whether the session can play or capture audio."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Remote Audio Playback"
            description="Play remote sound through the browser."
            aside={overrideControl('disableAudio')}
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Enable remote audio playback</span>
              <Switch
                checked={!Boolean(getValue('disableAudio'))}
                disabled={isDisabled('disableAudio')}
                onCheckedChange={(nextValue) => setField('disableAudio', !nextValue)}
                aria-label="Enable remote audio playback"
              />
            </label>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Microphone Input"
            description="Allow the remote session to receive microphone input."
            aside={overrideControl('enableAudioInput')}
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Enable microphone input</span>
              <Switch
                checked={Boolean(getValue('enableAudioInput'))}
                disabled={isDisabled('enableAudioInput')}
                onCheckedChange={(nextValue) => setField('enableAudioInput', nextValue)}
                aria-label="Enable microphone input"
              />
            </label>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Security"
        description="Control protocol security and certificate validation."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Security Type"
            description="Choose the negotiated protocol for authentication and transport."
            aside={overrideControl('security')}
          >
            <Select
              value={getValue('security') ?? 'any'}
              onValueChange={(nextValue) => setField('security', nextValue as RdpSettings['security'])}
              disabled={isDisabled('security')}
            >
              <SelectTrigger aria-label="Security Type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="any">Any (auto-negotiate)</SelectItem>
                <SelectItem value="nla">NLA</SelectItem>
                <SelectItem value="nla-ext">NLA Extended</SelectItem>
                <SelectItem value="tls">TLS</SelectItem>
                <SelectItem value="rdp">RDP (legacy)</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Certificate Handling"
            description="Allow sessions to continue when the server certificate is invalid."
            aside={overrideControl('ignoreCert')}
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Ignore server certificate errors</span>
              <Switch
                checked={Boolean(getValue('ignoreCert'))}
                disabled={isDisabled('ignoreCert')}
                onCheckedChange={(nextValue) => setField('ignoreCert', nextValue)}
                aria-label="Ignore server certificate errors"
              />
            </label>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Session"
        description="Set keyboard layout, timezone, and console-session behavior."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Keyboard Layout"
            description="Use the server default if you do not need a custom mapping."
            aside={overrideControl('serverLayout')}
          >
            <Select
              value={getValue('serverLayout') ?? NOT_SET_VALUE}
              onValueChange={(nextValue) =>
                setField('serverLayout', (nextValue === NOT_SET_VALUE ? undefined : nextValue) as RdpSettings['serverLayout'])
              }
              disabled={isDisabled('serverLayout')}
            >
              <SelectTrigger aria-label="Keyboard Layout">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NOT_SET_VALUE}>Default (en-us-qwerty)</SelectItem>
                {KEYBOARD_LAYOUTS.map((layout) => (
                  <SelectItem key={layout.value} value={layout.value}>
                    {layout.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Timezone"
            description="Useful for applications that depend on local time."
            aside={overrideControl('timezone')}
          >
            <Select
              value={getValue('timezone') ?? NOT_SET_VALUE}
              onValueChange={(nextValue) =>
                setField('timezone', (nextValue === NOT_SET_VALUE ? undefined : nextValue) as RdpSettings['timezone'])
              }
              disabled={isDisabled('timezone')}
            >
              <SelectTrigger aria-label="Timezone">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NOT_SET_VALUE}>Not set (server default)</SelectItem>
                {COMMON_TIMEZONES.map((timezone) => (
                  <SelectItem key={timezone} value={timezone}>
                    {timezone}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Console Session"
            description="Use the existing admin console session instead of starting a regular desktop."
            aside={overrideControl('console')}
            className="xl:col-span-2"
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Connect to the console / admin session</span>
              <Switch
                checked={Boolean(getValue('console'))}
                disabled={isDisabled('console')}
                onCheckedChange={(nextValue) => setField('console', nextValue)}
                aria-label="Console / admin session"
              />
            </label>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>
    </SettingsFieldGroup>
  );
}
