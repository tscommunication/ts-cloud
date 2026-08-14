import { useQuery } from '@tanstack/react-query'
import { Alert, Box, Card, CardContent, CircularProgress, Grid, Typography } from '@mui/material'
import PeopleIcon from '@mui/icons-material/People'
import SubscriptionsIcon from '@mui/icons-material/Subscriptions'
import ReceiptIcon from '@mui/icons-material/Receipt'
import PaymentsIcon from '@mui/icons-material/Payments'

import { getAgentDashboard } from '../../api/agentDashboard'
import { getAPIErrorMessage } from '../../api/errors'

const money = (value: number | undefined) => `BDT ${Number(value ?? 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`

export default function AgentDashboard() {
  const summary = useQuery({ queryKey: ['agent-dashboard'], queryFn: getAgentDashboard, refetchInterval: 30000 })
  if (summary.isLoading) return <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}><CircularProgress /></Box>
  if (summary.isError) return <Alert severity="error">{getAPIErrorMessage(summary.error, 'Failed to load agent dashboard.')}</Alert>

  const data = summary.data
  const cards = [
    ['My Customers', data?.total_customers ?? 0, <PeopleIcon />],
    ['Active Customers', data?.active_customers ?? 0, <PeopleIcon />],
    ['Active Subscriptions', data?.active_subscriptions ?? 0, <SubscriptionsIcon />],
    ['Overdue Invoices', data?.overdue_invoices ?? 0, <ReceiptIcon />],
    ['Total Collected', money(data?.total_collected), <PaymentsIcon />],
    ["Today's Collection", money(data?.today_collected), <PaymentsIcon />],
    ['Outstanding Bills', money(data?.total_outstanding), <ReceiptIcon />],
    ['Commission Payable', money(data?.commission_payable), <PaymentsIcon />],
	['Voided Collections', `${data?.voided_collections ?? 0} · ${money(data?.voided_amount)}`, <ReceiptIcon />],
  ]

  return <Box>
    <Typography variant="h4" sx={{ fontWeight: 700, mb: 1 }}>Agent Dashboard</Typography>
    <Typography color="text.secondary" sx={{ mb: 3 }}>Your customer billing, collection and commission overview.</Typography>
    <Grid container spacing={2}>
      {cards.map(([label, value, icon]) => <Grid key={String(label)} size={{ xs: 12, sm: 6, lg: 3 }}><Card sx={{ height: '100%' }}><CardContent><Box sx={{ display: 'flex', justifyContent: 'space-between', color: 'primary.main', mb: 2 }}><Typography color="text.secondary">{label}</Typography>{icon}</Box><Typography variant="h5" sx={{ fontWeight: 700 }}>{value}</Typography></CardContent></Card></Grid>)}
    </Grid>
    <Grid container spacing={2} sx={{ mt: 1 }}>
      <Grid size={{ xs: 12, md: 4 }}><Card><CardContent><Typography color="text.secondary">Total Invoiced</Typography><Typography variant="h6">{money(data?.total_invoiced)}</Typography></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 4 }}><Card><CardContent><Typography color="text.secondary">Commission Earned</Typography><Typography variant="h6">{money(data?.commission_earned)}</Typography></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 4 }}><Card><CardContent><Typography color="text.secondary">Commission Paid</Typography><Typography variant="h6">{money(data?.commission_paid)}</Typography></CardContent></Card></Grid>
    </Grid>
  </Box>
}
