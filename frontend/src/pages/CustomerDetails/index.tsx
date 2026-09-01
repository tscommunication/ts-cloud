import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Box, Button, Card, CardContent, Chip, Divider, Grid, IconButton, Stack, Tooltip, Typography } from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import VisibilityIcon from '@mui/icons-material/Visibility'
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff'
import WifiIcon from '@mui/icons-material/Wifi'

import { getCustomer, getCustomerInternetCredential, getCustomerNetworkPath, getCustomerSummary } from '../../api/customers'
import { getSubscriptions } from '../../api/subscriptions'
import { getNetworkPPPoESessions } from '../../api/networkRouters'

const rate = (bps: number) => bps >= 1_000_000 ? `${(bps / 1_000_000).toFixed(2)} Mbps` : `${Math.round(bps / 1_000)} Kbps`
const bytes = (value: number) => value >= 1_000_000_000 ? `${(value / 1_000_000_000).toFixed(2)} GB` : value >= 1_000_000 ? `${(value / 1_000_000).toFixed(2)} MB` : `${Math.round(value / 1_000)} KB`
const value = (item?: string | number | null) => item === undefined || item === null || item === '' ? '—' : item
const DetailLine = ({ label, children }: { label: string; children: React.ReactNode }) => <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, py: 0.75, borderBottom: '1px solid', borderColor: 'divider' }}><Typography variant="body2" color="text.secondary">{label}</Typography><Typography variant="body2" sx={{ textAlign: 'right', fontWeight: 600, overflowWrap: 'anywhere' }}>{children}</Typography></Box>

