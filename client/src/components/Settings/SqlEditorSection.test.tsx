import { fireEvent } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import SqlEditorSection from './SqlEditorSection';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';

describe('SqlEditorSection', () => {
  beforeEach(() => {
    useUiPreferencesStore.setState({
      sqlEditorTheme: 'auto',
      sqlEditorFontSize: 14,
      sqlEditorFontFamily: 'Cascadia Code, monospace',
      sqlEditorMinimap: false,
    });
  });

  it('updates editor preferences through the shared store', () => {
    render(<SqlEditorSection />);

    fireEvent.change(screen.getByLabelText('SQL Editor Font Family'), {
      target: { value: 'JetBrains Mono, monospace' },
    });
    fireEvent.click(screen.getByRole('switch', { name: 'Show minimap' }));

    expect(useUiPreferencesStore.getState().sqlEditorFontFamily).toBe('JetBrains Mono, monospace');
    expect(useUiPreferencesStore.getState().sqlEditorMinimap).toBe(true);
  });
});
