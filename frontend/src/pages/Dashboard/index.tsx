import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import type { KeyboardEvent } from 'react'
import {
  Box,
  Card,
  CardContent,
  Grid,
  Typography,
  Button,
  Alert,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material'

import PeopleIcon from '@mui/icons-material/People'
import CloudIcon from '@mui/icons-material/Cloud'
import UploadIcon from '@mui/icons-material/Upload'
import DownloadIcon from '@mui/icons-material/Download'
import LoginIcon from '@mui/icons-material/Login'
import RouterIcon from '@mui/icons-material/Router'
import WifiIcon from '@mui/icons-material/Wifi'
import VerifiedUserIcon from '@mui/icons-material/VerifiedUser'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import LinkOffIcon from '@mui/icons-material/LinkOff'
import DnsIcon from '@mui/icons-material/Dns'

import { getFTPDashboard } from '../../api/ftpDashboard'
import { getBillingRuns, getBillingSummary, runBilling } from '../../api/billing'
import { getStoredUser } from '../../api/auth'
import AgentDashboard from './AgentDashboard'
import OLTDashboard from '../OLTDashboard'
import { getNetworkPPPoESummary, getNetworkRouterAlerts, getNetworkRouters } from '../../api/networkRouters'
import { getNetworkDevices } from '../../api/networkDevices'
import { dashboardViews } from '../../dashboard/dashboardView'
import { useDashboardSettings } from '../../dashboard/useDashboardSettings'

function formatBytes(bytes: number) {
  if (bytes === 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.floor(Math.log(bytes) / Math.log(1024))

  return `${(bytes / Math.pow(1024, index)).toFixed(2)} ${units[index]}`
}

function AdminDashboard() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { dashboardView } = useDashboardSettings()
  const { data, isLoading, isError } = useQuery({
    queryKey: ['ftp-dashboard'],
    queryFn: getFTPDashboard,
    refetchInterval: 5000,
  })
  const billing = useQuery({ queryKey: ['billing-summary'], queryFn: getBillingSummary })
  const runs = useQuery({ queryKey: ['billing-runs'], queryFn: getBillingRuns })
  const routers = useQuery({ queryKey: ['network-routers'], queryFn: getNetworkRouters, refetchInterval: 30000 })
  const networkDevices = useQuery({ queryKey: ['network-devices'], queryFn: getNetworkDevices, refetchInterval: 30000 })
  const routerAlerts = useQuery({ queryKey: ['network-router-alerts', 'ACTIVE'], queryFn: () => getNetworkRouterAlerts('ACTIVE'), refetchInterval: 30000 })
  const pppoeSummary = useQuery({ queryKey: ['network-pppoe-summary'], queryFn: getNetworkPPPoESummary, refetchInterval: 30000 })
  const billingRun = useMutation({
    mutationFn: runBilling,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['billing-summary'] }),
        queryClient.invalidateQueries({ queryKey: ['billing-runs'] }),
      ])
    },
  })
  const isSuperadmin = getStoredUser()?.role === 'superadmin'
  const selectedView = dashboardViews.find((view) => view.value === dashboardView) ?? dashboardViews[0]
  const cardLinkProps = (path: string) => ({
    role: 'link' as const,
    tabIndex: 0,
    onClick: () => navigate(path),
    onKeyDown: (event: KeyboardEvent) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        navigate(path)
      }
    },
    sx: { cursor: 'pointer', height: '100%', borderRadius: 3, overflow: 'hidden', transition: 'transform 160ms ease, box-shadow 160ms ease, border-color 160ms ease', '&:hover': { transform: 'translateY(-3px)', boxShadow: 8, borderColor: 'primary.main' }, '&:focus-visible': { outline: '3px solid', outlineColor: 'primary.main', outlineOffset: 2 } },
  })

  const stats = [
    {
      title: 'FTP Users',
      value: data?.total_users ?? 0,
      icon: <PeopleIcon />,
      path: '/ftp',
    },
    {
      title: 'Online Users',
      value: data?.online_users ?? 0,
      icon: <CloudIcon />,
      path: '/ftp',
    },
    {
      title: "Today's Logins",
      value: data?.today_logins ?? 0,
      icon: <LoginIcon />,
      path: '/ftp',
    },
    {
      title: "Today's Uploads",
      value: data?.today_uploads ?? 0,
      icon: <UploadIcon />,
      path: '/ftp',
    },
    {
      title: "Today's Downloads",
      value: data?.today_downloads ?? 0,
      icon: <DownloadIcon />,
      path: '/ftp',
    },
    {
      title: 'Upload Traffic',
      value: formatBytes(data?.today_upload_bytes ?? 0),
      icon: <UploadIcon />,
      path: '/ftp',
    },
    {
      title: 'Download Traffic',
      value: formatBytes(data?.today_download_bytes ?? 0),
      icon: <DownloadIcon />,
      path: '/ftp',
    },
  ]
  const networkStats = [
    { label: 'Network Devices', value: networkDevices.data?.length ?? 0, icon: <DnsIcon />, color: 'primary.main', path: '/network/devices' },
    { label: 'Online OLTs', value: networkDevices.data?.filter((device) => device.device_type === 'OLT' && device.monitoring_status === 'ONLINE').length ?? 0, icon: <WifiIcon />, color: 'success.main', path: '/network/devices?type=OLT&status=ONLINE' },
    { label: 'Online Switches', value: networkDevices.data?.filter((device) => device.device_type === 'SWITCH' && device.monitoring_status === 'ONLINE').length ?? 0, icon: <WifiIcon />, color: 'success.main', path: '/network/devices?type=SWITCH&status=ONLINE' },
    { label: 'Offline Devices', value: networkDevices.data?.filter((device) => device.monitoring_enabled && device.monitoring_status === 'OFFLINE').length ?? 0, icon: <WarningAmberIcon />, color: (networkDevices.data?.filter((device) => device.monitoring_enabled && device.monitoring_status === 'OFFLINE').length ?? 0) > 0 ? 'error.main' : 'text.secondary', path: '/network/devices?status=OFFLINE' },
    { label: 'Total Routers', value: routers.data?.length ?? 0, icon: <RouterIcon />, color: 'primary.main', path: '/network/routers' },
    { label: 'Online Routers', value: routers.data?.filter((router) => router.connectivity_status === 'ONLINE').length ?? 0, icon: <WifiIcon />, color: 'success.main', path: '/network/routers' },
    { label: 'Authenticated', value: routers.data?.filter((router) => router.api_status === 'AUTHENTICATED').length ?? 0, icon: <VerifiedUserIcon />, color: 'success.main', path: '/network/routers' },
    { label: 'Active Alerts', value: routerAlerts.data?.length ?? 0, icon: <WarningAmberIcon />, color: (routerAlerts.data?.length ?? 0) > 0 ? 'error.main' : 'text.secondary', path: '/network/routers' },
    { label: 'Active PPPoE Users', value: pppoeSummary.data?.active_sessions ?? 0, icon: <PeopleIcon />, color: 'primary.main', path: '/network/pppoe-sessions' },
    { label: 'Mapped PPPoE Users', value: pppoeSummary.data?.mapped_sessions ?? 0, icon: <VerifiedUserIcon />, color: 'success.main', path: '/network/pppoe-sessions?mapping=mapped' },
    { label: 'Unmapped PPPoE Users', value: pppoeSummary.data?.unmapped_sessions ?? 0, icon: <LinkOffIcon />, color: (pppoeSummary.data?.unmapped_sessions ?? 0) > 0 ? 'warning.main' : 'text.secondary', path: '/network/pppoe-sessions?mapping=unmapped' },
  ]

  return (
    <Box sx={{ maxWidth: 1680, mx: 'auto', pb: 2 }}>
      <Box sx={{ display: 'flex', alignItems: { xs: 'stretch', sm: 'center' }, justifyContent: 'space-between', gap: 2, flexDirection: { xs: 'column', sm: 'row' }, mb: 3 }}>
        <Box><Typography variant="h4" sx={{ fontWeight: 800 }}>Dashboard</Typography><Typography color="text.secondary">{selectedView.description}</Typography></Box>
      </Box>
      {dashboardView !== 'standard' && <Card sx={{ mb: 3, borderRadius: 4, overflow: 'hidden', background: (theme) => `linear-gradient(118deg, ${theme.palette.background.paper} 0%, ${theme.palette.background.paper} 52%, ${theme.palette.primary.main}22 100%)` }}>
        <CardContent sx={{ py: { xs: 2.5, md: 3.5 }, px: { xs: 2.5, md: 4 } }}>
          <Grid container spacing={2} sx={{ alignItems: 'center' }}>
            <Grid size={{ xs: 12, md: 8 }}>
              <Typography variant="overline" sx={{ color: 'primary.main', fontWeight: 800, letterSpacing: 1.4 }}>TS-CLOUD · OPERATIONS CENTER</Typography>
              <Typography variant="h4" sx={{ fontWeight: 800, letterSpacing: '-.03em' }}>ISP billing, network &amp; service overview</Typography>
              <Typography color="text.secondary" sx={{ mt: 0.75 }}>Live billing, RouterOS, OLT and FTP service signals in one workspace.</Typography>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <Box sx={{ display: 'flex', justifyContent: { xs: 'flex-start', md: 'flex-end' }, gap: 1.5, flexWrap: 'wrap' }}>
                <Chip label={`${routers.data?.filter((router) => router.connectivity_status === 'ONLINE').length ?? 0} router online`} color="success" variant="outlined" />
                <Chip label={`${networkDevices.data?.filter((device) => device.monitoring_status === 'ONLINE').length ?? 0} device online`} color="primary" variant="outlined" />
                <Chip label={`${routerAlerts.data?.length ?? 0} active alert${(routerAlerts.data?.length ?? 0) === 1 ? '' : 's'}`} color={(routerAlerts.data?.length ?? 0) > 0 ? 'warning' : 'success'} />
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>}

      {dashboardView === 'architecture' && <Card sx={{ mb: 3, borderRadius: 4, overflow: 'hidden', background: (theme) => `linear-gradient(180deg, ${theme.palette.background.paper}, ${theme.palette.primary.main}0a)` }}>
        <CardContent sx={{ p: { xs: 2, md: 3 } }}>
          <Typography align="center" variant="overline" sx={{ display: 'block', color: 'primary.main', fontWeight: 800, letterSpacing: 1.4 }}>TS-CLOUD ARCHITECTURE</Typography>
          <Grid container spacing={1.5} sx={{ justifyContent: 'center', mt: .5 }}>
            <Grid size={{ xs: 12, sm: 7, md: 4 }}><Box sx={{ p: 1.5, textAlign: 'center', border: '1px solid', borderColor: 'divider', borderRadius: 2.5, bgcolor: 'background.default' }}><Typography sx={{ fontWeight: 800 }}>SUPER ADMIN / HEAD OFFICE</Typography><Typography variant="caption" color="text.secondary">Billing · customer · service control</Typography></Box></Grid>
            <Grid size={{ xs: 12 }}><Box sx={{ width: 2, height: 18, bgcolor: 'divider', mx: 'auto' }} /></Grid>
            <Grid size={{ xs: 12, md: 4 }}><Box sx={{ p: 1.5, borderRadius: 2.5, border: '1px solid', borderColor: 'primary.main', textAlign: 'center' }}><Typography sx={{ fontWeight: 800 }}>ROUTEROS / PPPoE</Typography><Typography variant="caption" color="text.secondary">{routers.data?.filter((router) => router.connectivity_status === 'ONLINE').length ?? 0} online router · {pppoeSummary.data?.active_sessions ?? 0} active session</Typography></Box></Grid>
            <Grid size={{ xs: 12, md: 4 }}><Box sx={{ p: 1.5, borderRadius: 2.5, border: '1px solid', borderColor: 'success.main', textAlign: 'center' }}><Typography sx={{ fontWeight: 800 }}>OLT / ONU LAYER</Typography><Typography variant="caption" color="text.secondary">{networkDevices.data?.filter((device) => device.device_type === 'OLT' && device.monitoring_status === 'ONLINE').length ?? 0} monitored OLT online</Typography></Box></Grid>
            <Grid size={{ xs: 12, md: 4 }}><Box sx={{ p: 1.5, borderRadius: 2.5, border: '1px solid', borderColor: 'secondary.main', textAlign: 'center' }}><Typography sx={{ fontWeight: 800 }}>SERVICE PLATFORM</Typography><Typography variant="caption" color="text.secondary">Internet · FTP · future IPTV / Cloud</Typography></Box></Grid>
            <Grid size={{ xs: 12 }}><Box sx={{ width: 2, height: 18, bgcolor: 'divider', mx: 'auto' }} /></Grid>
            <Grid size={{ xs: 12 }}><Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 1.25 }}><Box sx={{ p: 1.25, textAlign: 'center', borderRadius: 2, bgcolor: 'action.hover' }}><Typography variant="body2" sx={{ fontWeight: 800 }}>CUSTOMERS</Typography><Typography variant="caption" color="text.secondary">Subscriptions &amp; billing</Typography></Box><Box sx={{ p: 1.25, textAlign: 'center', borderRadius: 2, bgcolor: 'action.hover' }}><Typography variant="body2" sx={{ fontWeight: 800 }}>AGENTS / POP</Typography><Typography variant="caption" color="text.secondary">Distribution operations</Typography></Box><Box sx={{ p: 1.25, textAlign: 'center', borderRadius: 2, bgcolor: 'action.hover' }}><Typography variant="body2" sx={{ fontWeight: 800 }}>MONITORING</Typography><Typography variant="caption" color="text.secondary">Alerts &amp; network health</Typography></Box></Box></Grid>
          </Grid>
        </CardContent>
      </Card>}

      {billingRun.isError && <Alert severity="error" sx={{ mb: 2 }}>Billing run failed.</Alert>}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Box><Typography variant="h5" sx={{ fontWeight: 800 }}>Billing Overview</Typography><Typography variant="body2" color="text.secondary">Revenue, collections and invoice health</Typography></Box>
        {isSuperadmin && <Button variant="contained" disabled={billingRun.isPending} onClick={() => billingRun.mutate()}>{billingRun.isPending ? 'Running...' : 'Run Due Billing'}</Button>}
      </Box>
      <Grid container spacing={2} sx={{ mb: 4 }}>
        {[
          ['Total Invoiced', billing.data?.total_invoiced ?? 0, '/invoices'],
          ['Total Collected', billing.data?.total_collected ?? 0, '/payments'],
          ['Outstanding', billing.data?.total_outstanding ?? 0, '/invoices'],
          ["Today's Collection", billing.data?.today_collected ?? 0, '/payments'],
		  ['Voided Payments', billing.data?.voided_amount ?? 0, '/payments'],
        ].map(([label, value, path]) => (
          <Grid key={String(label)} size={{ xs: 12, sm: 6, lg: 3 }}><Card {...cardLinkProps(String(path))}><CardContent sx={{ p: 2.25 }}><Typography variant="body2" color="text.secondary" sx={{ fontWeight: 700 }}>{label}</Typography><Typography variant="h5" sx={{ fontWeight: 800, mt: .75 }}>BDT {Number(value).toLocaleString()}</Typography></CardContent></Card></Grid>
        ))}
		<Grid size={{ xs: 12 }}><Typography color="text.secondary">Overdue invoices: {billing.data?.overdue_invoices ?? 0} · Open invoices: {billing.data?.unpaid_invoices ?? 0} · Cancelled invoices: {billing.data?.cancelled_invoices ?? 0} · Voided payments: {billing.data?.voided_payments ?? 0} · Last billing run: {runs.data?.[0] ? `${runs.data[0].status} (${runs.data[0].created_count} created, ${runs.data[0].failed_count} failed)` : 'Not run yet'}</Typography></Grid>
      </Grid>

      <Box sx={{ mb: 2 }}><Typography variant="h5" sx={{ fontWeight: 800 }}>Network Operations</Typography><Typography variant="body2" color="text.secondary">Real-time router, OLT and PPPoE monitoring</Typography></Box>
      {(networkDevices.isError || routers.isError || routerAlerts.isError || pppoeSummary.isError) && <Alert severity="error" sx={{ mb: 2 }}>Unable to load complete network health.</Alert>}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        {networkStats.map((stat) => (
          <Grid key={stat.label} size={{ xs: 12, sm: 6, lg: 3 }}>
            <Card {...cardLinkProps(stat.path)}><CardContent sx={{ p: 2.25 }}><Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><Box><Typography variant="body2" color="text.secondary" sx={{ fontWeight: 700 }}>{stat.label}</Typography><Typography variant="h5" sx={{ fontWeight: 800, mt: .75 }}>{stat.value}</Typography></Box><Box sx={{ color: stat.color, display: 'grid', placeItems: 'center', width: 42, height: 42, borderRadius: 2.5, bgcolor: 'action.hover' }}>{stat.icon}</Box></Box></CardContent></Card>
          </Grid>
        ))}
      </Grid>
      <Card sx={{ mb: 4, borderRadius: 3 }}><CardContent><Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}><Box><Typography variant="h6" sx={{ fontWeight: 800 }}>Active Network Alerts</Typography><Typography variant="body2" color="text.secondary">Router conditions that need attention</Typography></Box><Chip label={`${routerAlerts.data?.length ?? 0} open`} color={(routerAlerts.data?.length ?? 0) > 0 ? 'warning' : 'success'} /></Box><TableContainer><Table size="small"><TableHead><TableRow><TableCell>Router</TableCell><TableCell>Severity</TableCell><TableCell>Type</TableCell><TableCell>Opened</TableCell><TableCell>Message</TableCell></TableRow></TableHead><TableBody>{routerAlerts.data?.map((alert) => <TableRow key={alert.id} hover onClick={() => navigate('/network/routers')} sx={{ cursor: 'pointer' }}><TableCell>{alert.router_code} — {alert.router_name}</TableCell><TableCell><Chip size="small" color={alert.severity === 'CRITICAL' ? 'error' : 'warning'} label={alert.severity} /></TableCell><TableCell>{alert.type.replaceAll('_', ' ')}</TableCell><TableCell>{new Date(alert.opened_at).toLocaleString()}</TableCell><TableCell>{alert.message}</TableCell></TableRow>)}{!routerAlerts.isLoading && (routerAlerts.data?.length ?? 0) === 0 && <TableRow><TableCell colSpan={5} align="center">No active network alerts.</TableCell></TableRow>}</TableBody></Table></TableContainer></CardContent></Card>

      <Box sx={{ mb: 2 }}><Typography variant="h5" sx={{ fontWeight: 800 }}>Service Activity</Typography><Typography variant="body2" color="text.secondary">Managed FTP usage and activity</Typography></Box>

      {isLoading && (
        <Typography color="text.secondary" sx={{ mb: 3 }}>
          Loading dashboard...
        </Typography>
      )}

      {isError && (
        <Typography color="error" sx={{ mb: 3 }}>
          Unable to load FTP dashboard data.
        </Typography>
      )}

      <Grid container spacing={3}>
        {stats.map((stat) => (
          <Grid
            key={stat.title}
            size={{
              xs: 12,
              sm: 6,
              md: 4,
              lg: 3,
            }}
          >
            <Card {...cardLinkProps(stat.path)}>
              <CardContent>
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    mb: 2,
                  }}
                >
                  <Typography color="text.secondary">
                    {stat.title}
                  </Typography>

                  <Box
                    sx={{
                      color: 'primary.main',
                      display: 'flex',
                    }}
                  >
                    {stat.icon}
                  </Box>
                </Box>

                <Typography
                  variant="h4"
                  sx={{
                    fontWeight: 700,
                  }}
                >
                  {stat.value}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  )
}

function Dashboard() {
  const role = getStoredUser()?.role

  if (role === 'noc') {
    return <OLTDashboard />
  }

  if (role === 'agent') {
    return <AgentDashboard />
  }

  return <AdminDashboard />
}

export default Dashboard
