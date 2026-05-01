import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Input } from '@/components/ui/input';
import {
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsPanel,
  SettingsSwitchRow,
} from './settings-ui';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';

const THEME_OPTIONS = [
  { value: 'auto', label: 'Auto (sync with WebUI)' },
  { value: 'vs', label: 'Light' },
  { value: 'vs-dark', label: 'Dark' },
  { value: 'dracula', label: 'Dracula' },
  { value: 'solarized', label: 'Solarized' },
];

export default function SqlEditorSection() {
  const sqlEditorTheme = useUiPreferencesStore((state) => state.sqlEditorTheme);
  const sqlEditorFontSize = useUiPreferencesStore((state) => state.sqlEditorFontSize);
  const sqlEditorFontFamily = useUiPreferencesStore((state) => state.sqlEditorFontFamily);
  const sqlEditorMinimap = useUiPreferencesStore((state) => state.sqlEditorMinimap);
  const setPreference = useUiPreferencesStore((state) => state.set);

  return (
    <SettingsPanel
      title="SQL Editor"
      description="Customize Monaco-based SQL editing without affecting the rest of the interface."
      contentClassName="space-y-4"
    >
      <SettingsFieldGroup>
        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="Theme"
            description="Use the WebUI theme or pin a dedicated editor palette."
          >
            <Select
              value={sqlEditorTheme}
              onValueChange={(value) => setPreference('sqlEditorTheme', value)}
            >
              <SelectTrigger aria-label="SQL Editor Theme">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {THEME_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          <SettingsFieldCard
            label="Font Size"
            description={`Current size: ${sqlEditorFontSize}px.`}
          >
            <Slider
              value={[sqlEditorFontSize]}
              min={10}
              max={24}
              step={1}
              onValueChange={([value]) => setPreference('sqlEditorFontSize', value)}
              aria-label="SQL Editor Font Size"
            />
          </SettingsFieldCard>
        </div>

        <SettingsFieldCard
          label="Font Family"
          description="Comma-separated fallback list for the editor font stack."
        >
          <Input
            value={sqlEditorFontFamily}
            onChange={(event) => setPreference('sqlEditorFontFamily', event.target.value)}
            aria-label="SQL Editor Font Family"
          />
        </SettingsFieldCard>

        <SettingsSwitchRow
          title="Show minimap"
          description="Display a compact code overview rail beside the editor."
          checked={sqlEditorMinimap}
          onCheckedChange={(checked) => setPreference('sqlEditorMinimap', checked)}
        />
      </SettingsFieldGroup>
    </SettingsPanel>
  );
}
