import {
  DatabaseZap,
  History,
  KeyRound,
  Menu,
  Monitor,
  MoonStar,
  Network,
  Settings2,
  SunMedium,
  TerminalSquare,
  Video,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  SidebarSeparator,
  useSidebar,
} from '@/components/ui/sidebar';
import type { ConnectionData } from '@/api/connections.api';
import type { Folder } from '@/store/connectionsStore';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import { useThemeStore } from '@/store/themeStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import VersionIndicator from '../Layout/VersionIndicator';
import ConnectionsSidePanel from '../SidePanels/ConnectionsSidePanel';

export type ConnectionFilter = 'remote' | 'database';

interface AppSidebarProps {
  onEditConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onCreateConnection: (folderId?: string, teamId?: string) => void;
  onCreateFolder: (parentId?: string, teamId?: string) => void;
  onEditFolder: (folder: Folder) => void;
  onShareFolder: (folderId: string, folderName: string) => void;
  onViewAuditLog?: (conn: ConnectionData) => void;
  onOpenSettings: (tab?: string) => void;
  onOpenKeychain: () => void;
  onOpenAuditLog: () => void;
  onOpenRecordings: () => void;
}

export default function AppSidebar({
  onEditConnection,
  onShareConnection,
  onConnectAsConnection,
  onCreateConnection,
  onCreateFolder,
  onEditFolder,
  onShareFolder,
  onViewAuditLog,
  onOpenSettings,
  onOpenKeychain,
  onOpenAuditLog,
  onOpenRecordings,
}: AppSidebarProps) {
  const connectionsEnabled = useFeatureFlagsStore((s) => s.connectionsEnabled);
  const databaseProxyEnabled = useFeatureFlagsStore((s) => s.databaseProxyEnabled);
  const keychainEnabled = useFeatureFlagsStore((s) => s.keychainEnabled);
  const recordingsEnabled = useFeatureFlagsStore((s) => s.recordingsEnabled);
  const themeMode = useThemeStore((s) => s.mode);
  const toggleTheme = useThemeStore((s) => s.toggle);
  const activeFilter = useUiPreferencesStore((s) => s.workspaceActiveView) as ConnectionFilter;
  const setPreference = useUiPreferencesStore((s) => s.set);
  const { setOpen } = useSidebar();

  const setFilter = (filter: ConnectionFilter) => {
    setPreference('workspaceActiveView', filter);
  };

  return (
    <Sidebar collapsible="icon" className="border-r">
      <SidebarHeader>
        <SidebarMenu>
          {/* Main menu button with multilevel dropdown */}
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton tooltip="Menu">
                  <Menu className="size-4" />
                  <span className="font-heading text-sm tracking-tight">Arsenale</span>
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent side="right" align="start" className="w-56">
                {/* Connections submenu */}
                {/*
                {anyConnectionFeature ? (
                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                      <TerminalSquare className="size-4" />
                      Connections
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent>
                      {connectionsEnabled ? (
                        <DropdownMenuItem onSelect={() => setFilter('remote')}>
                          <Monitor className="size-4" />
                          Remote Control
                          <span className="ml-auto text-xs text-muted-foreground">SSH/RDP/VNC</span>
                        </DropdownMenuItem>
                      ) : null}
                      {databaseProxyEnabled ? (
                        <DropdownMenuItem onSelect={() => setFilter('database')}>
                          <DatabaseZap className="size-4" />
                          Database Proxy
                          <span className="ml-auto text-xs text-muted-foreground">DB</span>
                        </DropdownMenuItem>
                      ) : null}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                ) : null}
                */}
                {/* Security submenu */}
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <KeyRound className="size-4" />
                    Security
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    {keychainEnabled ? (
                      <DropdownMenuItem onSelect={onOpenKeychain}>
                        <KeyRound className="size-4" />
                        Vault / Keychain
                      </DropdownMenuItem>
                    ) : null}
                    <DropdownMenuItem onSelect={onOpenAuditLog}>
                      <History className="size-4" />
                      Activity Log
                    </DropdownMenuItem>
                  </DropdownMenuSubContent>
                </DropdownMenuSub>

                {/* Operations submenu */}
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <Network className="size-4" />
                    Operations
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    <DropdownMenuItem onSelect={() => onOpenSettings('infrastructure')}>
                      <Network className="size-4" />
                      Gateways
                    </DropdownMenuItem>
                    {recordingsEnabled ? (
                      <DropdownMenuItem onSelect={onOpenRecordings}>
                        <Video className="size-4" />
                        Recordings
                      </DropdownMenuItem>
                    ) : null}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>

                <DropdownMenuSeparator />

                <DropdownMenuItem onSelect={() => onOpenSettings()}>
                  <Settings2 className="size-4" />
                  Settings
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={toggleTheme}>
                  {themeMode === 'dark' ? <SunMedium className="size-4" /> : <MoonStar className="size-4" />}
                  {themeMode === 'dark' ? 'Light Mode' : 'Dark Mode'}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>

        {/* Connection filter tabs */}
        <div className='px-2'>
          <SidebarSeparator />
        </div>
        <SidebarMenu>
          {connectionsEnabled ? (
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={activeFilter === 'remote'}
                tooltip="Remote Control (SSH/RDP/VNC)"
                onClick={() => setFilter('remote')}
                size="sm"
              >
                <TerminalSquare className="size-4" />
                <span>Remote Control</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ) : null}
          {databaseProxyEnabled ? (
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={activeFilter === 'database'}
                tooltip="Database Proxy"
                onClick={() => setFilter('database')}
                size="sm"
              >
                <DatabaseZap className="size-4" />
                <span>Database Proxy</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ) : null}
        </SidebarMenu>
      </SidebarHeader>

      <div className='px-2'>
        <SidebarSeparator />
      </div>

      <SidebarContent>
        <div className="group-data-[collapsible=icon]:hidden">
          <ConnectionsSidePanel
            typeFilter={activeFilter}
            onEditConnection={onEditConnection}
            onShareConnection={onShareConnection}
            onConnectAsConnection={onConnectAsConnection}
            onCreateConnection={onCreateConnection}
            onCreateFolder={onCreateFolder}
            onEditFolder={onEditFolder}
            onShareFolder={onShareFolder}
            onViewAuditLog={onViewAuditLog}
          />
        </div>
        <div className="hidden group-data-[collapsible=icon]:block px-2 pt-2">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton tooltip="My Connections" onClick={() => setOpen(true)}>
                <Monitor className="size-4" />
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </div>
      </SidebarContent>

      <SidebarFooter className="group-data-[collapsible=icon]:hidden">
        <VersionIndicator />
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  );
}
