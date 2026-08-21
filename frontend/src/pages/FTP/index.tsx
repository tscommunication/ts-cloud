import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import axios from 'axios'
import {
  Alert, Box, Button, Card, CardContent, CircularProgress, Dialog,
  DialogActions, DialogContent, DialogTitle, Grid, IconButton, MenuItem,
  Tab, Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Tabs, TextField, Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import DeleteIcon from '@mui/icons-material/Delete'
import EditIcon from '@mui/icons-material/Edit'
import PauseCircleIcon from '@mui/icons-material/PauseCircle'
import PlayCircleIcon from '@mui/icons-material/PlayCircle'
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'

import {
  createFTPServer, createFTPUser, deleteFTPServer, deleteFTPUser,
  enableFTPUser, getFTPServers, getFTPUsers, suspendFTPUser,
  updateFTPServer, updateFTPUser, type FTPServer, type FTPServerRequest,
  type FTPUser, type FTPUserRequest,
} from '../../api/ftp'
import { getSubscriptions, type Subscription } from '../../api/subscriptions'
import { getCustomers, type Customer } from '../../api/customers'

type Resource = FTPServer | FTPUser
const serverForm = (): FTPServerRequest => ({
  name: '', driver: 'VSFTPD', host: '', port: 21, username: '', password: '',
  root_path: '/srv/ftp', passive_port_start: 40000, passive_port_end: 40100,
  max_connections: 100, status: 'ACTIVE', description: '',
})
const userForm = (): FTPUserRequest => ({
  subscription_id: 0, ftp_server_id: 0, username: '', password: '',
  home_directory: '', storage_quota_gb: 10, upload_limit_mbps: 0,
  download_limit_mbps: 0, status: 'ACTIVE', remarks: '',
})
const message = (error: unknown, fallback: string) => {
  if (axios.isAxiosError<{ message?: string; error?: string }>(error)) {
    return error.response?.data?.message || error.response?.data?.error || fallback
  }
  return fallback
}
const formatDate = (value?: string) => value ? new Date(value).toLocaleString() : '-'

function FTP() {
  const [tab, setTab] = useState(0)
  const [servers, setServers] = useState<FTPServer[]>([])
  const [users, setUsers] = useState<FTPUser[]>([])
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [serverOpen, setServerOpen] = useState(false)
  const [editingServer, setEditingServer] = useState<FTPServer | null>(null)
  const [server, setServer] = useState<FTPServerRequest>(serverForm)
  const [userOpen, setUserOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<FTPUser | null>(null)
  const [user, setUser] = useState<FTPUserRequest>(userForm)
  const [deleteItem, setDeleteItem] = useState<Resource | null>(null)

  const loadData = async () => {
    try {
      setLoading(true)
      setError('')
      const [serverData, userData, subscriptionData, customerData] = await Promise.all([
        getFTPServers(), getFTPUsers(), getSubscriptions(), getCustomers(),
      ])
      setServers(serverData)
      setUsers(userData)
      setSubscriptions(subscriptionData.subscriptions)
      setCustomers(customerData.customers)
    } catch (err: unknown) {
      setError(message(err, 'Failed to load FTP data.'))
    } finally { setLoading(false) }
  }
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadData()
  }, [])

  const serverMap = useMemo(() => new Map(servers.map((item) => [item.id, item])), [servers])
  const customerMap = useMemo(() => new Map(customers.map((item) => [item.id, item])), [customers])
  const filteredServers = useMemo(() => {
    const query = search.toLowerCase().trim()
    return query ? servers.filter((item) => [item.name, item.host, item.driver, item.status].join(' ').toLowerCase().includes(query)) : servers
  }, [servers, search])
  const filteredUsers = useMemo(() => {
    const query = search.toLowerCase().trim()
    return query ? users.filter((item) => {
      const customer = customerMap.get(item.customer_id)
      return [item.username, item.status, item.last_ip, serverMap.get(item.ftp_server_id)?.name, customer?.full_name].join(' ').toLowerCase().includes(query)
    }) : users
  }, [users, search, customerMap, serverMap])

  const openServer = (item?: FTPServer) => {
    setEditingServer(item || null)
    setServer(item ? {
      name: item.name, driver: item.driver, host: item.host, port: item.port,
      username: item.username, password: '', root_path: item.root_path,
      passive_port_start: item.passive_port_start, passive_port_end: item.passive_port_end,
      max_connections: item.max_connections, status: item.status, description: item.description,
    } : serverForm())
    setError(''); setServerOpen(true)
  }
  const openUser = (item?: FTPUser) => {
    setEditingUser(item || null)
    setUser(item ? {
      subscription_id: item.subscription_id, ftp_server_id: item.ftp_server_id,
      username: item.username, password: '', home_directory: item.home_directory,
      storage_quota_gb: item.storage_quota_gb, upload_limit_mbps: item.upload_limit_mbps,
      download_limit_mbps: item.download_limit_mbps, status: item.status, remarks: item.remarks,
    } : userForm())
    setError(''); setUserOpen(true)
  }
  const submitServer = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setError('')
    try {
      if (editingServer) await updateFTPServer(editingServer.id, server)
      else await createFTPServer(server)
      setServerOpen(false); await loadData()
    } catch (err: unknown) { setError(message(err, 'Failed to save FTP server.')) }
    finally { setBusy(false) }
  }
  const submitUser = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setError('')
    try {
      if (editingUser) await updateFTPUser(editingUser.id, user)
      else await createFTPUser(user)
      setUserOpen(false); await loadData()
    } catch (err: unknown) { setError(message(err, 'Failed to save FTP user.')) }
    finally { setBusy(false) }
  }
  const toggleUser = async (item: FTPUser) => {
    setBusy(true); setError('')
    try {
      if (item.status.toUpperCase() === 'ACTIVE') await suspendFTPUser(item.id)
      else await enableFTPUser(item.id)
      await loadData()
    } catch (err: unknown) { setError(message(err, 'Failed to change FTP user status.')) }
    finally { setBusy(false) }
  }
  const remove = async () => {
    if (!deleteItem) return
    setBusy(true); setError('')
    try {
      if ('host' in deleteItem) await deleteFTPServer(deleteItem.id)
      else await deleteFTPUser(deleteItem.id)
      setDeleteItem(null); await loadData()
    } catch (err: unknown) { setError(message(err, 'Failed to delete FTP item.')) }
    finally { setBusy(false) }
  }
  const subscriptionLabel = (item: Subscription) => {
    const customer = customerMap.get(item.customer_id)
    return `${item.subscription_code} — ${customer?.full_name || `Customer #${item.customer_id}`}`
  }
  const text = (key: keyof FTPServerRequest, label: string, type = 'text', required = false) => (
    <TextField fullWidth label={label} type={type} required={required} value={server[key]} onChange={(event) => setServer((current) => ({ ...current, [key]: type === 'number' ? Number(event.target.value) : event.target.value }))} />
  )
  const userText = (key: keyof FTPUserRequest, label: string, type = 'text', required = false) => (
    <TextField fullWidth label={label} type={type} required={required} value={user[key]} onChange={(event) => setUser((current) => ({ ...current, [key]: type === 'number' ? Number(event.target.value) : event.target.value }))} />
  )

  return <Box>
    <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, justifyContent: 'space-between', gap: 2, mb: 3 }}>
      <Box><Typography variant="h4" sx={{ fontWeight: 700 }}>FTP Management</Typography><Typography color="text.secondary">Manage FTP servers, accounts, quotas and access.</Typography></Box>
      <Button variant="contained" startIcon={<AddIcon />} onClick={() => tab === 0 ? openUser() : openServer()} disabled={loading}>{tab === 0 ? 'Add FTP User' : 'Add FTP Server'}</Button>
    </Box>
    {error && <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError('')}>{error}</Alert>}
    <Card><CardContent>
      <Tabs value={tab} onChange={(_event, value: number) => { setTab(value); setSearch('') }} sx={{ mb: 2 }}><Tab label={`Users (${users.length})`} /><Tab label={`Servers (${servers.length})`} /></Tabs>
      <Box sx={{ display: 'flex', gap: 2, justifyContent: 'space-between', mb: 2 }}><TextField size="small" placeholder={`Search FTP ${tab === 0 ? 'users' : 'servers'}...`} value={search} onChange={(event) => setSearch(event.target.value)} sx={{ maxWidth: 400, width: '100%' }} slotProps={{ input: { startAdornment: <SearchIcon sx={{ mr: 1 }} /> } }} /><IconButton onClick={() => void loadData()} disabled={loading}><RefreshIcon /></IconButton></Box>
      {loading ? <Box sx={{ py: 8, textAlign: 'center' }}><CircularProgress /></Box> : tab === 0 ? (
        filteredUsers.length === 0 ? <Typography sx={{ py: 8, textAlign: 'center' }} color="text.secondary">No FTP users found</Typography> :
        <TableContainer><Table sx={{ minWidth: 1000 }}><TableHead><TableRow><TableCell>Username</TableCell><TableCell>Customer</TableCell><TableCell>Server</TableCell><TableCell>Home</TableCell><TableCell>Quota</TableCell><TableCell>Last Login</TableCell><TableCell>Status</TableCell><TableCell align="right">Actions</TableCell></TableRow></TableHead><TableBody>{filteredUsers.map((item) => {
          const customer = customerMap.get(item.customer_id)
          return <TableRow key={item.id} hover><TableCell><Typography sx={{ fontWeight: 600 }}>{item.username}</Typography></TableCell><TableCell>{customer?.full_name || `Subscription #${item.subscription_id}`}</TableCell><TableCell>{serverMap.get(item.ftp_server_id)?.name || `#${item.ftp_server_id}`}</TableCell><TableCell>{item.home_directory}</TableCell><TableCell>{item.storage_quota_gb} GB</TableCell><TableCell>{formatDate(item.last_login)}</TableCell><TableCell sx={{ color: item.status.toUpperCase() === 'ACTIVE' ? 'success.main' : 'warning.main', fontWeight: 600 }}>{item.status}</TableCell><TableCell align="right" sx={{ whiteSpace: 'nowrap' }}><IconButton title={item.status.toUpperCase() === 'ACTIVE' ? 'Suspend' : 'Enable'} onClick={() => void toggleUser(item)} disabled={busy}>{item.status.toUpperCase() === 'ACTIVE' ? <PauseCircleIcon /> : <PlayCircleIcon />}</IconButton><IconButton color="primary" onClick={() => openUser(item)}><EditIcon /></IconButton><IconButton color="error" onClick={() => setDeleteItem(item)}><DeleteIcon /></IconButton></TableCell></TableRow>
        })}</TableBody></Table></TableContainer>
      ) : filteredServers.length === 0 ? <Typography sx={{ py: 8, textAlign: 'center' }} color="text.secondary">No FTP servers found</Typography> :
        <TableContainer><Table sx={{ minWidth: 900 }}><TableHead><TableRow><TableCell>Name</TableCell><TableCell>Driver</TableCell><TableCell>Host</TableCell><TableCell>Root Path</TableCell><TableCell>Passive Ports</TableCell><TableCell>Max Connections</TableCell><TableCell>Status</TableCell><TableCell align="right">Actions</TableCell></TableRow></TableHead><TableBody>{filteredServers.map((item) => <TableRow key={item.id} hover><TableCell sx={{ fontWeight: 600 }}>{item.name}</TableCell><TableCell>{item.driver}</TableCell><TableCell>{item.host}:{item.port}</TableCell><TableCell>{item.root_path}</TableCell><TableCell>{item.passive_port_start}–{item.passive_port_end}</TableCell><TableCell>{item.max_connections}</TableCell><TableCell sx={{ color: item.status === 'ACTIVE' ? 'success.main' : 'warning.main', fontWeight: 600 }}>{item.status}</TableCell><TableCell align="right"><IconButton color="primary" onClick={() => openServer(item)}><EditIcon /></IconButton><IconButton color="error" onClick={() => setDeleteItem(item)}><DeleteIcon /></IconButton></TableCell></TableRow>)}</TableBody></Table></TableContainer>}
    </CardContent></Card>

    <Dialog open={serverOpen} onClose={() => !busy && setServerOpen(false)} fullWidth maxWidth="md"><Box component="form" onSubmit={submitServer}><DialogTitle>{editingServer ? 'Edit FTP Server' : 'Add FTP Server'}</DialogTitle><DialogContent dividers><Grid container spacing={2} sx={{ pt: 1 }}>
      <Grid size={{ xs: 12, md: 6 }}>{text('name', 'Name', 'text', true)}</Grid><Grid size={{ xs: 12, md: 6 }}>{text('driver', 'Driver')}</Grid><Grid size={{ xs: 12, md: 8 }}>{text('host', 'Host', 'text', true)}</Grid><Grid size={{ xs: 12, md: 4 }}>{text('port', 'Port', 'number')}</Grid><Grid size={{ xs: 12, md: 6 }}>{text('username', 'Admin Username', 'text', true)}</Grid><Grid size={{ xs: 12, md: 6 }}>{text('password', editingServer ? 'New Password (required)' : 'Password', 'password', true)}</Grid><Grid size={{ xs: 12 }}>{text('root_path', 'Root Path', 'text', true)}</Grid><Grid size={{ xs: 12, md: 4 }}>{text('passive_port_start', 'Passive Port Start', 'number')}</Grid><Grid size={{ xs: 12, md: 4 }}>{text('passive_port_end', 'Passive Port End', 'number')}</Grid><Grid size={{ xs: 12, md: 4 }}>{text('max_connections', 'Max Connections', 'number')}</Grid><Grid size={{ xs: 12, md: 6 }}><TextField fullWidth select label="Status" value={server.status} onChange={(event) => setServer((current) => ({ ...current, status: event.target.value }))}><MenuItem value="ACTIVE">ACTIVE</MenuItem><MenuItem value="DISABLED">DISABLED</MenuItem></TextField></Grid><Grid size={{ xs: 12 }}>{text('description', 'Description')}</Grid>
    </Grid></DialogContent><DialogActions><Button onClick={() => setServerOpen(false)} disabled={busy}>Cancel</Button><Button type="submit" variant="contained" disabled={busy}>{busy ? 'Saving...' : 'Save Server'}</Button></DialogActions></Box></Dialog>

    <Dialog open={userOpen} onClose={() => !busy && setUserOpen(false)} fullWidth maxWidth="md"><Box component="form" onSubmit={submitUser}><DialogTitle>{editingUser ? 'Edit FTP User' : 'Add FTP User'}</DialogTitle><DialogContent dividers><Grid container spacing={2} sx={{ pt: 1 }}>
      <Grid size={{ xs: 12, md: 6 }}><TextField fullWidth required select label="Subscription" value={user.subscription_id || ''} onChange={(event) => setUser((current) => ({ ...current, subscription_id: Number(event.target.value) }))}>{subscriptions.map((item) => <MenuItem key={item.id} value={item.id}>{subscriptionLabel(item)}</MenuItem>)}</TextField></Grid><Grid size={{ xs: 12, md: 6 }}><TextField fullWidth required select label="FTP Server" value={user.ftp_server_id || ''} onChange={(event) => setUser((current) => ({ ...current, ftp_server_id: Number(event.target.value) }))}>{servers.map((item) => <MenuItem key={item.id} value={item.id}>{item.name} ({item.host})</MenuItem>)}</TextField></Grid><Grid size={{ xs: 12, md: 6 }}>{userText('username', 'Username', 'text', true)}</Grid><Grid size={{ xs: 12, md: 6 }}>{userText('password', editingUser ? 'New Password (required)' : 'Password', 'password', true)}</Grid><Grid size={{ xs: 12 }}>{userText('home_directory', 'Home Directory', 'text', true)}</Grid><Grid size={{ xs: 12, md: 4 }}>{userText('storage_quota_gb', 'Storage Quota (GB)', 'number')}</Grid><Grid size={{ xs: 12, md: 4 }}>{userText('upload_limit_mbps', 'Upload Limit (Mbps)', 'number')}</Grid><Grid size={{ xs: 12, md: 4 }}>{userText('download_limit_mbps', 'Download Limit (Mbps)', 'number')}</Grid><Grid size={{ xs: 12, md: 6 }}><TextField fullWidth select label="Status" value={user.status} onChange={(event) => setUser((current) => ({ ...current, status: event.target.value }))}><MenuItem value="ACTIVE">ACTIVE</MenuItem><MenuItem value="SUSPENDED">SUSPENDED</MenuItem><MenuItem value="DISABLED">DISABLED</MenuItem></TextField></Grid><Grid size={{ xs: 12 }}>{userText('remarks', 'Remarks')}</Grid>
    </Grid></DialogContent><DialogActions><Button onClick={() => setUserOpen(false)} disabled={busy}>Cancel</Button><Button type="submit" variant="contained" disabled={busy || !user.subscription_id || !user.ftp_server_id}>{busy ? 'Saving...' : 'Save User'}</Button></DialogActions></Box></Dialog>

    <Dialog open={Boolean(deleteItem)} onClose={() => !busy && setDeleteItem(null)} fullWidth maxWidth="xs"><DialogTitle>Delete FTP {deleteItem && 'host' in deleteItem ? 'Server' : 'User'}</DialogTitle><DialogContent><Typography>Delete <strong>{deleteItem && ('host' in deleteItem ? deleteItem.name : deleteItem.username)}</strong>? This action cannot be undone.</Typography></DialogContent><DialogActions><Button onClick={() => setDeleteItem(null)} disabled={busy}>Cancel</Button><Button color="error" variant="contained" onClick={() => void remove()} disabled={busy}>{busy ? 'Deleting...' : 'Delete'}</Button></DialogActions></Dialog>
  </Box>
}

export default FTP
