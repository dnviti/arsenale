import { fireEvent } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { useState } from 'react';
import { useOverrideableSettings } from './settings-overrides';

interface HarnessSettings {
  width?: number;
}

function OverrideHarness() {
  const [value, setValue] = useState<Partial<HarnessSettings>>({});
  const { getValue, isOverridden, toggleOverride } = useOverrideableSettings<HarnessSettings>({
    value,
    onChange: setValue,
    defaults: { width: 1280 },
    mode: 'connection',
  });

  return (
    <div>
      <button type="button" onClick={() => toggleOverride('width', undefined)}>
        Toggle Width Override
      </button>
      <div>Overridden: {String(isOverridden('width'))}</div>
      <div>Value: {String(getValue('width'))}</div>
    </div>
  );
}

describe('useOverrideableSettings', () => {
  it('treats an explicit undefined override as overridden so auto-mode fields stay editable', () => {
    render(<OverrideHarness />);

    fireEvent.click(screen.getByRole('button', { name: 'Toggle Width Override' }));

    expect(screen.getByText('Overridden: true')).toBeInTheDocument();
    expect(screen.getByText('Value: undefined')).toBeInTheDocument();
  });
});
