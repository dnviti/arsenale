import { useEffect } from 'react';
import {
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, IconButton, Tooltip, Typography, Paper,
} from '@mui/material';
import { RestartAlt as RestartIcon } from '@mui/icons-material';
import { useGatewayStore } from '../../store/gatewayStore';

type InstanceStatus = 'PROVISIONING' | 'RUNNING' | 'STOPPED' | 'ERROR' | 'REMOVING';

const statusColor: Record<InstanceStatus, 'warning' | 'success' | 'default' | 'error'> = {
  PROVISIONING: 'warning',
  RUNNING: 'success',
  STOPPED: 'default',
  ERROR: 'error',
  REMOVING: 'warning',
};

interface GatewayInstanceListProps {
  gatewayId: string;
}

export default function GatewayInstanceList({ gatewayId }: GatewayInstanceListProps) {
  const instances = useGatewayStore((s) => s.instances[gatewayId] ?? []);
  const fetchInstances = useGatewayStore((s) => s.fetchInstances);
  const restartInstance = useGatewayStore((s) => s.restartInstance);

  useEffect(() => {
    fetchInstances(gatewayId);
  }, [gatewayId, fetchInstances]);

  if (instances.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
        No instances deployed
      </Typography>
    );
  }

  return (
    <TableContainer component={Paper} variant="outlined" sx={{ mt: 1 }}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Container ID</TableCell>
            <TableCell>Name</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Health</TableCell>
            <TableCell>Host:Port</TableCell>
            <TableCell>Created</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {instances.map((inst) => (
            <TableRow key={inst.id}>
              <TableCell>
                <Tooltip title={inst.containerId}>
                  <Typography variant="body2" fontFamily="monospace">
                    {inst.containerId.slice(0, 12)}
                  </Typography>
                </Tooltip>
              </TableCell>
              <TableCell>{inst.containerName}</TableCell>
              <TableCell>
                <Chip
                  label={inst.status}
                  size="small"
                  color={statusColor[inst.status as InstanceStatus] ?? 'default'}
                />
              </TableCell>
              <TableCell>
                <Tooltip title={inst.errorMessage || ''}>
                  <Typography variant="body2" color={inst.healthStatus === 'healthy' ? 'success.main' : 'text.secondary'}>
                    {inst.healthStatus || 'N/A'}
                  </Typography>
                </Tooltip>
              </TableCell>
              <TableCell>
                <Typography variant="body2" fontFamily="monospace">
                  {inst.host}:{inst.port}
                </Typography>
              </TableCell>
              <TableCell>
                <Typography variant="caption">
                  {new Date(inst.createdAt).toLocaleString()}
                </Typography>
              </TableCell>
              <TableCell align="right">
                <Tooltip title="Restart instance">
                  <span>
                    <IconButton
                      size="small"
                      onClick={() => restartInstance(gatewayId, inst.id)}
                      disabled={inst.status !== 'RUNNING' && inst.status !== 'ERROR'}
                    >
                      <RestartIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
