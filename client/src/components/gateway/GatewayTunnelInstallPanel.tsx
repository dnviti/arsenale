import { useMemo } from 'react';
import { Copy, Terminal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import type { GatewayData, TunnelTokenResponse } from '../../api/gateway.api';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';
import { buildTunnelInstallBundle } from './gatewayTunnelInstall';

interface GatewayTunnelInstallPanelProps {
  gateway: GatewayData;
  tokenBundle: TunnelTokenResponse;
  serverUrl: string;
}

export default function GatewayTunnelInstallPanel({
  gateway,
  tokenBundle,
  serverUrl,
}: GatewayTunnelInstallPanelProps) {
  const install = useMemo(
    () => buildTunnelInstallBundle({ gateway, tokenBundle, serverUrl }),
    [gateway, tokenBundle, serverUrl],
  );
  const { copied: tokenCopied, copy: copyToken } = useCopyToClipboard();
  const { copied: envCopied, copy: copyEnv } = useCopyToClipboard();
  const { copied: commandCopied, copy: copyCommand } = useCopyToClipboard();
  const { copied: composeCopied, copy: copyCompose } = useCopyToClipboard();
  const { copied: certCopied, copy: copyCert } = useCopyToClipboard();
  const { copied: keyCopied, copy: copyKey } = useCopyToClipboard();

  return (
    <div className="space-y-3">
      <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-400">
        Copy these values now. The tunnel token and client key are only shown in this enrollment result.
      </div>

      <div className="space-y-1.5">
        <p className="text-xs font-medium">Tunnel token</p>
        <div className="flex gap-2">
          <Input value={tokenBundle.token} readOnly className="font-mono text-xs" />
          <Button size="sm" variant="outline" onClick={() => copyToken(tokenBundle.token)}>
            <Copy className="h-3.5 w-3.5 mr-1" />
            {tokenCopied ? 'Copied' : 'Copy'}
          </Button>
        </div>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between gap-2">
          <p className="text-xs font-medium">Remote install commands</p>
          <Button size="sm" variant="outline" onClick={() => copyCommand(install.installCommands)}>
            <Terminal className="h-3.5 w-3.5 mr-1" />
            {commandCopied ? 'Copied' : 'Copy'}
          </Button>
        </div>
        <Textarea value={install.installCommands} readOnly rows={12} className="font-mono text-[0.7rem]" />
        <p className="text-xs text-muted-foreground">
          Run these commands on the remote network host. When the container starts, the gateway appears as connected automatically.
        </p>
      </div>

      <Accordion type="single" collapsible>
        <AccordionItem value="details">
          <AccordionTrigger>
            <span className="text-sm font-medium">Tunnel connection details</span>
          </AccordionTrigger>
          <AccordionContent>
            <div className="space-y-3">
              <div>
                <div className="mb-1 flex items-center justify-between gap-2">
                  <p className="text-xs font-medium">tunnel.env</p>
                  <Button size="sm" variant="ghost" onClick={() => copyEnv(install.envContent)}>
                    <Copy className="h-3.5 w-3.5 mr-1" />
                    {envCopied ? 'Copied' : 'Copy'}
                  </Button>
                </div>
                <Textarea value={install.envContent} readOnly rows={7} className="font-mono text-[0.7rem]" />
              </div>

              <div>
                <div className="mb-1 flex items-center justify-between gap-2">
                  <p className="text-xs font-medium">docker-compose.yml</p>
                  <Button size="sm" variant="ghost" onClick={() => copyCompose(install.dockerCompose)}>
                    <Copy className="h-3.5 w-3.5 mr-1" />
                    {composeCopied ? 'Copied' : 'Copy'}
                  </Button>
                </div>
                <Textarea value={install.dockerCompose} readOnly rows={10} className="font-mono text-[0.7rem]" />
              </div>

              <div>
                <div className="mb-1 flex items-center justify-between gap-2">
                  <p className="text-xs font-medium">Client certificate</p>
                  <Button size="sm" variant="ghost" onClick={() => copyCert(tokenBundle.tunnelClientCert)}>
                    <Copy className="h-3.5 w-3.5 mr-1" />
                    {certCopied ? 'Copied' : 'Copy'}
                  </Button>
                </div>
                <Textarea value={tokenBundle.tunnelClientCert} readOnly rows={4} className="font-mono text-[0.7rem]" />
              </div>

              <div>
                <div className="mb-1 flex items-center justify-between gap-2">
                  <p className="text-xs font-medium">Client key</p>
                  <Button size="sm" variant="ghost" onClick={() => copyKey(tokenBundle.tunnelClientKey)}>
                    <Copy className="h-3.5 w-3.5 mr-1" />
                    {keyCopied ? 'Copied' : 'Copy'}
                  </Button>
                </div>
                <Textarea value={tokenBundle.tunnelClientKey} readOnly rows={4} className="font-mono text-[0.7rem]" />
              </div>
            </div>
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>
  );
}
