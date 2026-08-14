import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Alert, Box, Button, Card, CardContent, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Grid, IconButton, MenuItem, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import RouterIcon from '@mui/icons-material/Router'
import WifiTetheringIcon from '@mui/icons-material/WifiTethering'

import { createNetworkRouter, getNetworkRouters, testNetworkRouterConnection, updateNetworkRouter, type NetworkRouter, type NetworkRouterInput } from '../../api/networkRouters'
import { getPOPs, type POP } from '../../api/distribution'
import { getStoredUser } from '../../api/auth'
import { getAPIErrorMessage } from '../../api/errors'

const emptyForm: NetworkRouterInput = { code: '', name: '', host: '', api_port: 8729, api_username: '', use_tls: true, status: 'ACTIVE', remarks: '' }

export default function NetworkRouters() {
  const [routers, setRouters] = useState<NetworkRouter[]>([])
  const [pops, setPOPs] = useState<POP[]>([])
  const [form, setForm] = useState<NetworkRouterInput>(emptyForm)
  const [editing, setEditing] = useState<NetworkRouter | null>(null)
  const [dialog, setDialog] = useState(false)
  const [saving, setSaving] = useState(false)
	const [testingID, setTestingID] = useState<number | null>(null)
  const [error, setError] = useState('')
  const isSuperadmin = getStoredUser()?.role === 'superadmin'

  const load = useCallback(async () => {
    try { setError(''); const [routerRows, popRows] = await Promise.all([getNetworkRouters(), getPOPs()]); setRouters(routerRows); setPOPs(popRows) }
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

  return <Box>
    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: { xs: 'stretch', sm: 'center' }, flexDirection: { xs: 'column', sm: 'row' }, gap: 2, mb: 3 }}>
      <Box><Typography variant="h4" sx={{ fontWeight: 700 }}>MikroTik Routers</Typography><Typography color="text.secondary">Manage router inventory and API connection metadata. Credentials are not stored yet.</Typography></Box>
      {isSuperadmin && <Button variant="contained" startIcon={<AddIcon />} onClick={() => openForm()}>Add Router</Button>}
    </Box>
    {error && <Alert severity="error" onClose={() => setError('')} sx={{ mb: 2 }}>{error}</Alert>}
    <Grid container spacing={2} sx={{ mb: 3 }}>
      {([['Routers', routers.length], ['Active', routers.filter((row) => row.status === 'ACTIVE').length], ['Maintenance', routers.filter((row) => row.status === 'MAINTENANCE').length]] as const).map(([label, value]) => <Grid key={label} size={{ xs: 12, sm: 4 }}><Card><CardContent><Typography color="text.secondary">{label}</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>{value}</Typography></CardContent></Card></Grid>)}
    </Grid>
    <Card><CardContent><TableContainer><Table sx={{ minWidth: 900 }}><TableHead><TableRow><TableCell>Router</TableCell><TableCell>POP</TableCell><TableCell>API Endpoint</TableCell><TableCell>API User</TableCell><TableCell>TLS</TableCell><TableCell>Status</TableCell><TableCell>Connectivity</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead><TableBody>
      {routers.map((row) => <TableRow key={row.id}><TableCell><Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}><RouterIcon color="primary" /><Box><b>{row.code}</b><br />{row.name}</Box></Box></TableCell><TableCell>{row.pop_name || 'Head Office'}</TableCell><TableCell>{row.host}:{row.api_port}</TableCell><TableCell>{row.api_username}</TableCell><TableCell>{row.use_tls ? 'Enabled' : 'Disabled'}</TableCell><TableCell><Chip size="small" color={row.status === 'ACTIVE' ? 'success' : row.status === 'MAINTENANCE' ? 'warning' : 'default'} label={row.status} /></TableCell><TableCell><Chip size="small" color={row.connectivity_status === 'ONLINE' ? 'success' : row.connectivity_status === 'OFFLINE' ? 'error' : 'default'} label={row.connectivity_status || 'UNKNOWN'} />{row.last_checked_at && <Typography variant="caption" sx={{ display: 'block' }}>{row.last_latency_ms} ms · {new Date(row.last_checked_at).toLocaleString()}</Typography>}{row.last_connection_error && <Tooltip title={row.last_connection_error}><Typography variant="caption" color="error">Connection failed</Typography></Tooltip>}</TableCell><TableCell align="right">{isSuperadmin && <><Tooltip title="Test TCP connectivity"><span><IconButton disabled={testingID === row.id} onClick={() => void testConnection(row.id)}><WifiTetheringIcon /></IconButton></span></Tooltip><Tooltip title="Edit router"><IconButton onClick={() => openForm(row)}><EditIcon /></IconButton></Tooltip></>}</TableCell></TableRow>)}
      {!routers.length && <TableRow><TableCell colSpan={8} align="center">No MikroTik routers configured.</TableCell></TableRow>}
    </TableBody></Table></TableContainer></CardContent></Card>
    <Dialog open={dialog} onClose={() => !saving && setDialog(false)} fullWidth maxWidth="md"><Box component="form" onSubmit={save}><DialogTitle>{editing ? 'Edit Router' : 'Add Router'}</DialogTitle><DialogContent><Alert severity="info" sx={{ mb: 2 }}>Router passwords are intentionally not accepted in this phase.</Alert><Grid container spacing={2}>
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
  </Box>
}
