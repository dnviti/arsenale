import WorkspaceShell from '@/components/Workspace/WorkspaceShell';
import { useWorkspaceBootstrap } from '@/hooks/useWorkspaceBootstrap';

export default function DashboardPage() {
  useWorkspaceBootstrap({ enableAutoconnect: true });
  return <WorkspaceShell view="dashboard" />;
}
