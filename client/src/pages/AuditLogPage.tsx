import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  AppBar, Toolbar, Typography, Box, IconButton, Card, CardContent,
  Table, TableHead, TableBody, TableRow, TableCell, TablePagination,
  Select, MenuItem, FormControl, InputLabel, TextField, Stack,
  CircularProgress, Chip, Alert,
} from '@mui/material';
import { ArrowBack as ArrowBackIcon } from '@mui/icons-material';
import { getAuditLogs, AuditLogEntry, AuditAction, AuditLogParams } from '../api/audit.api';

const ACTION_LABELS: Record<AuditAction, string> = {
  LOGIN: 'Login',
  LOGIN_OAUTH: 'OAuth Login',
  LOGIN_TOTP: 'TOTP Login',
  LOGOUT: 'Logout',
  REGISTER: 'Register',
  VAULT_UNLOCK: 'Vault Unlock',
  VAULT_LOCK: 'Vault Lock',
  VAULT_SETUP: 'Vault Setup',
  CREATE_CONNECTION: 'Create Connection',
  UPDATE_CONNECTION: 'Update Connection',
  DELETE_CONNECTION: 'Delete Connection',
  SHARE_CONNECTION: 'Share Connection',
  UNSHARE_CONNECTION: 'Unshare Connection',
  UPDATE_SHARE_PERMISSION: 'Update Share',
  CREATE_FOLDER: 'Create Folder',
  UPDATE_FOLDER: 'Update Folder',
  DELETE_FOLDER: 'Delete Folder',
  PASSWORD_CHANGE: 'Password Change',
  PROFILE_UPDATE: 'Profile Update',
  TOTP_ENABLE: '2FA Enabled',
  TOTP_DISABLE: '2FA Disabled',
  OAUTH_LINK: 'OAuth Link',
  OAUTH_UNLINK: 'OAuth Unlink',
  PASSWORD_REVEAL: 'Password Reveal',
};

function getActionColor(action: AuditAction): 'default' | 'primary' | 'secondary' | 'error' | 'warning' | 'success' | 'info' {
  if (['LOGIN', 'LOGIN_OAUTH', 'LOGIN_TOTP', 'REGISTER'].includes(action)) return 'success';
  if (['LOGOUT', 'VAULT_LOCK'].includes(action)) return 'default';
  if (['DELETE_CONNECTION', 'DELETE_FOLDER', 'UNSHARE_CONNECTION'].includes(action)) return 'error';
  if (['PASSWORD_CHANGE', 'PASSWORD_REVEAL', 'TOTP_ENABLE', 'TOTP_DISABLE'].includes(action)) return 'warning';
  if (['CREATE_CONNECTION', 'CREATE_FOLDER', 'VAULT_SETUP'].includes(action)) return 'info';
  return 'primary';
}

function formatDetails(details: Record<string, unknown> | null): string {
  if (!details) return '';
  return Object.entries(details)
    .map(([key, value]) => {
      if (Array.isArray(value)) return `${key}: ${value.join(', ')}`;
      return `${key}: ${value}`;
    })
    .join(' | ');
}

const ALL_ACTIONS: AuditAction[] = [
  'LOGIN', 'LOGIN_OAUTH', 'LOGIN_TOTP', 'LOGOUT', 'REGISTER',
  'VAULT_UNLOCK', 'VAULT_LOCK', 'VAULT_SETUP',
  'CREATE_CONNECTION', 'UPDATE_CONNECTION', 'DELETE_CONNECTION',
  'SHARE_CONNECTION', 'UNSHARE_CONNECTION', 'UPDATE_SHARE_PERMISSION',
  'CREATE_FOLDER', 'UPDATE_FOLDER', 'DELETE_FOLDER',
  'PASSWORD_CHANGE', 'PROFILE_UPDATE',
  'TOTP_ENABLE', 'TOTP_DISABLE',
  'OAUTH_LINK', 'OAUTH_UNLINK',
  'PASSWORD_REVEAL',
];

export default function AuditLogPage() {
  const navigate = useNavigate();
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [actionFilter, setActionFilter] = useState<AuditAction | ''>('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params: AuditLogParams = {
        page: page + 1,
        limit: rowsPerPage,
      };
      if (actionFilter) params.action = actionFilter;
      if (startDate) params.startDate = startDate;
      if (endDate) params.endDate = endDate;

      const result = await getAuditLogs(params);
      setLogs(result.data);
      setTotal(result.total);
    } catch {
      setError('Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, actionFilter, startDate, endDate]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleChangePage = (_: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (e: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(e.target.value, 10));
    setPage(0);
  };

  const handleFilterChange = () => {
    setPage(0);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <AppBar position="static">
        <Toolbar variant="dense">
          <IconButton edge="start" color="inherit" onClick={() => navigate(-1)} sx={{ mr: 1 }}>
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h6">Activity Log</Typography>
        </Toolbar>
      </AppBar>

      <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
        <Card sx={{ mb: 2 }}>
          <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
            <Stack direction="row" spacing={2} alignItems="center" flexWrap="wrap" useFlexGap>
              <FormControl size="small" sx={{ minWidth: 200 }}>
                <InputLabel>Action</InputLabel>
                <Select
                  value={actionFilter}
                  label="Action"
                  onChange={(e) => {
                    setActionFilter(e.target.value as AuditAction | '');
                    handleFilterChange();
                  }}
                >
                  <MenuItem value="">All Actions</MenuItem>
                  {ALL_ACTIONS.map((action) => (
                    <MenuItem key={action} value={action}>
                      {ACTION_LABELS[action]}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <TextField
                size="small"
                type="date"
                label="From"
                value={startDate}
                onChange={(e) => {
                  setStartDate(e.target.value);
                  handleFilterChange();
                }}
                slotProps={{ inputLabel: { shrink: true } }}
              />
              <TextField
                size="small"
                type="date"
                label="To"
                value={endDate}
                onChange={(e) => {
                  setEndDate(e.target.value);
                  handleFilterChange();
                }}
                slotProps={{ inputLabel: { shrink: true } }}
              />
            </Stack>
          </CardContent>
        </Card>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        <Card>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
              <CircularProgress />
            </Box>
          ) : logs.length === 0 ? (
            <Box sx={{ textAlign: 'center', py: 6 }}>
              <Typography color="text.secondary">
                {actionFilter || startDate || endDate
                  ? 'No logs match your filters'
                  : 'No activity recorded yet'}
              </Typography>
            </Box>
          ) : (
            <>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Date/Time</TableCell>
                    <TableCell>Action</TableCell>
                    <TableCell>Target</TableCell>
                    <TableCell>IP Address</TableCell>
                    <TableCell>Details</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id} hover>
                      <TableCell sx={{ whiteSpace: 'nowrap' }}>
                        {new Date(log.createdAt).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={ACTION_LABELS[log.action] || log.action}
                          color={getActionColor(log.action)}
                          size="small"
                        />
                      </TableCell>
                      <TableCell>
                        {log.targetType
                          ? `${log.targetType}${log.targetId ? ` ${log.targetId.slice(0, 8)}...` : ''}`
                          : '\u2014'}
                      </TableCell>
                      <TableCell>{log.ipAddress || '\u2014'}</TableCell>
                      <TableCell sx={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {formatDetails(log.details)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <TablePagination
                component="div"
                count={total}
                page={page}
                onPageChange={handleChangePage}
                rowsPerPage={rowsPerPage}
                onRowsPerPageChange={handleChangeRowsPerPage}
                rowsPerPageOptions={[25, 50, 100]}
              />
            </>
          )}
        </Card>
      </Box>
    </Box>
  );
}
