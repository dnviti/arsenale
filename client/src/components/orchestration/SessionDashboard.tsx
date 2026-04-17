import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowRight, Eye, Pause, ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { buildSessionsRoute } from '@/components/sessions/sessionConsoleRoute';

interface SessionDashboardProps {
  onOpenSessions?: () => void;
}

export default function SessionDashboard({ onOpenSessions }: SessionDashboardProps) {
  const sessionCount = useGatewayStore((s) => s.sessionCount);
  const canObserveSessions = useAuthStore((s) => s.permissions.canObserveSessions);
  const canControlSessions = useAuthStore((s) => s.permissions.canControlSessions);
  const navigate = useNavigate();

  const capabilityHint = useMemo(() => {
    if (canControlSessions) {
      return 'Pause, resume, stop, review recordings, and audit closed sessions from one console.';
    }
    if (canObserveSessions) {
      return 'Review live sessions and recording history with read-only controls.';
    }
    return 'Review the unified sessions console for current visibility.';
  }, [canControlSessions, canObserveSessions]);

  return (
    <Card className="border-border/70 bg-card/70">
      <CardHeader>
        <CardTitle>Sessions console</CardTitle>
        <CardDescription>{capabilityHint}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="grid gap-3 sm:grid-cols-3">
          <SummaryPill label="Live sessions" value={sessionCount} icon={<ShieldCheck className="size-4" />} />
          <SummaryPill label="Observe" value={canObserveSessions ? 'Enabled' : 'Disabled'} icon={<Eye className="size-4" />} />
          <SummaryPill label="Control" value={canControlSessions ? 'Enabled' : 'Read only'} icon={<Pause className="size-4" />} />
        </div>
        <Button
          type="button"
          onClick={() => {
            if (onOpenSessions) {
              onOpenSessions();
              return;
            }
            navigate(buildSessionsRoute());
          }}
          className="gap-2 self-start md:self-auto"
        >
          Open sessions console
          <ArrowRight className="size-4" />
        </Button>
      </CardContent>
    </Card>
  );
}

function SummaryPill({
  label,
  value,
  icon,
}: {
  label: string;
  value: number | string;
  icon: React.ReactElement;
}) {
  return (
    <div className="flex min-w-[10rem] items-center gap-3 rounded-xl border border-border/70 bg-muted/20 px-4 py-3">
      <span className="rounded-full border border-border/70 bg-background/70 p-2 text-muted-foreground">
        {icon}
      </span>
      <div>
        <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">{label}</div>
        <div className="mt-1 text-sm font-semibold text-foreground">{value}</div>
      </div>
    </div>
  );
}