export default function CustomerDetails() {
  const { id } = useParams()
  const customerID = Number(id)
  const navigate = useNavigate()
  const [showPassword, setShowPassword] = useState(false)
  const customer = useQuery({ queryKey: ['customer', customerID], queryFn: () => getCustomer(customerID), enabled: customerID > 0 })
  const summary = useQuery({ queryKey: ['customer-summary', customerID], queryFn: () => getCustomerSummary(customerID), enabled: customerID > 0 })
  const credential = useQuery({ queryKey: ['customer-internet-credential', customerID], queryFn: () => getCustomerInternetCredential(customerID), enabled: customerID > 0 })
  const networkPath = useQuery({ queryKey: ['customer-network-path', customerID], queryFn: () => getCustomerNetworkPath(customerID), enabled: customerID > 0 })
  const subscriptions = useQuery({ queryKey: ['subscriptions', 'customer-details'], queryFn: () => getSubscriptions() })
  const sessions = useQuery({ queryKey: ['network-pppoe-sessions', 'customer-details'], queryFn: () => getNetworkPPPoESessions(true), refetchInterval: 30000 })
  const subscription = useMemo(() => (subscriptions.data?.subscriptions ?? []).find((item) => item.customer_id === customerID), [subscriptions.data, customerID])
  const session = useMemo(() => (sessions.data ?? []).find((item) => item.customer_id === customerID), [sessions.data, customerID])
  const profile = networkPath.data?.profile
  const onu = networkPath.data?.onu
  const optical = networkPath.data?.optical
  const pppoeUsername = subscription?.pppoe_username || credential.data?.pppoe_username || session?.username || ''
  const packageName = subscription?.package_name || session?.package_name || ''
  const serviceStatus = subscription?.status || (session?.active ? 'ACTIVE' : '')
  const expiry = subscription?.expiry_date ? new Date(subscription.expiry_date) : null
  if (customer.isError) return <Alert severity="error">Customer not found.</Alert>
  const row = customer.data

  return <Box>
    <Button startIcon={<ArrowBackIcon />} onClick={() => navigate(-1)} sx={{ mb: 2 }}>Back</Button>
    <Card sx={{ mb: 2, background: 'linear-gradient(120deg, #1e3a8a, #4f46e5)', color: 'white' }}><CardContent>
      <Stack direction={{ xs: 'column', sm: 'row' }} sx={{ justifyContent: 'space-between', gap: 1 }}><Box><Typography variant="h4" sx={{ fontWeight: 700 }}>{row?.full_name || 'Loading customer…'} {row && <Chip size="small" color={row.status === 'ACTIVE' ? 'success' : 'default'} label={row.status} sx={{ ml: 1 }} />}</Typography><Typography sx={{ mt: 1 }}>{pppoeUsername || 'No PPPoE username'} · {packageName || 'No package'} · CID {row?.customer_code}</Typography></Box><Typography>{session?.agent_name || 'Unassigned agent'}</Typography></Stack>
    </CardContent></Card>

    <Grid container spacing={2}>
      <Grid size={{ xs: 12, md: 5 }}><Card><CardContent><Typography variant="h6">Customer Information</Typography><Divider sx={{ my: 1 }} />
        <DetailLine label="CID">{value(row?.customer_code)}</DetailLine><DetailLine label="Username">{value(pppoeUsername)}</DetailLine><DetailLine label="Password">{credential.data?.pppoe_password ? <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'flex-end' }}><span>{showPassword ? credential.data.pppoe_password : '••••••••'}</span><Tooltip title={showPassword ? 'Hide password' : 'Show password'}><IconButton size="small" onClick={() => setShowPassword((current) => !current)}>{showPassword ? <VisibilityOffIcon fontSize="small" /> : <VisibilityIcon fontSize="small" />}</IconButton></Tooltip></Stack> : 'Not saved'}</DetailLine><DetailLine label="Contact">{value(row?.mobile)}</DetailLine><DetailLine label="Alternative contact">{value(row?.alt_mobile)}</DetailLine><DetailLine label="Address">{value(row?.address || row?.road_or_area)}</DetailLine><DetailLine label="Area">{[row?.upazila, row?.district].filter(Boolean).join(', ') || '—'}</DetailLine><DetailLine label="Agent / Reseller">{value(session?.agent_name)}</DetailLine><DetailLine label="Billing date">{row?.billing_day ? `${row.billing_day} of month` : '—'}</DetailLine><DetailLine label="Package">{value(packageName)}</DetailLine><DetailLine label="Expiry">{expiry ? expiry.toLocaleDateString() : '—'}</DetailLine><DetailLine label="Status">{value(serviceStatus)}</DetailLine>
      </CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 3 }}><Card sx={{ height: '100%' }}><CardContent><Typography variant="h6">Connection Status</Typography><Divider sx={{ my: 1 }} /><Stack spacing={1.25}><Chip icon={<WifiIcon />} color={session ? 'success' : 'default'} label={session ? 'ONLINE' : 'OFFLINE'} /><DetailLine label="MikroTik">{value(session?.router_name || session?.router_code)}</DetailLine><DetailLine label="IP address">{value(session?.address || credential.data?.static_ip_address)}</DetailLine><DetailLine label="Caller ID / MAC">{value(session?.caller_id || credential.data?.mac_address)}</DetailLine><DetailLine label="Uptime">{value(session?.uptime)}</DetailLine><DetailLine label="Live RX">{rate(session?.rx_rate_bps ?? 0)}</DetailLine><DetailLine label="Live TX">{rate(session?.tx_rate_bps ?? 0)}</DetailLine></Stack></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 4 }}><Card sx={{ height: '100%' }}><CardContent><Typography variant="h6">Other Details</Typography><Divider sx={{ my: 1 }} /><DetailLine label="NID">{value(row?.nid)}</DetailLine><DetailLine label="Father name">{value(row?.father_name)}</DetailLine><DetailLine label="Mother name">{value(row?.mother_name)}</DetailLine><DetailLine label="Date of birth">{value(row?.date_of_birth)}</DetailLine><DetailLine label="Joining date">{value(row?.joining_date)}</DetailLine><DetailLine label="Email">{value(row?.email)}</DetailLine><DetailLine label="Occupation">{value(row?.occupation)}</DetailLine><DetailLine label="Company">{value(row?.company_name)}</DetailLine><DetailLine label="TIN">{value(row?.tin)}</DetailLine><DetailLine label="Outstanding">{summary.data ? `${summary.data.outstanding_amount} BDT` : '—'}</DetailLine></CardContent></Card></Grid>

      <Grid size={{ xs: 12, md: 5 }}><Card><CardContent><Typography variant="h6">Session & Usage</Typography><Divider sx={{ my: 1 }} /><Grid container spacing={2}><Grid size={{ xs: 12, sm: 6 }}><DetailLine label="Last link / seen">{session?.last_seen_at ? new Date(session.last_seen_at).toLocaleString() : '—'}</DetailLine><DetailLine label="Session ID">{value(session?.session_id)}</DetailLine><DetailLine label="Client MAC">{value(session?.caller_id)}</DetailLine></Grid><Grid size={{ xs: 12, sm: 6 }}><Typography variant="body2" color="text.secondary">Current session traffic</Typography><Typography variant="h6" color="primary">↓ {bytes(session?.rx_bytes ?? 0)}</Typography><Typography variant="h6" color="secondary">↑ {bytes(session?.tx_bytes ?? 0)}</Typography><Typography variant="caption" color="text.secondary">Counters reset when RouterOS session reconnects.</Typography></Grid></Grid></CardContent></Card></Grid>
      <Grid size={{ xs: 12, md: 7 }}><Card><CardContent><Typography variant="h6">ONU & Network Topology</Typography><Divider sx={{ my: 1 }} />{!onu && !profile?.onu_mac && !profile?.onu_serial ? <Typography color="text.secondary">No automatic ONU match yet. It will appear after the mapped PPPoE user's caller-ID/MAC is present in monitored OLT inventory.</Typography> : <Grid container spacing={2}><Grid size={{ xs: 12, sm: 4 }}><Typography sx={{ fontWeight: 700 }}>ONU</Typography><DetailLine label="Model">{value(profile?.onu_model || profile?.onu_type || onu?.model)}</DetailLine><DetailLine label="MAC">{value(profile?.onu_mac || onu?.mac_address)}</DetailLine><DetailLine label="Serial">{value(profile?.onu_serial || profile?.onu_sn || onu?.serial_number)}</DetailLine></Grid><Grid size={{ xs: 12, sm: 4 }}><Typography sx={{ fontWeight: 700 }}>OLT / Switch</Typography><DetailLine label="OLT">{networkPath.data?.olt_code ? `${networkPath.data.olt_code} — ${networkPath.data.olt_name}` : '—'}</DetailLine><DetailLine label="PON / Port">{`${value(profile?.olt_pon || (onu ? onu.pon_no : ''))} / ${value(profile?.olt_port || (onu ? onu.onu_no : ''))}`}</DetailLine><DetailLine label="Switch port">{value(profile?.switch_port)}</DetailLine></Grid><Grid size={{ xs: 12, sm: 4 }}><Typography sx={{ fontWeight: 700 }}>Laser / MikroTik</Typography><DetailLine label="RX laser">{optical?.rx_power_dbm !== undefined && optical?.rx_power_dbm !== null ? `${optical.rx_power_dbm.toFixed(2)} dBm` : 'No reading'}</DetailLine><DetailLine label="TX laser">{optical?.tx_power_dbm !== undefined && optical?.tx_power_dbm !== null ? `${optical.tx_power_dbm.toFixed(2)} dBm` : 'No reading'}</DetailLine><DetailLine label="MikroTik port">{value(profile?.mikrotik_port)}</DetailLine></Grid></Grid>}</CardContent></Card></Grid>
    </Grid>
  </Box>
}
