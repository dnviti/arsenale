import { useEffect, useState } from 'react';
import { Sparkles, Loader2 } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import {
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsLoadingState,
  SettingsPanel,
  SettingsStatusBadge,
  SettingsSwitchRow,
} from './settings-ui';
import { getAiConfig, updateAiConfig } from '../../api/aiQuery.api';
import type { AiConfig } from '../../api/aiQuery.api';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';

const PROVIDERS = [
  { value: 'none', label: 'None (Disabled)' },
  { value: 'anthropic', label: 'Anthropic (Claude)' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'ollama', label: 'Ollama (Local)' },
  { value: 'openai-compatible', label: 'OpenAI-Compatible' },
];

export default function AiQueryConfigSection() {
  const notify = useNotificationStore((state) => state.notify);
  const [config, setConfig] = useState<AiConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [provider, setProvider] = useState('none');
  const [apiKey, setApiKey] = useState('');
  const [modelId, setModelId] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [maxTokens, setMaxTokens] = useState(4000);
  const [dailyLimit, setDailyLimit] = useState(100);
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    getAiConfig()
      .then((nextConfig) => {
        setConfig(nextConfig);
        setProvider(nextConfig.provider);
        setModelId(nextConfig.modelId);
        setBaseUrl(nextConfig.baseUrl ?? '');
        setMaxTokens(nextConfig.maxTokensPerRequest);
        setDailyLimit(nextConfig.dailyRequestLimit);
        setEnabled(nextConfig.enabled);
        setApiKey('');
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setError('');
    setSaving(true);

    try {
      const payload: Record<string, unknown> = {
        provider,
        modelId,
        baseUrl: baseUrl || null,
        maxTokensPerRequest: maxTokens,
        dailyRequestLimit: dailyLimit,
        enabled,
      };

      if (apiKey) {
        payload.apiKey = apiKey;
      }

      const nextConfig = await updateAiConfig(payload);
      setConfig(nextConfig);
      setApiKey('');
      notify('AI configuration saved', 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to save AI configuration'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="AI Query"
        description="Model-backed assistance for database users."
      >
        <SettingsLoadingState message="Loading AI configuration..." />
      </SettingsPanel>
    );
  }

  return (
    <SettingsPanel
      title="AI Query"
      description="Configure natural-language SQL generation and the provider behind it."
      heading={(
        <div className="flex flex-wrap items-center gap-2">
          <SettingsStatusBadge tone={enabled ? 'success' : 'neutral'}>
            <Sparkles className="mr-1 size-3.5" />
            {enabled ? 'Enabled' : 'Disabled'}
          </SettingsStatusBadge>
          {provider !== 'none' && <SettingsStatusBadge tone="neutral">{provider}</SettingsStatusBadge>}
        </div>
      )}
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsFieldGroup>
        <SettingsSwitchRow
          title="Enable AI Query Generation"
          description="Allow users to ask questions in plain English and receive validated SELECT queries."
          checked={enabled}
          onCheckedChange={setEnabled}
        />

        <div className="grid gap-4 xl:grid-cols-2">
          <SettingsFieldCard
            label="AI Provider"
            description="Choose the backing provider or disable the feature entirely."
          >
            <Select value={provider} onValueChange={setProvider}>
              <SelectTrigger aria-label="AI Provider">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PROVIDERS.map((providerOption) => (
                  <SelectItem key={providerOption.value} value={providerOption.value}>
                    {providerOption.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsFieldCard>

          {provider !== 'none' && (
            <SettingsFieldCard
              label="Model"
              description="Leave empty to use the provider default."
            >
              <Input
                value={modelId}
                onChange={(event) => setModelId(event.target.value)}
                placeholder={provider === 'anthropic' ? 'claude-sonnet-4-20250514' : 'gpt-4o'}
                aria-label="AI Model"
              />
            </SettingsFieldCard>
          )}
        </div>

        {provider !== 'none' && (
          <>
            <SettingsFieldCard
              label="API Key"
              description={
                config?.hasApiKey
                  ? 'An API key is already configured. Leave this blank to keep it.'
                  : 'Required. The key is encrypted at rest.'
              }
            >
              <Input
                type="password"
                value={apiKey}
                onChange={(event) => setApiKey(event.target.value)}
                placeholder={config?.hasApiKey ? 'Leave empty to keep existing key' : 'Enter API key'}
                aria-label="API Key"
              />
            </SettingsFieldCard>

            {(provider === 'openai' || provider === 'ollama' || provider === 'openai-compatible') && (
              <SettingsFieldCard
                label="Base URL"
                description={
                  provider === 'ollama'
                    ? 'Required for Ollama.'
                    : 'Leave empty for the provider default API endpoint.'
                }
              >
                <Input
                  value={baseUrl}
                  onChange={(event) => setBaseUrl(event.target.value)}
                  placeholder={provider === 'ollama' ? 'http://localhost:11434' : 'https://api.openai.com/v1'}
                  aria-label="Base URL"
                />
              </SettingsFieldCard>
            )}

            <div className="grid gap-4 xl:grid-cols-2">
              <SettingsFieldCard
                label="Max Tokens"
                description="Upper bound for one generated response."
              >
                <Input
                  type="number"
                  min={100}
                  max={16000}
                  value={maxTokens}
                  onChange={(event) => setMaxTokens(Number.parseInt(event.target.value, 10) || 4000)}
                  aria-label="Max Tokens"
                />
              </SettingsFieldCard>

              <SettingsFieldCard
                label="Daily Request Limit"
                description="Tenant-wide cap per day."
              >
                <Input
                  type="number"
                  min={1}
                  max={10000}
                  value={dailyLimit}
                  onChange={(event) => setDailyLimit(Number.parseInt(event.target.value, 10) || 100)}
                  aria-label="Daily Request Limit"
                />
              </SettingsFieldCard>
            </div>
          </>
        )}

        <div className="flex justify-start">
          <Button type="button" onClick={handleSave} disabled={saving}>
            {saving && <Loader2 className="animate-spin" />}
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </SettingsFieldGroup>
    </SettingsPanel>
  );
}
