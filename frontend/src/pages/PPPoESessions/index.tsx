import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { Alert, Autocomplete, Box, Button, Card, CardContent, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Grid, MenuItem, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Typography } from '@mui/material'
import PeopleIcon from '@mui/icons-material/People'
import VerifiedUserIcon from '@mui/icons-material/VerifiedUser'
import LinkOffIcon from '@mui/icons-material/LinkOff'

import { getNetworkPPPoESessions, mapNetworkPPPoESession, type NetworkRouterPPPoESession } from '../../api/networkRouters'
import { getSubscriptions } from '../../api/subscriptions'
import { getAPIErrorMessage } from '../../api/errors'

export default function PPPoESessions() {
  const queryClient = useQueryClient()
  const [params, setParams] = useSearchParams()
  const [search, setSearch] = useState('')
  const [mapSession, setMapSession] = useState<NetworkRouterPPPoESession | null>(null)
  const [subscriptionID, setSubscriptionID] = useState<number | null>(null)
  const [mappingBusy, setMappingBusy] = useState(false)
  const [actionError, setActionError] = useState('')
  const mapping = params.get('mapping') ?? 'all'
  const sessions = useQuery({ queryKey: ['network-pppoe-sessions', 'active'], queryFn: () => getNetworkPPPoESessions(true), refetchInterval: 30000 })
  const subscriptions = useQuery({ queryKey: ['subscriptions', 'pppoe-mapping'], queryFn: () => getSubscriptions() })
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
  const mapSelected = async () => {
    if (!mapSession || !subscriptionID) return
    try {
      setMappingBusy(true); setActionError('')
      await mapNetworkPPPoESession(mapSession.id, subscriptionID)
      await Promise.all([queryClient.invalidateQueries({ queryKey: ['network-pppoe-sessions'] }), queryClient.invalidateQueries({ queryKey: ['network-pppoe-summary'] })])
      setMapSession(null); setSubscriptionID(null)
    } catch (error) { setActionError(getAPIErrorMessage(error, 'Failed to map PPPoE user.')) } finally { setMappingBusy(false) }
  }

  return <Box>
    <Typography variant="h4" sx={{ fontWeight: 700 }}>Live PPPoE Users</Typography>
    <Typography color="text.secondary" sx={{ mb: 3 }}>Active sessions synchronized from every MikroTik router. Refreshes every 30 seconds.</Typography>
    {sessions.isError && <Alert severity="error" sx={{ mb: 2 }}>Failed to load live PPPoE users.</Alert>}
    {actionError && <Alert severity="error" onClose={() => setActionError('')} sx={{ mb: 2 }}>{actionError}</Alert>}
    <Grid container spacing={2} sx={{ mb: 3 }}>
      {[{ label: 'Active Users', value: sessions.data?.length ?? 0, icon: <PeopleIcon />, color: 'primary.main' }, { label: 'Mapped Users', value: mapped, icon: <VerifiedUserIcon />, color: 'success.main' }, { label: 'Unmapped Users', value: unmapped, icon: <LinkOffIcon />, color: unmapped > 0 ? 'warning.main' : 'text.secondary' }].map((stat) => <Grid key={stat.label} size={{ xs: 12, sm: 4 }}><Card><CardContent><Box sx={{ display: 'flex', justifyContent: 'space-between' }}><Box><Typography color="text.secondary">{stat.label}</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>{stat.value}</Typography></Box><Box sx={{ color: stat.color }}>{stat.icon}</Box></Box></CardContent></Card></Grid>)}
    </Grid>
    <Card><CardContent>
      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 8 }}><TextField fullWidth label="Search username, customer, package, IP or router" value={search} onChange={(event) => setSearch(event.target.value)} /></Grid>
        <Grid size={{ xs: 12, md: 4 }}><TextField fullWidth select label="Mapping" value={mapping} onChange={(event) => setParams(event.target.value === 'all' ? {} : { mapping: event.target.value })}><MenuItem value="all">All active users</MenuItem><MenuItem value="mapped">Mapped only</MenuItem><MenuItem value="unmapped">Unmapped only</MenuItem></TextField></Grid>
      </Grid>
      <TableContainer><Table sx={{ minWidth: 1320 }} size="small"><TableHead><TableRow><TableCell>#</TableCell><TableCell>Username</TableCell><TableCell>Router</TableCell><TableCell>Customer</TableCell><TableCell>Package</TableCell><TableCell>Mapping</TableCell><TableCell>IP Address</TableCell><TableCell>Caller ID / MAC</TableCell><TableCell>Service</TableCell><TableCell>Uptime</TableCell><TableCell>Last Seen</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead><TableBody>
        {rows.map((item, index) => <TableRow key={item.id} hover><TableCell>{index + 1}</TableCell><TableCell sx={{ fontWeight: 700 }}>{item.username}</TableCell><TableCell>{item.router_code}<br />{item.router_name}</TableCell><TableCell>{item.customer_name ? <>{item.customer_code}<br />{item.customer_name}</> : '—'}</TableCell><TableCell>{item.package_name ? <>{item.package_code}<br />{item.package_name}</> : '—'}</TableCell><TableCell><Chip size="small" color={item.subscription_id ? 'success' : 'warning'} label={item.subscription_id ? 'MAPPED' : 'UNMAPPED'} /></TableCell><TableCell>{item.address || '—'}</TableCell><TableCell>{item.caller_id || '—'}</TableCell><TableCell>{item.service || '—'}</TableCell><TableCell>{item.uptime || '—'}</TableCell><TableCell>{new Date(item.last_seen_at).toLocaleString()}</TableCell><TableCell align="right">{!item.subscription_id && <Button size="small" onClick={() => { setMapSession(item); setSubscriptionID(null); setActionError('') }}>Map</Button>}</TableCell></TableRow>)}
        {!sessions.isLoading && rows.length === 0 && <TableRow><TableCell colSpan={12} align="center">No matching active PPPoE users.</TableCell></TableRow>}
        {sessions.isLoading && <TableRow><TableCell colSpan={12} align="center">Loading live PPPoE users...</TableCell></TableRow>}
      </TableBody></Table></TableContainer>
    </CardContent></Card>
    <Dialog open={mapSession !== null} onClose={() => !mappingBusy && setMapSession(null)} fullWidth maxWidth="sm"><DialogTitle>Map PPPoE User to Subscription</DialogTitle><DialogContent><Alert severity="info" sx={{ mb: 2 }}>This updates TS-Cloud only. No MikroTik configuration will be changed.</Alert><Typography sx={{ mb: 2 }}><b>{mapSession?.username}</b> · {mapSession?.router_code} · {mapSession?.address}</Typography><Autocomplete options={(subscriptions.data?.subscriptions ?? []).filter((item) => item.status !== 'DISCONNECTED')} value={(subscriptions.data?.subscriptions ?? []).find((item) => item.id === subscriptionID) ?? null} onChange={(_, value) => setSubscriptionID(value?.id ?? null)} getOptionLabel={(item) => `${item.subscription_code} — ${item.customer_code || `Customer #${item.customer_id}`} ${item.customer_name} — ${item.package_name || `Package #${item.package_id}`}${item.pppoe_username ? ` — current: ${item.pppoe_username}` : ''}`} renderInput={(params) => <TextField {...params} label="Search and select subscription" />} /></DialogContent><DialogActions><Button onClick={() => setMapSession(null)} disabled={mappingBusy}>Cancel</Button><Button variant="contained" onClick={() => void mapSelected()} disabled={mappingBusy || !subscriptionID}>{mappingBusy ? 'Mapping...' : 'Confirm Mapping'}</Button></DialogActions></Dialog>
  </Box>
}
