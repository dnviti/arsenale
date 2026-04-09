import { useEffect } from "react";
import { Loader2, Network } from "lucide-react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { useGatewayStore } from "@/store/gatewayStore";
import { useAuthStore } from "@/store/authStore";
import {
  gatewayStatusBadgeClass,
  gatewayStatusLabel,
} from "@/utils/gatewayStatus";

interface GatewaysSidePanelProps {
  onSelectGateway?: (gatewayId: string) => void;
}

export default function GatewaysSidePanel({
  onSelectGateway,
}: GatewaysSidePanelProps) {
  const gateways = useGatewayStore((s) => s.gateways);
  const loading = useGatewayStore((s) => s.loading);
  const fetchGateways = useGatewayStore((s) => s.fetchGateways);
  const hasTenant = useAuthStore((s) => Boolean(s.user?.tenantId));
  const canManageGateways = useAuthStore(
    (s) => s.permissions.canManageGateways,
  );
  const permissionsLoaded = useAuthStore((s) => s.permissionsLoaded);

  useEffect(() => {
    if (hasTenant && permissionsLoaded && canManageGateways) {
      fetchGateways();
    }
  }, [hasTenant, permissionsLoaded, canManageGateways, fetchGateways]);

  const noAccess = !hasTenant || (permissionsLoaded && !canManageGateways);

  return (
    <SidebarGroup>
      <SidebarGroupLabel>
        <Network className="size-4" />
        Gateways
        {gateways.length > 0 && (
          <Badge
            variant="secondary"
            className="ml-auto text-[10px] px-1.5 py-0"
          >
            {gateways.length}
          </Badge>
        )}
      </SidebarGroupLabel>
      <SidebarGroupContent>
        {noAccess ? (
          <div className="px-2 py-4 text-center text-xs text-muted-foreground">
            Gateway access is restricted.
          </div>
        ) : loading ? (
          <div className="flex justify-center py-6">
            <Loader2 className="size-4 animate-spin text-muted-foreground" />
          </div>
        ) : gateways.length === 0 ? (
          <div className="px-2 py-4 text-center text-xs text-muted-foreground">
            No gateways configured.
          </div>
        ) : (
          <ScrollArea className="h-[calc(100vh-10rem)]">
            <SidebarMenu>
              {gateways.map((gw) => (
                <SidebarMenuItem key={gw.id}>
                  <SidebarMenuButton
                    tooltip={`${gw.name} (${gw.type})`}
                    onClick={() => onSelectGateway?.(gw.id)}
                    className="h-auto py-1.5"
                  >
                    <Network className="size-4 shrink-0 text-muted-foreground" />
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="truncate text-xs font-medium">
                        {gw.name}
                      </span>
                      <span className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
                        <Badge
                          className={`px-1 py-0 text-[9px] leading-tight ${gatewayStatusBadgeClass(gw.operationalStatus)}`}
                        >
                          {gatewayStatusLabel(gw.operationalStatus)}
                        </Badge>
                        {gw.deploymentMode === "MANAGED_GROUP" &&
                          !gw.tunnelEnabled && (
                            <span>
                              {gw.healthyInstances}/{gw.totalInstances} healthy
                            </span>
                          )}
                      </span>
                    </div>
                  </SidebarMenuButton>
                  {gw.tunnelConnected && (
                    <SidebarMenuBadge className="text-[9px] text-green-400">
                      tunnel
                    </SidebarMenuBadge>
                  )}
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </ScrollArea>
        )}
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
