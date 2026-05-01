import { useEffect, useState } from 'react';
import { ArrowUpRight, Sparkles } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { checkVersion, type VersionInfo } from '@/api/version.api';
import { useAuthStore } from '@/store/authStore';
import { isAdminOrAbove } from '@/utils/roles';

export default function VersionIndicator() {
  const [info, setInfo] = useState<VersionInfo | null>(null);
  const tenantRole = useAuthStore((state) => state.user?.tenantRole);

  useEffect(() => {
    let cancelled = false;
    checkVersion()
      .then((version) => {
        if (!cancelled) {
          setInfo(version);
        }
      })
      .catch(() => {
        // Version checks are informational only.
      });

    return () => {
      cancelled = true;
    };
  }, []);

  if (!info) {
    return null;
  }

  const showUpdate = info.updateAvailable && isAdminOrAbove(tenantRole);

  return (
    <div className="flex items-center gap-2 border-t px-3 py-3">
      <Badge variant="outline" className="text-[11px] text-muted-foreground">
        v{info.current}
      </Badge>
      {showUpdate && info.latest && info.latestUrl ? (
        <a
          href={info.latestUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-[11px] font-medium text-primary transition-colors hover:bg-primary/15"
          title={`Update available: v${info.latest}`}
        >
          <Sparkles className="size-3.5" />
          v{info.latest}
          <ArrowUpRight className="size-3.5" />
        </a>
      ) : null}
    </div>
  );
}
