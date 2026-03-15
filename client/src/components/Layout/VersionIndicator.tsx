import { useEffect, useState } from 'react';
import { Box, Chip, Link, Tooltip } from '@mui/material';
import { NewReleases as NewReleasesIcon } from '@mui/icons-material';
import { checkVersion, type VersionInfo } from '../../api/version.api';
import { useAuthStore } from '../../store/authStore';
import { isAdminOrAbove } from '../../utils/roles';

export default function VersionIndicator() {
  const [info, setInfo] = useState<VersionInfo | null>(null);
  const tenantRole = useAuthStore((s) => s.user?.tenantRole);

  useEffect(() => {
    let cancelled = false;
    checkVersion()
      .then((v) => { if (!cancelled) setInfo(v); })
      .catch(() => { /* silent */ });
    return () => { cancelled = true; };
  }, []);

  if (!info) return null;

  const showUpdate = info.updateAvailable && isAdminOrAbove(tenantRole);

  return (
    <Box
      sx={{
        px: 1.5,
        py: 0.75,
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        borderTop: 1,
        borderColor: 'divider',
      }}
    >
      <Chip
        label={`v${info.current}`}
        size="small"
        variant="outlined"
        sx={{ height: 20, fontSize: '0.7rem', '& .MuiChip-label': { px: 0.75 } }}
      />
      {showUpdate && info.latest && info.latestUrl && (
        <Tooltip title={`Update available: v${info.latest}`} arrow>
          <Link
            href={info.latestUrl}
            target="_blank"
            rel="noopener noreferrer"
            underline="none"
            sx={{ display: 'flex', alignItems: 'center' }}
          >
            <Chip
              icon={<NewReleasesIcon sx={{ fontSize: '0.85rem !important' }} />}
              label={`v${info.latest}`}
              size="small"
              color="warning"
              variant="outlined"
              clickable
              sx={{ height: 20, fontSize: '0.7rem', '& .MuiChip-label': { px: 0.75 } }}
            />
          </Link>
        </Tooltip>
      )}
    </Box>
  );
}
