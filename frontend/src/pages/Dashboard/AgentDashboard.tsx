import { useQuery } from '@tanstack/react-query'
import { Alert, Box, Button, Card, CardContent, CircularProgress, Grid, Stack, TextField, Typography } from '@mui/material'
import PeopleIcon from '@mui/icons-material/People'
import ReceiptIcon from '@mui/icons-material/Receipt'
import PaymentsIcon from '@mui/icons-material/Payments'
import WifiIcon from '@mui/icons-material/Wifi'
import WifiOffIcon from '@mui/icons-material/WifiOff'
import SearchIcon from '@mui/icons-material/Search'
import PersonAddIcon from '@mui/icons-material/PersonAdd'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import { useNavigate } from 'react-router-dom'
import { useState } from 'react'
import type { ReactNode } from 'react'

import { getAgentDashboard } from '../../api/agentDashboard'
import { getAPIErrorMessage } from '../../api/errors'

const money = (value: number | undefined) => `BDT ${Number(value ?? 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`

export default function AgentDashboard() {
	const navigate = useNavigate()
	const [search, setSearch] = useState('')
  const summary = useQuery({ queryKey: ['agent-dashboard'], queryFn: getAgentDashboard, refetchInterval: 30000 })
  if (summary.isLoading) return <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}><CircularProgress /></Box>
  if (summary.isError) return <Alert severity="error">{getAPIErrorMessage(summary.error, 'Failed to load agent dashboard.')}</Alert>

  const data = summary.data
  const cards: Array<{ label: string; value: string | number; icon: ReactNode; path: string; color: string }> = [
    { label: 'My Customers', value: data?.total_customers ?? 0, icon: <PeopleIcon />, path: '/customers', color: 'primary.main' },
    { label: 'Online Now', value: data?.online_customers ?? 0, icon: <WifiIcon />, path: '/customers?view=ONLINE', color: 'success.main' },
    { label: 'Offline Users', value: data?.offline_customers ?? 0, icon: <WifiOffIcon />, path: '/customers?view=OFFLINE', color: 'text.secondary' },
    { label: 'Expiry Attention', value: data?.expired_customers ?? 0, icon: <WarningAmberIcon />, path: '/customers?view=EXPIRED', color: 'warning.main' },
    { label: 'Overdue Invoices', value: data?.overdue_invoices ?? 0, icon: <ReceiptIcon />, path: '/invoices?status=OVERDUE', color: 'error.main' },
    { label: "Today's Collection", value: money(data?.today_collected), icon: <PaymentsIcon />, path: '/agent-collections', color: 'success.main' },
    { label: 'Outstanding Bills', value: money(data?.total_outstanding), icon: <ReceiptIcon />, path: '/invoices', color: 'warning.main' },
    { label: 'Commission Payable', value: money(data?.commission_payable), icon: <PaymentsIcon />, path: '/agent-collections', color: 'info.main' },
  ]

  return <Box>
    <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ justifyContent: 'space-between', alignItems: { md: 'center' }, mb: 3 }}>
      <Box><Typography variant="h4" sx={{ fontWeight: 700, mb: 0.5 }}>Agent Workspace</Typography><Typography color="text.secondary">Customers, live connections, collection and commission — only your assigned area.</Typography></Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ width: { xs: '100%', md: 'auto' } }}>
        <TextField size="small" label="Search CID, username, name or mobile" value={search} onChange={(event) => setSearch(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') navigate(`/customers?search=${encodeURIComponent(search.trim())}`) }} sx={{ minWidth: { md: 310 } }} />
        <Button variant="contained" startIcon={<SearchIcon />} onClick={() => navigate(`/customers?search=${encodeURIComponent(search.trim())}`)}>Search</Button>
        <Button variant="outlined" startIcon={<PersonAddIcon />} onClick={() => navigate('/customers?action=add')}>Add Customer</Button>
      </Stack>
    </Stack>
    <Grid container spacing={2}>
      {cards.map((card) => <Grid key={card.label} size={{ xs: 12, sm: 6, lg: 3 }}><Card sx={{ height: '100%', cursor: 'pointer' }} onClick={() => navigate(card.path)}><CardContent><Box sx={{ display: 'flex', justifyContent: 'space-between', color: card.color, mb: 2 }}><Typography color="text.secondary">{card.label}</Typography>{card.icon}</Box><Typography variant="h5" sx={{ fontWeight: 700 }}>{card.value}</Typography></CardContent></Card></Grid>)}
    </Grid>
    <Grid container spacing={2} sx={{ mt: 1 }}>
      <Grid size={{ xs: 12, md: 3 }}><Card><CardContent><Typography color="text.secondary">Active Services</Typography><Typography variant="h6">{data?.active_subscriptions ?? 0}</Typography></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 3 }}><Card><CardContent><Typography color="text.secondary">Total Collected</Typography><Typography variant="h6">{money(data?.total_collected)}</Typography></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 3 }}><Card><CardContent><Typography color="text.secondary">Commission Earned</Typography><Typography variant="h6">{money(data?.commission_earned)}</Typography></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 3 }}><Card><CardContent><Typography color="text.secondary">Commission Paid</Typography><Typography variant="h6">{money(data?.commission_paid)}</Typography></CardContent></Card></Grid>
    </Grid>
    <Card sx={{ mt: 2 }}><CardContent><Typography variant="h6" sx={{ fontWeight: 700 }}>Quick Actions</Typography><Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 2 }}><Button variant="outlined" onClick={() => navigate('/network/pppoe-sessions')}>Live PPPoE Users</Button><Button variant="outlined" onClick={() => navigate('/customers?view=EXPIRED')}>Expiry List</Button><Button variant="outlined" onClick={() => navigate('/agent-collections')}>Collections</Button><Button variant="outlined" onClick={() => navigate('/agent-collections')}>Commission & Settlement</Button></Stack></CardContent></Card>
  </Box>
}
