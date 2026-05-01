import { useMemo } from 'react';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import {
  FONT_FAMILIES,
  TERMINAL_DEFAULTS,
  THEME_PRESETS,
  THEME_PRESET_NAMES,
  type SshTerminalConfig,
  type TerminalThemeColors,
} from '../../constants/terminalThemes';
import { useThemeStore } from '../../store/themeStore';
import {
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsSectionBlock,
} from './settings-ui';
import { SettingsOverrideToggle, useOverrideableSettings } from './settings-overrides';
import {
  ANSI_COLOR_KEYS,
  TerminalLivePreview,
  TerminalThemeOptionCard,
  themeLabel,
} from './terminalSettingsPreview';

interface TerminalSettingsSectionProps {
  value: Partial<SshTerminalConfig>;
  onChange: (updated: Partial<SshTerminalConfig>) => void;
  mode: 'global' | 'connection';
  resolvedDefaults?: ReturnType<typeof import('../../constants/terminalThemes').mergeTerminalConfig>;
  enforcedFields?: Partial<SshTerminalConfig>;
}

const selectClassName = 'w-full';

export default function TerminalSettingsSection({
  value,
  onChange,
  mode,
  resolvedDefaults,
  enforcedFields,
}: TerminalSettingsSectionProps) {
  const defaults = resolvedDefaults ?? TERMINAL_DEFAULTS;
  const webUiMode = useThemeStore((state) => state.mode);
  const {
    getValue,
    isOverridden,
    isEnforced,
    isDisabled,
    setField,
    toggleOverride,
  } = useOverrideableSettings<SshTerminalConfig>({
    value,
    onChange,
    defaults,
    mode,
    enforcedFields,
  });

  const syncThemeWithWebUI = Boolean(getValue('syncThemeWithWebUI'));
  const currentTheme = useMemo(() => {
    if (syncThemeWithWebUI) {
      return webUiMode === 'light'
        ? getValue('syncLightTheme') ?? 'solarized-light'
        : getValue('syncDarkTheme') ?? 'default-dark';
    }
    return getValue('theme') ?? 'default-dark';
  }, [getValue, syncThemeWithWebUI, webUiMode]);

  const currentColors: TerminalThemeColors = useMemo(() => {
    if (currentTheme === 'custom') {
      return {
        ...defaults.customColors,
        ...value.customColors,
      };
    }
    return THEME_PRESETS[currentTheme] ?? THEME_PRESETS['default-dark'];
  }, [currentTheme, defaults.customColors, value.customColors]);

  const overrideControl = (key: keyof SshTerminalConfig) =>
    mode === 'connection' ? (
      <SettingsOverrideToggle
        checked={isOverridden(key)}
        enforced={isEnforced(key)}
        onCheckedChange={() => toggleOverride(key)}
      />
    ) : undefined;

  const updateCustomColor = (colorKey: keyof TerminalThemeColors, nextColor: string) => {
    onChange({
      ...value,
      customColors: {
        ...value.customColors,
        [colorKey]: nextColor,
      },
    });
  };

  return (
    <SettingsFieldGroup className="space-y-5">
      <SettingsSectionBlock
        title="Font"
        description="Tune the terminal’s density and readability."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Font Family"
            description="Choose the monospace stack used in terminal sessions."
            aside={overrideControl('fontFamily')}
          >
            <Select
              value={getValue('fontFamily') ?? TERMINAL_DEFAULTS.fontFamily}
              onValueChange={(nextValue) => setField('fontFamily', nextValue)}
              disabled={isDisabled('fontFamily')}
            >
              <SelectTrigger className={selectClassName} aria-label="Font Family">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FONT_FAMILIES.map((font) => (
                  <SelectItem key={font.value} value={font.value}>
                    <span style={{ fontFamily: font.value }}>{font.label}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Font Size"
            description={`Current size: ${getValue('fontSize') ?? 14}px.`}
            aside={overrideControl('fontSize')}
          >
            <Slider
              value={[getValue('fontSize') ?? 14]}
              min={10}
              max={24}
              step={1}
              disabled={isDisabled('fontSize')}
              onValueChange={([nextValue]) => setField('fontSize', nextValue)}
              aria-label="Font Size"
            />
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Line Height"
            description={`Current ratio: ${(getValue('lineHeight') ?? 1).toFixed(1)}.`}
            aside={overrideControl('lineHeight')}
          >
            <Slider
              value={[getValue('lineHeight') ?? 1]}
              min={1}
              max={2}
              step={0.1}
              disabled={isDisabled('lineHeight')}
              onValueChange={([nextValue]) => setField('lineHeight', nextValue)}
              aria-label="Line Height"
            />
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Letter Spacing"
            description={`Current spacing: ${getValue('letterSpacing') ?? 0}px.`}
            aside={overrideControl('letterSpacing')}
          >
            <Slider
              value={[getValue('letterSpacing') ?? 0]}
              min={0}
              max={5}
              step={1}
              disabled={isDisabled('letterSpacing')}
              onValueChange={([nextValue]) => setField('letterSpacing', nextValue)}
              aria-label="Letter Spacing"
            />
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Cursor"
        description="Pick how the insertion point looks and behaves."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Cursor Style"
            description="Block is best for dense terminal work; underline and bar are lighter."
            aside={overrideControl('cursorStyle')}
          >
            <ToggleGroup
              type="single"
              value={getValue('cursorStyle') ?? 'block'}
              onValueChange={(nextValue) => {
                if (nextValue) {
                  setField('cursorStyle', nextValue as SshTerminalConfig['cursorStyle']);
                }
              }}
              disabled={isDisabled('cursorStyle')}
              className="flex-wrap"
            >
              <ToggleGroupItem value="block" variant="outline">
                Block
              </ToggleGroupItem>
              <ToggleGroupItem value="underline" variant="outline">
                Underline
              </ToggleGroupItem>
              <ToggleGroupItem value="bar" variant="outline">
                Bar
              </ToggleGroupItem>
            </ToggleGroup>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Cursor Blink"
            description="Blinking makes the active prompt more obvious."
            aside={overrideControl('cursorBlink')}
          >
            <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
              <span className="text-sm text-muted-foreground">Animate the terminal cursor</span>
              <Switch
                checked={getValue('cursorBlink') ?? true}
                disabled={isDisabled('cursorBlink')}
                onCheckedChange={(nextValue) => setField('cursorBlink', nextValue)}
                aria-label="Cursor Blink"
              />
            </label>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Theme Sync"
        description="Keep the terminal in step with the WebUI’s light and dark mode."
      >
        <SettingsFieldCard
          label="Sync Terminal Theme"
          description="When enabled, the terminal automatically swaps between your chosen light and dark presets."
          aside={overrideControl('syncThemeWithWebUI')}
        >
          <label className="flex items-center justify-between gap-4 rounded-xl border border-border/70 bg-background px-4 py-3">
            <span className="text-sm text-muted-foreground">Follow the WebUI color mode</span>
            <Switch
              checked={syncThemeWithWebUI}
              disabled={isDisabled('syncThemeWithWebUI')}
              onCheckedChange={(nextValue) => setField('syncThemeWithWebUI', nextValue)}
              aria-label="Sync theme with WebUI light/dark mode"
            />
          </label>
        </SettingsFieldCard>

        {syncThemeWithWebUI && (
          <>
            <div className="grid gap-4 xl:grid-cols-2">
              <SettingsFieldCard
                label="Light Mode Theme"
                description="Preset used when the WebUI is in light mode."
                aside={overrideControl('syncLightTheme')}
              >
                <Select
                  value={getValue('syncLightTheme') ?? 'solarized-light'}
                  onValueChange={(nextValue) => setField('syncLightTheme', nextValue)}
                  disabled={isDisabled('syncLightTheme')}
                >
                  <SelectTrigger className={selectClassName} aria-label="Light Mode Theme">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {THEME_PRESET_NAMES.map((name) => (
                      <SelectItem key={name} value={name}>
                        {themeLabel(name)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </SettingsFieldCard>

              <SettingsFieldCard
                label="Dark Mode Theme"
                description="Preset used when the WebUI is in dark mode."
                aside={overrideControl('syncDarkTheme')}
              >
                <Select
                  value={getValue('syncDarkTheme') ?? 'default-dark'}
                  onValueChange={(nextValue) => setField('syncDarkTheme', nextValue)}
                  disabled={isDisabled('syncDarkTheme')}
                >
                  <SelectTrigger className={selectClassName} aria-label="Dark Mode Theme">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {THEME_PRESET_NAMES.map((name) => (
                      <SelectItem key={name} value={name}>
                        {themeLabel(name)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </SettingsFieldCard>
            </div>

            <p className="text-sm text-muted-foreground">
              Currently using {themeLabel(currentTheme)} because the WebUI is in {webUiMode} mode.
            </p>
          </>
        )}
      </SettingsSectionBlock>

      {!syncThemeWithWebUI && (
        <SettingsSectionBlock
          title="Color Theme"
          description="Pick a preset or use a fully custom palette."
        >
          {mode === 'connection' && (
            <div className="flex justify-end">
              {overrideControl('theme')}
            </div>
          )}

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {THEME_PRESET_NAMES.map((name) => (
              <TerminalThemeOptionCard
                key={name}
                label={themeLabel(name)}
                colors={THEME_PRESETS[name]}
                selected={currentTheme === name}
                disabled={isDisabled('theme')}
                onSelect={() => setField('theme', name)}
              />
            ))}
            <TerminalThemeOptionCard
              label="Custom"
              selected={currentTheme === 'custom'}
              disabled={isDisabled('theme')}
              description="Pick each ANSI color manually."
              onSelect={() => setField('theme', 'custom')}
            />
          </div>

          {currentTheme === 'custom' && !isDisabled('theme') && (
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              {ANSI_COLOR_KEYS.map((colorKey) => (
                <div
                  key={colorKey}
                  className="space-y-2 rounded-xl border border-border/70 bg-background p-3"
                >
                  <Label htmlFor={`terminal-color-${colorKey}`}>{colorKey}</Label>
                  <div className="flex items-center gap-3">
                    <input
                      id={`terminal-color-${colorKey}`}
                      type="color"
                      value={currentColors[colorKey]}
                      onChange={(event) => updateCustomColor(colorKey, event.target.value)}
                      className="h-10 w-14 cursor-pointer rounded-md border border-border bg-transparent p-1"
                    />
                    <code className="text-xs text-muted-foreground">{currentColors[colorKey]}</code>
                  </div>
                </div>
              ))}
            </div>
          )}
        </SettingsSectionBlock>
      )}

      <SettingsSectionBlock
        title="Performance"
        description="Control history depth and audible or visual feedback."
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Scrollback"
            description={`Retain ${getValue('scrollback') ?? 1000} lines in session history.`}
            aside={overrideControl('scrollback')}
          >
            <Slider
              value={[getValue('scrollback') ?? 1000]}
              min={100}
              max={10000}
              step={100}
              disabled={isDisabled('scrollback')}
              onValueChange={([nextValue]) => setField('scrollback', nextValue)}
              aria-label="Scrollback"
            />
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Bell Style"
            description="Choose what happens when the remote shell emits a bell."
            aside={overrideControl('bellStyle')}
          >
            <Select
              value={getValue('bellStyle') ?? 'none'}
              onValueChange={(nextValue) => setField('bellStyle', nextValue as SshTerminalConfig['bellStyle'])}
              disabled={isDisabled('bellStyle')}
            >
              <SelectTrigger className={selectClassName} aria-label="Bell Style">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                <SelectItem value="sound">Sound</SelectItem>
                <SelectItem value="visual">Visual</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldCard>
        </div>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Preview"
        description="Live preview of the effective terminal appearance."
      >
        <TerminalLivePreview
          colors={currentColors}
          fontFamily={getValue('fontFamily') ?? TERMINAL_DEFAULTS.fontFamily}
          fontSize={getValue('fontSize') ?? 14}
          lineHeight={getValue('lineHeight') ?? 1}
          letterSpacing={getValue('letterSpacing') ?? 0}
          cursorBlink={getValue('cursorBlink') ?? true}
        />
      </SettingsSectionBlock>
    </SettingsFieldGroup>
  );
}
