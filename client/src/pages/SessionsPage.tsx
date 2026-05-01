import WorkspaceShell from '@/components/Workspace/WorkspaceShell';
import { readSessionsRouteState } from '@/components/sessions/sessionConsoleRoute';
import { useWorkspaceBootstrap } from '@/hooks/useWorkspaceBootstrap';
import { useSearchParams } from 'react-router-dom';

export default function SessionsPage() {
  useWorkspaceBootstrap();
  const [searchParams] = useSearchParams();

  return (
    <WorkspaceShell
      initialSessionsDialogOpen
      initialSessionsDialogState={readSessionsRouteState(searchParams)}
    />
  );
}
