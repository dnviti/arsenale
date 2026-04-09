import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AiQueryConfigSection from './AiQueryConfigSection';
import { useNotificationStore } from '../../store/notificationStore';

const { getAiConfig, updateAiConfig } = vi.hoisted(() => ({
  getAiConfig: vi.fn(),
  updateAiConfig: vi.fn(),
}));

vi.mock('../../api/aiQuery.api', () => ({
  getAiConfig,
  updateAiConfig,
}));

describe('AiQueryConfigSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });

    getAiConfig.mockResolvedValue({
      provider: 'openai',
      hasApiKey: true,
      modelId: 'gpt-4o',
      baseUrl: null,
      maxTokensPerRequest: 4000,
      dailyRequestLimit: 100,
      enabled: false,
    });
    updateAiConfig.mockResolvedValue({
      provider: 'openai',
      hasApiKey: true,
      modelId: 'gpt-4.1',
      baseUrl: null,
      maxTokensPerRequest: 4000,
      dailyRequestLimit: 100,
      enabled: true,
    });
  });

  it('loads AI settings and saves the updated configuration', async () => {
    render(<AiQueryConfigSection />);

    fireEvent.click(await screen.findByRole('switch', { name: 'Enable AI Query Generation' }));
    fireEvent.change(screen.getByLabelText('AI Model'), {
      target: { value: 'gpt-4.1' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateAiConfig).toHaveBeenCalledWith({
        provider: 'openai',
        modelId: 'gpt-4.1',
        baseUrl: null,
        maxTokensPerRequest: 4000,
        dailyRequestLimit: 100,
        enabled: true,
      });
    });

    expect(useNotificationStore.getState().notification).toMatchObject({
      message: 'AI configuration saved',
      severity: 'success',
    });
  });
});
