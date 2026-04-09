import { ThemeProvider } from '@/components/ui/material';
import { AccessTime, VisibilityOff } from '@/components/ui/material-icons';
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { themes } from '../../../theme';

describe('material icon compatibility adapter', () => {
  it('supports MUI-style fontSize and color props', () => {
    render(
      <ThemeProvider theme={themes.primer.dark}>
        <AccessTime data-testid="icon" fontSize="small" color="error" />
      </ThemeProvider>,
    );

    const icon = screen.getByTestId('icon');
    expect(icon).toHaveStyle({
      color: themes.primer.dark.palette.error.main,
      height: '16px',
      width: '16px',
    });
  });

  it('translates sx fontSize and color into SVG dimensions', () => {
    render(
      <ThemeProvider theme={themes.primer.dark}>
        <VisibilityOff
          data-testid="icon"
          sx={{ color: 'text.secondary', fontSize: 28 }}
        />
      </ThemeProvider>,
    );

    const icon = screen.getByTestId('icon');
    expect(icon).toHaveStyle({
      color: themes.primer.dark.palette.text.secondary,
      height: '28px',
      width: '28px',
    });
  });
});
