import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Alert, Box, Button, Card, CardContent, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Grid, IconButton, MenuItem, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import RouterIcon from '@mui/icons-material/Router'
import WifiTetheringIcon from '@mui/icons-material/WifiTethering'
import LockResetIcon from '@mui/icons-material/LockReset'
import SyncIcon from '@mui/icons-material/Sync'
import HistoryIcon from '@mui/icons-material/History'

import { createNetworkRouter, getNetworkRouterAlerts, getNetworkRouterHistory, getNetworkRouters, setNetworkRouterCredentials, syncNetworkRouterResource, testNetworkRouterConnection, updateNetworkRouter, type NetworkRouter, type NetworkRouterAlert, type NetworkRouterHealth, type NetworkRouterInput } from '../../api/networkRouters'
import { getPOPs, type POP } from '../../api/distribution'
import { getStoredUser } from '../../api/auth'
import { getAPIErrorMessage } from '../../api/errors'

const emptyForm: NetworkRouterInput = { code: '', name: '', host: '', api_port: 8729, api_username: '', use_tls: true, status: 'ACTIVE', remarks: '' }
const formatBytes = (value: number) => value > 0 ? `${(value / 1024 / 1024 / 1024).toFixed(1)} GiB` : '—'
const isResourceAlert = (alert: NetworkRouterAlert) => alert.type === 'HIGH_CPU' || alert.type === 'HIGH_MEMORY'

