import { useCallback, useEffect, useState } from 'react';
import { Loader2, Monitor, Play, Terminal, Video } from 'lucide-react';
import { SidebarGroup, SidebarGroupContent, SidebarGroupLabel, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { listRecordings } from '@/api/recordings.api';
import type { Recording } from '@/api/recordings.api';

const protocolIcon: Record<string, React.ElementType> = {
  SSH: Terminal,
  RDP: Monitor,
  VNC: Monitor,
};

const protocolBadgeClass: Record<string, string> = {
  SSH: 'bg-green-500/15 text-green-400 border-green-500/30',
  RDP: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  VNC: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
};

function formatDuration(seconds: number | null): string {
  if (seconds === null) return '--:--';
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

interface RecordingsSidePanelProps {
  onPlayRecording?: (recording: Recording) => void;
}

export default function RecordingsSidePanel({ onPlayRecording }: RecordingsSidePanelProps) {
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listRecordings({ status: 'COMPLETE', limit: 50 });
      setRecordings(result.recordings);
    } catch {
      // silently handle
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return (
    <SidebarGroup>
      <SidebarGroupLabel>
        <Video className="size-4" />
        Recordings
        {recordings.length > 0 && (
          <Badge variant="secondary" className="ml-auto text-[10px] px-1.5 py-0">
            {recordings.length}
          </Badge>
        )}
      </SidebarGroupLabel>
      <SidebarGroupContent>
        {loading ? (
          <div className="flex justify-center py-6">
            <Loader2 className="size-4 animate-spin text-muted-foreground" />
          </div>
        ) : recordings.length === 0 ? (
          <div className="px-2 py-4 text-center text-xs text-muted-foreground">
            No session recordings found.
          </div>
        ) : (
          <ScrollArea className="h-[calc(100vh-10rem)]">
            <SidebarMenu>
              {recordings.map((rec) => {
                const Icon = protocolIcon[rec.protocol] ?? Monitor;
                return (
                  <SidebarMenuItem key={rec.id}>
                    <SidebarMenuButton
                      tooltip={`${rec.connection.name} - ${rec.protocol}`}
                      onClick={() => onPlayRecording?.(rec)}
                      className="h-auto py-1.5"
                    >
                      <Icon className="size-4 shrink-0 text-muted-foreground" />
                      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                        <span className="truncate text-xs font-medium">
                          {rec.connection.name}
                        </span>
                        <span className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
                          <Badge className={`px-1 py-0 text-[9px] leading-tight ${protocolBadgeClass[rec.protocol] ?? ''}`}>
                            {rec.protocol}
                          </Badge>
                          <span>{formatDate(rec.createdAt)}</span>
                          <span className="flex items-center gap-0.5">
                            <Play className="size-2.5" />
                            {formatDuration(rec.duration)}
                          </span>
                        </span>
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </ScrollArea>
        )}
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
