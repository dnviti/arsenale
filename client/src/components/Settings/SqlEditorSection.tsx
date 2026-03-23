import {
  Card, CardContent, Typography, Box, Slider, TextField,
  FormControlLabel, Switch, Select, MenuItem, InputLabel, FormControl,
} from '@mui/material';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';

const THEME_OPTIONS = [
  { value: 'auto', label: 'Auto (sync with WebUI)' },
  { value: 'vs', label: 'Light' },
  { value: 'vs-dark', label: 'Dark' },
  { value: 'dracula', label: 'Dracula' },
  { value: 'solarized', label: 'Solarized' },
];

export default function SqlEditorSection() {
  const sqlEditorTheme = useUiPreferencesStore((s) => s.sqlEditorTheme);
  const sqlEditorFontSize = useUiPreferencesStore((s) => s.sqlEditorFontSize);
  const sqlEditorFontFamily = useUiPreferencesStore((s) => s.sqlEditorFontFamily);
  const sqlEditorMinimap = useUiPreferencesStore((s) => s.sqlEditorMinimap);
  const setPref = useUiPreferencesStore((s) => s.set);

  return (
    <Card variant="outlined">
      <CardContent>
        <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
          SQL Editor
        </Typography>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          Customize the Monaco-based SQL editor appearance and behavior.
        </Typography>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, mt: 2 }}>
          {/* Theme selector */}
          <FormControl size="small" fullWidth>
            <InputLabel id="sql-editor-theme-label">Theme</InputLabel>
            <Select
              labelId="sql-editor-theme-label"
              value={sqlEditorTheme}
              label="Theme"
              onChange={(e) => setPref('sqlEditorTheme', e.target.value)}
            >
              {THEME_OPTIONS.map((opt) => (
                <MenuItem key={opt.value} value={opt.value}>
                  {opt.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          {/* Font size slider */}
          <Box>
            <Typography variant="body2" gutterBottom>
              Font Size: {sqlEditorFontSize}px
            </Typography>
            <Slider
              aria-label="Font Size"
              value={sqlEditorFontSize}
              onChange={(_, val) => setPref('sqlEditorFontSize', val as number)}
              min={10}
              max={24}
              step={1}
              marks={[
                { value: 10, label: '10' },
                { value: 14, label: '14' },
                { value: 18, label: '18' },
                { value: 24, label: '24' },
              ]}
              valueLabelDisplay="auto"
              size="small"
            />
          </Box>

          {/* Font family */}
          <TextField
            label="Font Family"
            size="small"
            fullWidth
            value={sqlEditorFontFamily}
            onChange={(e) => setPref('sqlEditorFontFamily', e.target.value)}
            helperText="Comma-separated list of font families"
          />

          {/* Minimap toggle */}
          <FormControlLabel
            control={
              <Switch
                checked={sqlEditorMinimap}
                onChange={(_, checked) => setPref('sqlEditorMinimap', checked)}
              />
            }
            label="Show minimap"
          />
        </Box>
      </CardContent>
    </Card>
  );
}
