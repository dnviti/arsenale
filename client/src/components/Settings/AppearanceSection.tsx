import { Check, MoonStar, SunMedium } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { cn } from '@/lib/utils';
import { useThemeStore } from '../../store/themeStore';
import { themeRegistry, type ThemeName, type ThemeMode } from '../../theme/index';

export default function AppearanceSection() {
  const themeName = useThemeStore((s) => s.themeName);
  const mode = useThemeStore((s) => s.mode);
  const setTheme = useThemeStore((s) => s.setTheme);
  const setMode = useThemeStore((s) => s.setMode);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Appearance</CardTitle>
        <CardDescription>
          Select a theme and color mode for the interface.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {themeRegistry.map((t) => {
            const isSelected = t.name === themeName;
            const swatchColor = mode === 'dark' ? t.accent : t.accentLight;

            return (
              <button
                key={t.name}
                type="button"
                onClick={() => setTheme(t.name as ThemeName)}
                className={cn(
                  'group relative rounded-xl border bg-card px-4 py-4 text-left shadow-sm transition-[border-color,box-shadow,transform] hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
                  isSelected && 'border-primary/70 shadow-[0_0_0_1px_var(--primary)]',
                )}
              >
                {isSelected && (
                  <span className="absolute right-3 top-3 inline-flex size-6 items-center justify-center rounded-full bg-primary text-primary-foreground">
                    <Check className="size-3.5" />
                  </span>
                )}

                <span
                  className="mb-3 inline-flex size-9 rounded-full ring-4 ring-background"
                  style={{
                    backgroundColor: swatchColor,
                    boxShadow: `0 0 16px ${swatchColor}40`,
                  }}
                  aria-hidden="true"
                />

                <div className="space-y-1">
                  <div className="truncate text-sm font-semibold text-foreground">
                    {t.label}
                  </div>
                  <div className="truncate text-xs text-muted-foreground">
                    {t.description}
                  </div>
                </div>
              </button>
            );
          })}
        </div>

        <div className="space-y-2">
          <div className="text-sm font-medium">
          Color mode
          </div>
          <ToggleGroup
            type="single"
            value={mode}
            onValueChange={(value) => {
              if (value) setMode(value as ThemeMode);
            }}
            aria-label="Choose a color mode"
          >
            <ToggleGroupItem value="dark" aria-label="Dark mode">
              <MoonStar className="size-4" />
              Dark
            </ToggleGroupItem>
            <ToggleGroupItem value="light" aria-label="Light mode">
              <SunMedium className="size-4" />
              Light
            </ToggleGroupItem>
          </ToggleGroup>
        </div>
      </CardContent>
    </Card>
  );
}
