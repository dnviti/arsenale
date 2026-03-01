import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  AppBar, Toolbar, Typography, IconButton, Box, Button, Alert,
  CircularProgress, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Chip, Dialog, DialogTitle, DialogContent,
  DialogContentText, DialogActions,
} from '@mui/material';
import {
  ArrowBack, Add as AddIcon, Delete as DeleteIcon,
  Edit as EditIcon, PlayArrow as TestIcon,
  Router as RouterIcon,
} from '@mui/icons-material';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { testGateway } from '../../api/gateway.api';
import type { GatewayData } from '../../api/gateway.api';
import GatewayDialog from './GatewayDialog';

interface TestState {
  gatewayId: string;
  loading: boolean;
  result?: { reachable: boolean; latencyMs: number | null; error: string | null };
}

export default function GatewayList() {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const gateways = useGatewayStore((s) => s.gateways);
  const loading = useGatewayStore((s) => s.loading);
  const fetchGateways = useGatewayStore((s) => s.fetchGateways);
  const deleteGatewayAction = useGatewayStore((s) => s.deleteGateway);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingGateway, setEditingGateway] = useState<GatewayData | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<GatewayData | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState('');
  const [testStates, setTestStates] = useState<Record<string, TestState>>({});

  const hasTenant = Boolean(user?.tenantId);

  useEffect(() => {
    if (hasTenant) fetchGateways();
  }, [fetchGateways, hasTenant]);

  const handleEdit = (gw: GatewayData) => {
    setEditingGateway(gw);
    setDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setError('');
    try {
      await deleteGatewayAction(deleteTarget.id);
    } catch (err: unknown) {
      setError(
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        'Failed to delete gateway'
      );
    } finally {
      setDeleting(false);
      setDeleteTarget(null);
    }
  };

  const handleTest = async (gw: GatewayData) => {
    setTestStates((prev) => ({
      ...prev,
      [gw.id]: { gatewayId: gw.id, loading: true },
    }));
    try {
      const result = await testGateway(gw.id);
      setTestStates((prev) => ({
        ...prev,
        [gw.id]: { gatewayId: gw.id, loading: false, result },
      }));
    } catch {
      setTestStates((prev) => ({
        ...prev,
        [gw.id]: {
          gatewayId: gw.id,
          loading: false,
          result: { reachable: false, latencyMs: null, error: 'Test request failed' },
        },
      }));
    }
  };

  if (!hasTenant) {
    return (
      <Box>
        <AppBar position="static">
          <Toolbar variant="dense">
            <IconButton edge="start" color="inherit" onClick={() => navigate('/settings')}>
              <ArrowBack />
            </IconButton>
            <Typography variant="h6" sx={{ ml: 1 }}>Gateways</Typography>
          </Toolbar>
        </AppBar>
        <Box sx={{ maxWidth: 500, mx: 'auto', mt: 6, px: 2, textAlign: 'center' }}>
          <Typography variant="h6" gutterBottom>No Organization</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            You need to create or join an organization before managing gateways.
          </Typography>
          <Button variant="contained" onClick={() => navigate('/settings/tenant')}>
            Set Up Organization
          </Button>
        </Box>
      </Box>
    );
  }

  return (
    <Box>
      <AppBar position="static">
        <Toolbar variant="dense">
          <IconButton edge="start" color="inherit" onClick={() => navigate('/settings')}>
            <ArrowBack />
          </IconButton>
          <Typography variant="h6" sx={{ ml: 1, flexGrow: 1 }}>Gateways</Typography>
          <Button
            color="inherit"
            startIcon={<AddIcon />}
            onClick={() => { setEditingGateway(null); setDialogOpen(true); }}
          >
            New Gateway
          </Button>
        </Toolbar>
      </AppBar>

      <Box sx={{ maxWidth: 900, mx: 'auto', mt: 3, px: 2, pb: 4 }}>
        {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>{error}</Alert>}

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
            <CircularProgress />
          </Box>
        ) : gateways.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 6 }}>
            <RouterIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
            <Typography variant="h6" gutterBottom>No Gateways Yet</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
              Add a gateway to route connections through GUACD or SSH bastion hosts.
            </Typography>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => { setEditingGateway(null); setDialogOpen(true); }}
            >
              Add Gateway
            </Button>
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Host</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {gateways.map((gw) => {
                  const test = testStates[gw.id];
                  return (
                    <TableRow key={gw.id}>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="body2">{gw.name}</Typography>
                          {gw.isDefault && (
                            <Chip label="Default" size="small" color="primary" variant="outlined" />
                          )}
                        </Box>
                        {gw.description && (
                          <Typography variant="caption" color="text.secondary">
                            {gw.description}
                          </Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={gw.type === 'GUACD' ? 'GUACD' : 'SSH Bastion'}
                          size="small"
                          color={gw.type === 'GUACD' ? 'info' : 'warning'}
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">{gw.host}:{gw.port}</Typography>
                      </TableCell>
                      <TableCell>
                        {test?.loading ? (
                          <CircularProgress size={16} />
                        ) : test?.result ? (
                          test.result.reachable ? (
                            <Chip
                              label={`Reachable${test.result.latencyMs != null ? ` (${test.result.latencyMs}ms)` : ''}`}
                              size="small"
                              color="success"
                            />
                          ) : (
                            <Chip
                              label={test.result.error || 'Unreachable'}
                              size="small"
                              color="error"
                            />
                          )
                        ) : (
                          <Typography variant="caption" color="text.secondary">Not tested</Typography>
                        )}
                      </TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={() => handleTest(gw)} title="Test connectivity">
                          <TestIcon fontSize="small" />
                        </IconButton>
                        <IconButton size="small" onClick={() => handleEdit(gw)} title="Edit">
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton size="small" color="error" onClick={() => setDeleteTarget(gw)} title="Delete">
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>

      <GatewayDialog
        open={dialogOpen}
        onClose={() => { setDialogOpen(false); setEditingGateway(null); }}
        gateway={editingGateway}
      />

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Gateway</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete <strong>{deleteTarget?.name}</strong>?
            Connections using this gateway will revert to direct connection.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button onClick={handleDelete} color="error" variant="contained" disabled={deleting}>
            {deleting ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
