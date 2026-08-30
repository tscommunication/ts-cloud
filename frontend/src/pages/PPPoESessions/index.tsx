import { useEffect, useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Alert, Autocomplete, Box, Button, Card, CardContent, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, Grid, MenuItem, Switch, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Typography } from '@mui/material'
import PeopleIcon from '@mui/icons-material/People'
import VerifiedUserIcon from '@mui/icons-material/VerifiedUser'
import LinkOffIcon from '@mui/icons-material/LinkOff'
import { Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip as ChartTooltip, XAxis, YAxis } from 'recharts'

import { getNetworkPPPoEDailyUsageSummary, getNetworkPPPoESessionLiveTraffic, getNetworkPPPoESessions, getNetworkPPPoEUserUsage, getNetworkRouterPPPSecrets, mapNetworkPPPoESession, mapNetworkRouterPPPSecret, type NetworkRouterPPPoESession, type NetworkRouterPPPSecret } from '../../api/networkRouters'
import { getSubscriptions } from '../../api/subscriptions'
import { getCustomer } from '../../api/customers'
import { getAPIErrorMessage } from '../../api/errors'
import { getStoredUser } from '../../api/auth'

const formatRate = (bps: number) => bps > 0 ? `${(bps / 1_000_000).toFixed(bps >= 10_000_000 ? 0 : 2)} Mbps` : '0 Mbps'
const formatTraffic = (bytes: number) => bytes > 0 ? `${(bytes / 1024 / 1024).toFixed(bytes >= 1024 * 1024 * 1024 ? 0 : 1)} ${bytes >= 1024 * 1024 * 1024 ? 'GiB' : 'MiB'}` : '0 B'

