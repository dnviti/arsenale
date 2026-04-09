import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from '@/components/ui/accordion';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { isGatewayGroup } from '../../utils/gatewayMode';
import SessionDashboard from './SessionDashboard';
import GatewayInstanceList from './GatewayInstanceList';
import ScalingControls from './ScalingControls';

interface OrchestrationSectionProps {
  onNavigateToTab?: (tabId: string) => void;
}

export default function OrchestrationSection({ onNavigateToTab }: OrchestrationSectionProps) {
  const user = useAuthStore((s) => s.user);
  const gateways = useGatewayStore((s) => s.gateways);
  const fetchGateways = useGatewayStore((s) => s.fetchGateways);

  const subTab = useUiPreferencesStore((s) => s.orchestrationDashboardTab);
  const setSubTab = useUiPreferencesStore((s) => s.set);

  const hasTenant = Boolean(user?.tenantId);

  useEffect(() => {
    if (hasTenant) {
      fetchGateways();
    }
  }, [hasTenant, fetchGateways]);

  if (!hasTenant) {
    return (
      <div className="text-center py-12">
        <h3 className="text-lg font-semibold mb-2">No Organization</h3>
        <p className="text-sm text-muted-foreground mb-6">
          You need to create or join an organization before using orchestration features.
        </p>
        <Button onClick={() => onNavigateToTab?.('organization')}>
          Set Up Organization
        </Button>
      </div>
    );
  }

  const managedGateways = gateways.filter(
    (g) => isGatewayGroup(g) && (g.type === 'MANAGED_SSH' || g.type === 'GUACD' || g.type === 'DB_PROXY'),
  );

  return (
    <div>
      <h3 className="text-lg font-semibold mb-3">Orchestration</h3>

      <Tabs value={subTab} onValueChange={(v) => setSubTab('orchestrationDashboardTab', v)}>
        <TabsList>
          <TabsTrigger value="sessions">Active Sessions</TabsTrigger>
          <TabsTrigger value="gateways">Gateway Scaling</TabsTrigger>
        </TabsList>

        <TabsContent value="sessions">
          <SessionDashboard />
        </TabsContent>

        <TabsContent value="gateways">
          {managedGateways.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-muted-foreground">
                No deployable gateways found. Gateways of type MANAGED_SSH or GUACD can be managed here.
              </p>
            </div>
          ) : (
            <Accordion type="multiple" defaultValue={managedGateways.filter(isGatewayGroup).map(g => g.id)}>
              {managedGateways.map((gw) => (
                <AccordionItem key={gw.id} value={gw.id}>
                  <AccordionTrigger>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{gw.name}</span>
                      <Badge variant="outline">{gw.type}</Badge>
                      {isGatewayGroup(gw) && (
                        <Badge>Managed</Badge>
                      )}
                      {isGatewayGroup(gw) && (
                        <span className="text-xs text-muted-foreground">
                          {gw.runningInstances}/{gw.totalInstances} instances
                        </span>
                      )}
                    </div>
                  </AccordionTrigger>
                  <AccordionContent>
                    <ScalingControls gatewayId={gw.id} gateway={gw} />
                    {isGatewayGroup(gw) && gw.totalInstances > 0 && (
                      <div className="mt-4">
                        <p className="text-sm font-medium mb-2">Instances</p>
                        <GatewayInstanceList gatewayId={gw.id} />
                      </div>
                    )}
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