export default function NetworkRouters() {
  const [routers, setRouters] = useState<NetworkRouter[]>([])
  const [pops, setPOPs] = useState<POP[]>([])
  const [form, setForm] = useState<NetworkRouterInput>(emptyForm)
  const [editing, setEditing] = useState<NetworkRouter | null>(null)
  const [dialog, setDialog] = useState(false)
  const [saving, setSaving] = useState(false)
	const [testingID, setTestingID] = useState<number | null>(null)
  const [credentialRouter, setCredentialRouter] = useState<NetworkRouter | null>(null)
  const [routerPassword, setRouterPassword] = useState('')
  const [savingCredentials, setSavingCredentials] = useState(false)
  const [syncingID, setSyncingID] = useState<number | null>(null)
  const [historyRouter, setHistoryRouter] = useState<NetworkRouter | null>(null)
  const [history, setHistory] = useState<NetworkRouterHealth[]>([])
  const [loadingHistory, setLoadingHistory] = useState(false)
  const [alerts, setAlerts] = useState<NetworkRouterAlert[]>([])
  const [error, setError] = useState('')
  const isSuperadmin = getStoredUser()?.role === 'superadmin'

  const load = useCallback(async () => {
    try { setError(''); const [routerRows, popRows, alertRows] = await Promise.all([getNetworkRouters(), getPOPs(), getNetworkRouterAlerts()]); setRouters(routerRows); setPOPs(popRows); setAlerts(alertRows) }
    catch (err) { setError(getAPIErrorMessage(err, 'Failed to load network routers.')) }
  }, [])
  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load()
  }, [load])

  const openForm = (row?: NetworkRouter) => {
    setEditing(row ?? null)
    setForm(row ? { code: row.code, name: row.name, pop_id: row.pop_id, host: row.host, api_port: row.api_port, api_username: row.api_username, use_tls: row.use_tls, status: row.status, remarks: row.remarks } : { ...emptyForm, pop_id: pops.find((pop) => pop.status === 'ACTIVE')?.id })
    setDialog(true)
  }
  const save = async (event: FormEvent) => {
    event.preventDefault()
    try { setSaving(true); setError(''); if (editing) await updateNetworkRouter(editing.id, form); else await createNetworkRouter(form); setDialog(false); await load() }
    catch (err) { setError(getAPIErrorMessage(err, 'Failed to save router.')) } finally { setSaving(false) }
  }
	const testConnection = async (id: number) => {
	  try { setTestingID(id); setError(''); await testNetworkRouterConnection(id); await load() }
	  catch (err) { setError(getAPIErrorMessage(err, 'Router connectivity test failed.')) } finally { setTestingID(null) }
	}
  const saveCredentials = async (event: FormEvent) => {
    event.preventDefault()
    if (!credentialRouter) return
    try { setSavingCredentials(true); setError(''); await setNetworkRouterCredentials(credentialRouter.id, routerPassword); setCredentialRouter(null); setRouterPassword(''); await load() }
    catch (err) { setError(getAPIErrorMessage(err, 'Failed to save encrypted router credentials.')) } finally { setSavingCredentials(false) }
  }
  const syncResource = async (id: number) => {
    try { setSyncingID(id); setError(''); await syncNetworkRouterResource(id); await load() }
    catch (err) { setError(getAPIErrorMessage(err, 'Authenticated RouterOS sync failed.')); await load() } finally { setSyncingID(null) }
  }
  const openHistory = async (router: NetworkRouter) => {
    try { setHistoryRouter(router); setLoadingHistory(true); setError(''); setHistory(await getNetworkRouterHistory(router.id)) }
    catch (err) { setError(getAPIErrorMessage(err, 'Failed to load router health history.')); setHistory([]) } finally { setLoadingHistory(false) }
  }

  return <Box sx={{ '& .MuiDialogTitle-root': { color: 'text.primary', fontWeight: 700 } }}>
    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: { xs: 'stretch', sm: 'center' }, flexDirection: { xs: 'column', sm: 'row' }, gap: 2, mb: 3 }}>
      <Box><Typography variant="h4" sx={{ fontWeight: 700 }}>MikroTik Routers</Typography><Typography color="text.secondary">Manage router inventory, encrypted API credentials and connectivity health.</Typography></Box>
      {isSuperadmin && <Button variant="contained" startIcon={<AddIcon />} onClick={() => openForm()}>Add Router</Button>}
    </Box>
    {error && <Alert severity="error" onClose={() => setError('')} sx={{ mb: 2 }}>{error}</Alert>}
    <Grid container spacing={2} sx={{ mb: 3 }}>
      {([['Routers', routers.length], ['Active', routers.filter((row) => row.status === 'ACTIVE').length], ['Maintenance', routers.filter((row) => row.status === 'MAINTENANCE').length], ['Active Alerts', alerts.length]] as const).map(([label, value]) => <Grid key={label} size={{ xs: 12, sm: 6, md: 3 }}><Card><CardContent><Typography color="text.secondary">{label}</Typography><Typography variant="h5" color={label === 'Active Alerts' && value > 0 ? 'warning.main' : 'text.primary'} sx={{ fontWeight: 700 }}>{value}</Typography></CardContent></Card></Grid>)}
    </Grid>
    <Card><CardContent><TableContainer><Table sx={{ minWidth: 1100 }}><TableHead><TableRow><TableCell>Router</TableCell><TableCell>POP</TableCell><TableCell>API Endpoint</TableCell><TableCell>API User</TableCell><TableCell>RouterOS Resource</TableCell><TableCell>TLS</TableCell><TableCell>Status</TableCell><TableCell>Connectivity</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead><TableBody>
      {routers.map((row) => <TableRow key={row.id}><TableCell><Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}><RouterIcon color="primary" /><Box><b>{row.code}</b><br />{row.name}</Box></Box></TableCell><TableCell>{row.pop_name || 'Head Office'}</TableCell><TableCell>{row.host}:{row.api_port}</TableCell><TableCell>{row.api_username}<br /><Chip size="small" variant="outlined" color={row.credentials_configured ? 'success' : 'default'} label={row.credentials_configured ? 'Credential set' : 'Not set'} /></TableCell><TableCell><Chip size="small" color={row.api_status === 'AUTHENTICATED' ? 'success' : row.api_status === 'AUTH_FAILED' ? 'error' : 'default'} label={row.api_status || 'UNKNOWN'} />{row.router_identity && <Typography variant="caption" sx={{ display: 'block' }}>{row.router_identity} · RouterOS {row.routeros_version}</Typography>}{row.board_name && <Typography variant="caption" sx={{ display: 'block' }}>{row.board_name} · CPU {row.cpu_load}% · Uptime {row.router_uptime}</Typography>}{row.last_api_error && <Tooltip title={row.last_api_error}><Typography variant="caption" color="error">API sync failed</Typography></Tooltip>}</TableCell><TableCell>{row.use_tls ? 'Enabled' : 'Disabled'}</TableCell><TableCell><Chip size="small" color={row.status === 'ACTIVE' ? 'success' : row.status === 'MAINTENANCE' ? 'warning' : 'default'} label={row.status} /></TableCell><TableCell><Chip size="small" color={row.connectivity_status === 'ONLINE' ? 'success' : row.connectivity_status === 'OFFLINE' ? 'error' : 'default'} label={row.connectivity_status || 'UNKNOWN'} />{row.last_checked_at && <Typography variant="caption" sx={{ display: 'block' }}>{row.last_latency_ms} ms · {new Date(row.last_checked_at).toLocaleString()}</Typography>}{row.last_tcp_error && <Tooltip title={row.last_tcp_error}><Typography variant="caption" color="error">TCP connection failed</Typography></Tooltip>}</TableCell><TableCell align="right"><Tooltip title="Health history"><IconButton onClick={() => void openHistory(row)}><HistoryIcon /></IconButton></Tooltip>{isSuperadmin && <><Tooltip title="Set encrypted API password"><IconButton onClick={() => { setCredentialRouter(row); setRouterPassword('') }}><LockResetIcon /></IconButton></Tooltip><Tooltip title="Authenticated RouterOS sync"><span><IconButton disabled={syncingID === row.id || !row.credentials_configured} onClick={() => void syncResource(row.id)}><SyncIcon /></IconButton></span></Tooltip><Tooltip title="Test TCP connectivity"><span><IconButton disabled={testingID === row.id} onClick={() => void testConnection(row.id)}><WifiTetheringIcon /></IconButton></span></Tooltip><Tooltip title="Edit router"><IconButton onClick={() => openForm(row)}><EditIcon /></IconButton></Tooltip></>}</TableCell></TableRow>)}
      {!routers.length && <TableRow><TableCell colSpan={9} align="center">No MikroTik routers configured.</TableCell></TableRow>}
    </TableBody></Table></TableContainer></CardContent></Card>
    <Card sx={{ mt: 3 }}><CardContent><Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>Active Router Alerts</Typography><TableContainer><Table size="small"><TableHead><TableRow><TableCell>Router</TableCell><TableCell>Type</TableCell><TableCell>Current</TableCell><TableCell>Threshold</TableCell><TableCell>Opened</TableCell><TableCell>Message</TableCell></TableRow></TableHead><TableBody>{alerts.map((alert) => <TableRow key={alert.id}><TableCell>{alert.router_code} — {alert.router_name}</TableCell><TableCell><Chip size="small" color={alert.severity === 'CRITICAL' ? 'error' : 'warning'} label={alert.type.replaceAll('_', ' ')} /></TableCell><TableCell>{isResourceAlert(alert) ? `${alert.current_value.toFixed(1)}%` : '—'}</TableCell><TableCell>{isResourceAlert(alert) ? `${alert.threshold.toFixed(1)}%` : '—'}</TableCell><TableCell>{new Date(alert.opened_at).toLocaleString()}</TableCell><TableCell>{alert.message}</TableCell></TableRow>)}{alerts.length === 0 && <TableRow><TableCell colSpan={6} align="center">No active router alerts.</TableCell></TableRow>}</TableBody></Table></TableContainer></CardContent></Card>
    <Dialog open={dialog} onClose={() => !saving && setDialog(false)} fullWidth maxWidth="md"><Box component="form" onSubmit={save}><DialogTitle>{editing ? 'Edit Router' : 'Add Router'}</DialogTitle><DialogContent><Alert severity="info" sx={{ mb: 2 }}>Save router metadata here, then use the lock icon to store or replace its encrypted API password.</Alert><Grid container spacing={2}>
      <Grid size={{ xs: 12, sm: 4 }}><TextField fullWidth required label="Code" disabled={!!editing} value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} /></Grid>
      <Grid size={{ xs: 12, sm: 8 }}><TextField fullWidth required label="Router Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Grid>
      <Grid size={{ xs: 12, sm: 6 }}><TextField fullWidth select label="POP" value={form.pop_id ?? ''} onChange={(e) => setForm({ ...form, pop_id: e.target.value ? Number(e.target.value) : undefined })}><MenuItem value="">Head Office / Unassigned</MenuItem>{pops.map((pop) => <MenuItem key={pop.id} value={pop.id}>{pop.code} — {pop.name}</MenuItem>)}</TextField></Grid>
      <Grid size={{ xs: 12, sm: 6 }}><TextField fullWidth required label="Host / IP" placeholder="10.0.0.1" value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} /></Grid>
      <Grid size={{ xs: 12, sm: 4 }}><TextField fullWidth required type="number" label="API Port" value={form.api_port} slotProps={{ htmlInput: { min: 1, max: 65535 } }} onChange={(e) => setForm({ ...form, api_port: Number(e.target.value) })} /></Grid>
      <Grid size={{ xs: 12, sm: 4 }}><TextField fullWidth required label="API Username" value={form.api_username} onChange={(e) => setForm({ ...form, api_username: e.target.value })} /></Grid>
      <Grid size={{ xs: 12, sm: 4 }}><TextField fullWidth select label="TLS" value={form.use_tls ? 'true' : 'false'} onChange={(e) => setForm({ ...form, use_tls: e.target.value === 'true' })}><MenuItem value="true">Enabled</MenuItem><MenuItem value="false">Disabled</MenuItem></TextField></Grid>
      <Grid size={{ xs: 12, sm: 4 }}><TextField fullWidth select label="Status" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value as NetworkRouterInput['status'] })}>{['ACTIVE','INACTIVE','MAINTENANCE'].map((status) => <MenuItem key={status} value={status}>{status}</MenuItem>)}</TextField></Grid>
      <Grid size={{ xs: 12 }}><TextField fullWidth multiline minRows={2} label="Remarks" value={form.remarks} onChange={(e) => setForm({ ...form, remarks: e.target.value })} /></Grid>
    </Grid></DialogContent><DialogActions><Button onClick={() => setDialog(false)} disabled={saving}>Cancel</Button><Button type="submit" variant="contained" disabled={saving}>Save Router</Button></DialogActions></Box></Dialog>
    <Dialog open={credentialRouter !== null} onClose={() => !savingCredentials && setCredentialRouter(null)} fullWidth maxWidth="xs"><Box component="form" onSubmit={saveCredentials}><DialogTitle>Set Router API Password</DialogTitle><DialogContent><Alert severity="info" sx={{ mb: 2 }}>The password will be encrypted before it is stored and will never be returned by the API.</Alert><TextField fullWidth required type="password" label="API Password" autoComplete="new-password" value={routerPassword} onChange={(event) => setRouterPassword(event.target.value)} /></DialogContent><DialogActions><Button onClick={() => setCredentialRouter(null)} disabled={savingCredentials}>Cancel</Button><Button type="submit" variant="contained" disabled={savingCredentials || !routerPassword}>Save Credential</Button></DialogActions></Box></Dialog>
    <Dialog open={historyRouter !== null} onClose={() => setHistoryRouter(null)} fullWidth maxWidth="lg"><DialogTitle>{historyRouter?.name} — 30-Day Health History</DialogTitle><DialogContent><TableContainer><Table size="small" sx={{ minWidth: 850 }}><TableHead><TableRow><TableCell>Observed</TableCell><TableCell>Connectivity</TableCell><TableCell>API</TableCell><TableCell>Latency</TableCell><TableCell>CPU</TableCell><TableCell>Memory</TableCell><TableCell>Uptime</TableCell><TableCell>Error</TableCell></TableRow></TableHead><TableBody>{history.map((item) => <TableRow key={item.id}><TableCell>{new Date(item.observed_at).toLocaleString()}</TableCell><TableCell><Chip size="small" color={item.connectivity_status === 'ONLINE' ? 'success' : 'error'} label={item.connectivity_status} /></TableCell><TableCell><Chip size="small" color={item.api_status === 'AUTHENTICATED' ? 'success' : 'error'} label={item.api_status} /></TableCell><TableCell>{item.latency_ms} ms</TableCell><TableCell>{item.cpu_load}%</TableCell><TableCell>{formatBytes(item.free_memory)} / {formatBytes(item.total_memory)}</TableCell><TableCell>{item.router_uptime || '—'}</TableCell><TableCell>{item.api_error || item.tcp_error || '—'}</TableCell></TableRow>)}{!loadingHistory && history.length === 0 && <TableRow><TableCell colSpan={8} align="center">No health history recorded yet.</TableCell></TableRow>}{loadingHistory && <TableRow><TableCell colSpan={8} align="center">Loading history...</TableCell></TableRow>}</TableBody></Table></TableContainer></DialogContent><DialogActions><Button onClick={() => setHistoryRouter(null)}>Close</Button></DialogActions></Dialog>
  </Box>
}