export default function PPPoESessions() {
  const queryClient = useQueryClient()
	const isAgent = getStoredUser()?.role === 'agent'
	const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const [search, setSearch] = useState('')
	const [showHistory, setShowHistory] = useState(false)
	const [usageDays, setUsageDays] = useState(7)
  const [mapSession, setMapSession] = useState<NetworkRouterPPPoESession | null>(null)
	const [detailSession, setDetailSession] = useState<NetworkRouterPPPoESession | null>(null)
	const [trafficSession, setTrafficSession] = useState<NetworkRouterPPPoESession | null>(null)
	const [trafficSamples, setTrafficSamples] = useState<{ time: string; rx: number; tx: number }[]>([])
  const [mapSecret, setMapSecret] = useState<NetworkRouterPPPSecret | null>(null)
  const [subscriptionID, setSubscriptionID] = useState<number | null>(null)
  const [mappingBusy, setMappingBusy] = useState(false)
  const [actionError, setActionError] = useState('')
  const mapping = params.get('mapping') ?? 'all'
  const sessions = useQuery({ queryKey: ['network-pppoe-sessions', showHistory ? 'all' : 'active'], queryFn: () => getNetworkPPPoESessions(!showHistory), refetchInterval: 30000 })
	const usage = useQuery({ queryKey: ['network-pppoe-usage', usageDays], queryFn: () => getNetworkPPPoEDailyUsageSummary(usageDays), refetchInterval: 30000 })
  const userUsage = useQuery({ queryKey: ['network-pppoe-user-usage', usageDays], queryFn: () => getNetworkPPPoEUserUsage(usageDays), refetchInterval: 30000 })
	const detailCustomer = useQuery({ queryKey: ['pppoe-customer-detail', detailSession?.customer_id], queryFn: () => getCustomer(detailSession!.customer_id!), enabled: Boolean(detailSession?.customer_id) })
	const currentTrafficSession = useMemo(() => (sessions.data ?? []).find((item) => item.id === trafficSession?.id) ?? trafficSession, [sessions.data, trafficSession])
	const liveTraffic = useQuery({ queryKey: ['network-pppoe-live-traffic', trafficSession?.id], queryFn: () => getNetworkPPPoESessionLiveTraffic(trafficSession!.id), enabled: Boolean(trafficSession?.active), refetchInterval: 5000 })
	useEffect(() => {
		const current = currentTrafficSession
		if (!current) return
		const timer = window.setTimeout(() => setTrafficSamples((previous) => [...previous.slice(-19), { time: new Date().toLocaleTimeString(), rx: (liveTraffic.data?.download_bps ?? current.rx_rate_bps) / 1_000_000, tx: (liveTraffic.data?.upload_bps ?? current.tx_rate_bps) / 1_000_000 }]), 0)
		return () => window.clearTimeout(timer)
	}, [currentTrafficSession, liveTraffic.data])
	const secrets = useQuery({ queryKey: ['network-ppp-secrets'], queryFn: getNetworkRouterPPPSecrets, enabled: !isAgent, refetchInterval: 30000 })
	const subscriptions = useQuery({ queryKey: ['subscriptions', 'pppoe-mapping'], queryFn: () => getSubscriptions(), enabled: !isAgent })
  const rows = useMemo(() => {
    const query = search.trim().toLowerCase()
    return (sessions.data ?? []).filter((item) => {
      if (mapping === 'mapped' && !item.subscription_id) return false
      if (mapping === 'unmapped' && item.subscription_id) return false
      if (!query) return true
      return [item.username, item.address, item.caller_id, item.router_code, item.router_name, item.customer_code, item.customer_name, item.package_code, item.package_name].join(' ').toLowerCase().includes(query)
    })
  }, [mapping, search, sessions.data])
  const mapped = (sessions.data ?? []).filter((item) => item.subscription_id).length
  const unmapped = (sessions.data?.length ?? 0) - mapped
	const activeUsers = (sessions.data ?? []).filter((item) => item.active).length
  const secretRows = useMemo(() => {
    const query = search.trim().toLowerCase()
    return (secrets.data ?? []).filter((item) => !query || [item.username, item.profile, item.router_code, item.router_name, item.customer_code, item.customer_name].join(' ').toLowerCase().includes(query))
  }, [search, secrets.data])
  const unmappedSecrets = (secrets.data ?? []).filter((item) => !item.subscription_id).length
  const exportUsageCSV = () => {
    const escape = (value: string | number) => `"${String(value).replaceAll('"', '""')}"`
    const lines = [['Username', 'Router', 'RX Bytes', 'TX Bytes', 'Total Bytes'], ...(userUsage.data ?? []).map((item) => [item.username, item.router_code, item.rx_bytes, item.tx_bytes, item.rx_bytes + item.tx_bytes])]
    const blob = new Blob([`\uFEFF${lines.map((line) => line.map(escape).join(',')).join('\n')}\n`], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url; link.download = `pppoe-traffic-${usageDays}-days.csv`; link.click()
    URL.revokeObjectURL(url)
  }
  const mapSelected = async () => {
    if (!mapSession || !subscriptionID) return
    try {
      setMappingBusy(true); setActionError('')
      await mapNetworkPPPoESession(mapSession.id, subscriptionID)
      await Promise.all([queryClient.invalidateQueries({ queryKey: ['network-pppoe-sessions'] }), queryClient.invalidateQueries({ queryKey: ['network-pppoe-summary'] })])
      setMapSession(null); setSubscriptionID(null)
    } catch (error) { setActionError(getAPIErrorMessage(error, 'Failed to map PPPoE user.')) } finally { setMappingBusy(false) }
  }
  const mapSelectedSecret = async () => {
    if (!mapSecret || !subscriptionID) return
    try {
      setMappingBusy(true); setActionError('')
      await mapNetworkRouterPPPSecret(mapSecret.id, subscriptionID)
      await queryClient.invalidateQueries({ queryKey: ['network-ppp-secrets'] })
      setMapSecret(null); setSubscriptionID(null)
    } catch (error) { setActionError(getAPIErrorMessage(error, 'Failed to map PPP secret.')) } finally { setMappingBusy(false) }
  }

  return <Box>
    <Typography variant="h4" sx={{ fontWeight: 700 }}>Live PPPoE Users</Typography>
    <Typography color="text.secondary" sx={{ mb: 3 }}>{isAgent ? 'Only customers assigned to your agent account are shown. Refreshes every 30 seconds.' : 'Active sessions synchronized from every MikroTik router. Refreshes every 30 seconds.'}</Typography>
    {sessions.isError && <Alert severity="error" sx={{ mb: 2 }}>Failed to load live PPPoE users.</Alert>}
    {actionError && <Alert severity="error" onClose={() => setActionError('')} sx={{ mb: 2 }}>{actionError}</Alert>}
    <Grid container spacing={2} sx={{ mb: 3 }}>
      {[{ label: 'Active Users', value: activeUsers, icon: <PeopleIcon />, color: 'primary.main' }, { label: 'Mapped Users', value: mapped, icon: <VerifiedUserIcon />, color: 'success.main' }, { label: 'Unmapped Users', value: unmapped, icon: <LinkOffIcon />, color: unmapped > 0 ? 'warning.main' : 'text.secondary' }, { label: `${usageDays}-Day RX / TX`, value: `${formatTraffic(usage.data?.rx_bytes ?? 0)} / ${formatTraffic(usage.data?.tx_bytes ?? 0)}`, icon: <PeopleIcon />, color: 'info.main' }].map((stat) => <Grid key={stat.label} size={{ xs: 12, sm: 6, md: 3 }}><Card><CardContent><Box sx={{ display: 'flex', justifyContent: 'space-between' }}><Box><Typography color="text.secondary">{stat.label}</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>{stat.value}</Typography></Box><Box sx={{ color: stat.color }}>{stat.icon}</Box></Box></CardContent></Card></Grid>)}
    </Grid>
    <Card><CardContent>
      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 8 }}><TextField fullWidth label="Search username, customer, package, IP or router" value={search} onChange={(event) => setSearch(event.target.value)} /></Grid>
        <Grid size={{ xs: 12, md: 4 }}><TextField fullWidth select label="Mapping" value={mapping} onChange={(event) => setParams(event.target.value === 'all' ? {} : { mapping: event.target.value })}><MenuItem value="all">All users</MenuItem><MenuItem value="mapped">Mapped only</MenuItem><MenuItem value="unmapped">Unmapped only</MenuItem></TextField></Grid>
        <Grid size={{ xs: 12 }}><FormControlLabel control={<Switch checked={showHistory} onChange={(event) => setShowHistory(event.target.checked)} />} label="Include disconnected session history" /></Grid>
      </Grid>
      <TableContainer><Table sx={{ minWidth: 1710 }} size="small"><TableHead><TableRow><TableCell>#</TableCell><TableCell>Username</TableCell><TableCell>Router</TableCell><TableCell>Customer</TableCell>{!isAgent && <TableCell>Agent</TableCell>}<TableCell>Package</TableCell><TableCell>Mapping</TableCell><TableCell>Status</TableCell><TableCell>IP Address</TableCell><TableCell>Caller ID / MAC</TableCell><TableCell>Service</TableCell><TableCell>Live RX / TX</TableCell><TableCell>Traffic RX / TX</TableCell><TableCell>Uptime</TableCell><TableCell>Last Seen / Disconnected</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead><TableBody>
		{rows.map((item, index) => <TableRow key={item.id} hover><TableCell>{index + 1}</TableCell><TableCell><Button size="small" sx={{ fontWeight: 700, textTransform: 'none' }} onClick={() => item.customer_id ? navigate(`/customers/${item.customer_id}`) : setDetailSession(item)}>{item.username}</Button></TableCell><TableCell>{item.router_code}<br />{item.router_name}</TableCell><TableCell>{item.customer_name ? <>{item.customer_code}<br />{item.customer_name}</> : '—'}</TableCell>{!isAgent && <TableCell>{item.agent_name ? <>{item.agent_code}<br />{item.agent_name}</> : 'Unassigned'}</TableCell>}<TableCell>{item.package_name ? <>{item.package_code}<br />{item.package_name}</> : '—'}</TableCell><TableCell><Chip size="small" color={item.subscription_id ? 'success' : 'warning'} label={item.subscription_id ? 'MAPPED' : 'UNMAPPED'} /></TableCell><TableCell><Chip size="small" color={item.active ? 'success' : 'default'} label={item.active ? 'ACTIVE' : 'DISCONNECTED'} />{!item.active && <Typography variant="caption" sx={{ display: 'block' }}>{item.disconnect_reason || 'UNKNOWN'}</Typography>}</TableCell><TableCell>{item.address || '—'}</TableCell><TableCell>{item.caller_id || '—'}</TableCell><TableCell>{item.service || '—'}</TableCell><TableCell>↓ {formatRate(item.rx_rate_bps)}<br />↑ {formatRate(item.tx_rate_bps)}</TableCell><TableCell>↓ {formatTraffic(item.rx_bytes)}<br />↑ {formatTraffic(item.tx_bytes)}</TableCell><TableCell>{item.uptime || '—'}</TableCell><TableCell>{new Date(item.last_seen_at).toLocaleString()}{item.disconnected_at && <><br /><Typography variant="caption">Disconnected: {new Date(item.disconnected_at).toLocaleString()}</Typography></>}</TableCell><TableCell align="right">{item.active && <Button size="small" onClick={() => { setTrafficSession(item); setTrafficSamples([{ time: new Date().toLocaleTimeString(), rx: item.rx_rate_bps / 1_000_000, tx: item.tx_rate_bps / 1_000_000 }]) }}>Traffic</Button>}{!isAgent && !item.subscription_id && <Button size="small" onClick={() => { setMapSession(item); setSubscriptionID(null); setActionError('') }}>Map</Button>}</TableCell></TableRow>)}
        {!sessions.isLoading && rows.length === 0 && <TableRow><TableCell colSpan={isAgent ? 15 : 16} align="center">No matching PPPoE sessions.</TableCell></TableRow>}
        {sessions.isLoading && <TableRow><TableCell colSpan={isAgent ? 15 : 16} align="center">Loading PPPoE sessions...</TableCell></TableRow>}
      </TableBody></Table></TableContainer>
    </CardContent></Card>
    <Dialog open={detailSession !== null} onClose={() => setDetailSession(null)} fullWidth maxWidth="sm"><DialogTitle>{detailSession?.customer_name || detailSession?.username} — PPPoE Information</DialogTitle><DialogContent><Grid container spacing={2} sx={{ pt: 1 }}><Grid size={{ xs: 6 }}><Typography color="text.secondary">Username</Typography><Typography>{detailSession?.username}</Typography></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">Connection</Typography><Chip size="small" color={detailSession?.active ? 'success' : 'default'} label={detailSession?.active ? 'ONLINE' : 'OFFLINE'} /></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">Customer</Typography><Typography>{detailSession?.customer_code || 'Not mapped'}<br />{detailSession?.customer_name}</Typography></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">Package</Typography><Typography>{detailSession?.package_name || '—'}</Typography></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">Mobile</Typography><Typography>{detailCustomer.data?.mobile || '—'}</Typography></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">Billing day</Typography><Typography>{detailCustomer.data?.billing_day ? `${detailCustomer.data.billing_day} of month` : '—'}</Typography></Grid><Grid size={{ xs: 12 }}><Typography color="text.secondary">Address</Typography><Typography>{detailCustomer.data?.address || detailCustomer.data?.road_or_area || '—'}</Typography></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">Router / IP</Typography><Typography>{detailSession?.router_code}<br />{detailSession?.address}</Typography></Grid><Grid size={{ xs: 6 }}><Typography color="text.secondary">MAC / Uptime</Typography><Typography>{detailSession?.caller_id || '—'}<br />{detailSession?.uptime || '—'}</Typography></Grid></Grid></DialogContent><DialogActions><Button onClick={() => { if (detailSession?.active) { setTrafficSession(detailSession); setTrafficSamples([{ time: new Date().toLocaleTimeString(), rx: detailSession.rx_rate_bps / 1_000_000, tx: detailSession.tx_rate_bps / 1_000_000 }]) } }}>Live Traffic</Button><Button onClick={() => setDetailSession(null)}>Close</Button></DialogActions></Dialog>
    <Dialog open={trafficSession !== null} onClose={() => setTrafficSession(null)} fullWidth maxWidth="md"><DialogTitle>Live Traffic — {currentTrafficSession?.username}</DialogTitle><DialogContent><Grid container spacing={2} sx={{ mb: 2 }}><Grid size={{ xs: 6 }}><Card variant="outlined"><CardContent><Typography color="text.secondary">Download</Typography><Typography variant="h5">{formatRate(liveTraffic.data?.download_bps ?? currentTrafficSession?.rx_rate_bps ?? 0)}</Typography></CardContent></Card></Grid><Grid size={{ xs: 6 }}><Card variant="outlined"><CardContent><Typography color="text.secondary">Upload</Typography><Typography variant="h5">{formatRate(liveTraffic.data?.upload_bps ?? currentTrafficSession?.tx_rate_bps ?? 0)}</Typography></CardContent></Card></Grid></Grid>{liveTraffic.isError && <Alert severity="error" sx={{ mb: 2 }}>Router থেকে live traffic পাওয়া যায়নি।</Alert>}<Box sx={{ width: '100%', height: 300 }}><ResponsiveContainer><AreaChart data={trafficSamples} margin={{ top: 12, right: 16, bottom: 8, left: 8 }}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="time" interval="preserveStartEnd" minTickGap={42} tick={{ fontSize: 11 }} /><YAxis width={48} tick={{ fontSize: 11 }} tickFormatter={(value) => Number(value).toFixed(value < 1 ? 2 : 1)} label={{ value: 'Mbps', angle: -90, position: 'insideLeft', style: { fontSize: 12 } }} /><ChartTooltip formatter={(value) => [`${Number(value ?? 0).toFixed(3)} Mbps`]} /><Legend wrapperStyle={{ fontSize: 12 }} /><Area type="monotone" dataKey="rx" name="Download" stroke="#2563eb" fill="#93c5fd" /><Area type="monotone" dataKey="tx" name="Upload" stroke="#db2777" fill="#f9a8d4" /></AreaChart></ResponsiveContainer></Box><Typography variant="caption" color="text.secondary">প্রতি ৫ সেকেন্ডে Router থেকে সরাসরি নতুন sample নেওয়া হয়।</Typography></DialogContent><DialogActions><Button color="error" onClick={() => setTrafficSession(null)}>Stop Live Traffic</Button></DialogActions></Dialog>
    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1, mt: 4, mb: 1 }}><Typography variant="h5" sx={{ fontWeight: 700 }}>Top PPPoE Traffic</Typography><Box sx={{ display: 'flex', gap: 1 }}><TextField select size="small" label="Usage period" value={usageDays} onChange={(event) => setUsageDays(Number(event.target.value))} sx={{ minWidth: 150 }}>{[1, 7, 30, 90].map((days) => <MenuItem key={days} value={days}>{days === 1 ? 'Today' : `Last ${days} days`}</MenuItem>)}</TextField><Button variant="outlined" onClick={exportUsageCSV} disabled={userUsage.isLoading || (userUsage.data?.length ?? 0) === 0}>Export CSV</Button></Box></Box>
    <Card><CardContent><TableContainer><Table size="small"><TableHead><TableRow><TableCell>Username</TableCell><TableCell>Router</TableCell><TableCell>RX</TableCell><TableCell>TX</TableCell><TableCell>Total</TableCell></TableRow></TableHead><TableBody>{(userUsage.data ?? []).map((item) => <TableRow key={`${item.router_id}-${item.username}`}><TableCell>{item.username}</TableCell><TableCell>{item.router_code}</TableCell><TableCell>{formatTraffic(item.rx_bytes)}</TableCell><TableCell>{formatTraffic(item.tx_bytes)}</TableCell><TableCell>{formatTraffic(item.rx_bytes + item.tx_bytes)}</TableCell></TableRow>)}{!userUsage.isLoading && (userUsage.data?.length ?? 0) === 0 && <TableRow><TableCell colSpan={5} align="center">No traffic usage collected yet.</TableCell></TableRow>}</TableBody></Table></TableContainer></CardContent></Card>
    {!isAgent && <><Typography variant="h5" sx={{ mt: 4, mb: 1, fontWeight: 700 }}>MikroTik PPP Secrets</Typography>
    <Typography color="text.secondary" sx={{ mb: 2 }}>Existing RouterOS PPP accounts, including offline and disabled users. Passwords are never imported.</Typography>
    <Card><CardContent>
      <Box sx={{ display: 'flex', gap: 2, mb: 2 }}><Chip color="primary" label={`Secrets ${secrets.data?.length ?? 0}`} /><Chip color={unmappedSecrets ? 'warning' : 'default'} label={`Unmapped ${unmappedSecrets}`} /></Box>
      {secrets.isError && <Alert severity="error" sx={{ mb: 2 }}>Failed to load PPP secrets.</Alert>}
      <TableContainer><Table sx={{ minWidth: 1100 }} size="small"><TableHead><TableRow><TableCell>#</TableCell><TableCell>Username</TableCell><TableCell>Router</TableCell><TableCell>Profile</TableCell><TableCell>Customer</TableCell><TableCell>Status</TableCell><TableCell>Mapping</TableCell><TableCell>Last Sync</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead><TableBody>
        {secretRows.map((item, index) => <TableRow key={item.id} hover><TableCell>{index + 1}</TableCell><TableCell sx={{ fontWeight: 700 }}>{item.username}</TableCell><TableCell>{item.router_code}<br />{item.router_name}</TableCell><TableCell>{item.profile || '—'}</TableCell><TableCell>{item.customer_name ? <>{item.customer_code}<br />{item.customer_name}</> : '—'}</TableCell><TableCell><Chip size="small" color={item.disabled ? 'default' : 'success'} label={item.disabled ? 'DISABLED' : 'ENABLED'} /></TableCell><TableCell><Chip size="small" color={item.subscription_id ? 'success' : 'warning'} label={item.subscription_id ? 'MAPPED' : 'UNMAPPED'} /></TableCell><TableCell>{new Date(item.last_seen_at).toLocaleString()}</TableCell><TableCell align="right">{!item.subscription_id && <Button size="small" onClick={() => { setMapSecret(item); setSubscriptionID(null); setActionError('') }}>Map</Button>}</TableCell></TableRow>)}
        {!secrets.isLoading && secretRows.length === 0 && <TableRow><TableCell colSpan={9} align="center">No synchronized PPP secrets.</TableCell></TableRow>}
        {secrets.isLoading && <TableRow><TableCell colSpan={9} align="center">Loading PPP secrets...</TableCell></TableRow>}
      </TableBody></Table></TableContainer>
    </CardContent></Card>
    <Dialog open={mapSession !== null} onClose={() => !mappingBusy && setMapSession(null)} fullWidth maxWidth="sm"><DialogTitle>Map PPPoE User to Subscription</DialogTitle><DialogContent><Alert severity="info" sx={{ mb: 2 }}>This updates TS-Cloud only. No MikroTik configuration will be changed.</Alert><Typography sx={{ mb: 2 }}><b>{mapSession?.username}</b> · {mapSession?.router_code} · {mapSession?.address}</Typography><Autocomplete options={(subscriptions.data?.subscriptions ?? []).filter((item) => item.status !== 'DISCONNECTED')} value={(subscriptions.data?.subscriptions ?? []).find((item) => item.id === subscriptionID) ?? null} onChange={(_, value) => setSubscriptionID(value?.id ?? null)} getOptionLabel={(item) => `${item.subscription_code} — ${item.customer_code || `Customer #${item.customer_id}`} ${item.customer_name} — ${item.package_name || `Package #${item.package_id}`}${item.pppoe_username ? ` — current: ${item.pppoe_username}` : ''}`} renderInput={(params) => <TextField {...params} label="Search and select subscription" />} /></DialogContent><DialogActions><Button onClick={() => setMapSession(null)} disabled={mappingBusy}>Cancel</Button><Button variant="contained" onClick={() => void mapSelected()} disabled={mappingBusy || !subscriptionID}>{mappingBusy ? 'Mapping...' : 'Confirm Mapping'}</Button></DialogActions></Dialog>
    <Dialog open={mapSecret !== null} onClose={() => !mappingBusy && setMapSecret(null)} fullWidth maxWidth="sm"><DialogTitle>Map PPP Secret to Subscription</DialogTitle><DialogContent><Alert severity="info" sx={{ mb: 2 }}>Only the username and router mapping are saved. MikroTik and its password are not changed.</Alert><Typography sx={{ mb: 2 }}><b>{mapSecret?.username}</b> · {mapSecret?.router_code} · {mapSecret?.profile}</Typography><Autocomplete options={(subscriptions.data?.subscriptions ?? []).filter((item) => item.status !== 'DISCONNECTED')} value={(subscriptions.data?.subscriptions ?? []).find((item) => item.id === subscriptionID) ?? null} onChange={(_, value) => setSubscriptionID(value?.id ?? null)} getOptionLabel={(item) => `${item.subscription_code} — ${item.customer_code || `Customer #${item.customer_id}`} ${item.customer_name} — ${item.package_name || `Package #${item.package_id}`}${item.pppoe_username ? ` — current: ${item.pppoe_username}` : ''}`} renderInput={(params) => <TextField {...params} label="Search and select subscription" />} /></DialogContent><DialogActions><Button onClick={() => setMapSecret(null)} disabled={mappingBusy}>Cancel</Button><Button variant="contained" onClick={() => void mapSelectedSecret()} disabled={mappingBusy || !subscriptionID}>{mappingBusy ? 'Mapping...' : 'Confirm Mapping'}</Button></DialogActions></Dialog></>}
  </Box>
}
