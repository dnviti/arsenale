import { SidebarGroup, SidebarGroupContent } from '@/components/ui/sidebar';
import type { ConnectionData } from '@/api/connections.api';
import type { Folder } from '@/store/connectionsStore';
import ConnectionTree from '../Sidebar/ConnectionTree';
import type { ConnectionFilter } from '../Workspace/AppSidebar';

interface ConnectionsSidePanelProps {
  typeFilter: ConnectionFilter;
  onEditConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onCreateConnection: (folderId?: string, teamId?: string) => void;
  onCreateFolder: (parentId?: string, teamId?: string) => void;
  onEditFolder: (folder: Folder) => void;
  onShareFolder: (folderId: string, folderName: string) => void;
  onViewAuditLog?: (conn: ConnectionData) => void;
}

export default function ConnectionsSidePanel({
  typeFilter,
  onEditConnection,
  onShareConnection,
  onConnectAsConnection,
  onCreateConnection,
  onCreateFolder,
  onEditFolder,
  onShareFolder,
  onViewAuditLog,
}: ConnectionsSidePanelProps) {
  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <ConnectionTree
          typeFilter={typeFilter === 'remote' ? ['SSH', 'RDP', 'VNC'] : ['DATABASE', 'DB_TUNNEL']}
          onEditConnection={onEditConnection}
          onShareConnection={onShareConnection}
          onConnectAsConnection={onConnectAsConnection}
          onCreateConnection={onCreateConnection}
          onCreateFolder={onCreateFolder}
          onEditFolder={onEditFolder}
          onShareFolder={onShareFolder}
          onViewAuditLog={onViewAuditLog}
        />
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
