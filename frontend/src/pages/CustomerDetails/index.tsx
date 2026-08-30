import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Box, Button, Card, CardContent, Chip, Grid, Typography } from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import WifiIcon from '@mui/icons-material/Wifi'

import { getCustomer, getCustomerSummary } from '../../api/customers'
import { getSubscriptions } from '../../api/subscriptions'
import { getNetworkPPPoESessions } from '../../api/networkRouters'

const rate = (bps: number) => bps >= 1_000_000 ? `${(bps / 1_000_000).toFixed(2)} Mbps` : `${Math.round(bps / 1_000)} Kbps`

export default function CustomerDetails() {
  const { id } = useParams()
  const customerID = Number(id)
  const navigate = useNavigate()
  const customer = useQuery({ queryKey: ['customer', customerID], queryFn: () => getCustomer(customerID), enabled: customerID > 0 })
  const summary = useQuery({ queryKey: ['customer-summary', customerID], queryFn: () => getCustomerSummary(customerID), enabled: customerID > 0 })
  const subscriptions = useQuery({ queryKey: ['subscriptions', 'customer-details'], queryFn: () => getSubscriptions() })
  const sessions = useQuery({ queryKey: ['network-pppoe-sessions', 'customer-details'], queryFn: () => getNetworkPPPoESessions(true), refetchInterval: 30000 })
  const subscription = useMemo(() => (subscriptions.data?.subscriptions ?? []).find((item) => item.customer_id === customerID), [subscriptions.data, customerID])
  const session = useMemo(() => (sessions.data ?? []).find((item) => item.customer_id === customerID), [sessions.data, customerID])
  if (customer.isError) return <Alert severity="error">Customer not found.</Alert>
  const row = customer.data
  return <Box><Button startIcon={<ArrowBackIcon />} onClick={() => navigate(-1)} sx={{ mb: 2 }}>Back</Button>
    <Card sx={{ mb: 3, background: 'linear-gradient(120deg, #1e3a8a, #4f46e5)', color: 'white' }}><CardContent><Typography variant="h4" sx={{ fontWeight: 700 }}>{row?.full_name || 'Loading customer…'} {row && <Chip size="small" color={row.status === 'ACTIVE' ? 'success' : 'default'} label={row.status} sx={{ ml: 1 }} />}</Typography><Typography sx={{ mt: 1 }}>{subscription?.pppoe_username || 'No PPPoE username'} · {subscription?.package_name || 'No package'} · CID {row?.customer_code}</Typography></CardContent></Card>
    <Grid container spacing={2}><Grid size={{ xs: 12, md: 5 }}><Card><CardContent><Typography variant="h6">Customer Information</Typography><Typography sx={{ mt: 2 }}><b>Mobile:</b> {row?.mobile || '—'}</Typography><Typography><b>Address:</b> {row?.address || row?.road_or_area || '—'}</Typography><Typography><b>Billing day:</b> {row?.billing_day || '—'} of month</Typography><Typography><b>NID:</b> {row?.nid || '—'}</Typography></CardContent></Card></Grid><Grid size={{ xs: 12, md: 4 }}><Card><CardContent><Typography variant="h6">Service & Billing</Typography><Typography sx={{ mt: 2 }}><b>Package:</b> {subscription?.package_name || '—'}</Typography><Typography><b>Status:</b> {subscription?.status || '—'}</Typography><Typography><b>Expiry:</b> {subscription?.expiry_date ? new Date(subscription.expiry_date).toLocaleDateString() : '—'}</Typography><Typography><b>Outstanding:</b> {summary.data?.outstanding_amount ?? 0} BDT</Typography></CardContent></Card></Grid><Grid size={{ xs: 12, md: 3 }}><Card><CardContent><Typography variant="h6">Live Connection</Typography><Typography sx={{ mt: 2 }}><Chip icon={<WifiIcon />} size="small" color={session ? 'success' : 'default'} label={session ? 'ONLINE' : 'OFFLINE'} /></Typography><Typography sx={{ mt: 1 }}><b>IP:</b> {session?.address || '—'}</Typography><Typography><b>RX:</b> {rate(session?.rx_rate_bps ?? 0)}</Typography><Typography><b>TX:</b> {rate(session?.tx_rate_bps ?? 0)}</Typography><Typography><b>Uptime:</b> {session?.uptime || '—'}</Typography></CardContent></Card></Grid></Grid>
  </Box>
}
