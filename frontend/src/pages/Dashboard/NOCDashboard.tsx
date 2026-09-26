import { useEffect, useState } from 'react'
import { useQueries, useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import {
  Autocomplete,
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  MenuItem,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import DnsIcon from '@mui/icons-material/Dns'
import FiberManualRecordIcon from '@mui/icons-material/FiberManualRecord'
import MonitorHeartIcon from '@mui/icons-material/MonitorHeart'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import { Area, AreaChart, Bar, BarChart, CartesianGrid, Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

import { getNetworkDevices } from '../../api/networkDevices'
import {
  getNetworkPPPoESessions,
  getNetworkRouterAlerts,
  getNetworkRouterInterfaces,
  getNetworkRouterPPPoELiveTraffic,
  getNetworkRouterVLANTraffic,
  getNetworkRouters,
  type NetworkRouterPPPoESessionTraffic,
} from '../../api/networkRouters'
import { getOLTDashboard } from '../../api/oltDashboard'
import { getDefaultRouterInterface, saveDefaultRouterInterface } from '../../api/networkRouterTrafficPreference'

const onlineColor = '#2e7d32'
const offlineColor = '#d32f2f'
const neutralColor = '#94a3b8'
const downloadTrafficColor = '#0077B6'
const uploadTrafficColor = '#D14900'
const downloadChartColor = '#00B4D8'
const uploadChartColor = '#FF6B00'
const modules = [
  ['BILLING', 'Invoice · Payment · Due', '/invoices'],
  ['CUSTOMER', 'Customer · Package · Service', '/customers'],
  ['NETWORK', 'MikroTik · PPPoE · VLAN', '/network/routers'],
  ['OLT / ONU', 'OLT · PON · ONU', '/network/olt-dashboard'],
  ['SERVICES', 'FTP · IPTV · Cloud', '/ftp'],
  ['MONITORING', 'Traffic · Alerts · NOC', '/network/devices'],
  ['PORTAL', 'Customer support · Tickets', '/notifications'],
]

function formatBytes(bytes: number) {
  if (!bytes) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** index).toFixed(1)} ${units[index]}`
}

function formatRate(bitsPerSecond: number) {
  if (!bitsPerSecond) return '0 bps'
  const units = ['bps', 'Kbps', 'Mbps', 'Gbps']
  const index = Math.min(Math.floor(Math.log(bitsPerSecond) / Math.log(1000)), units.length - 1)
  return `${(bitsPerSecond / 1000 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function DashboardCard({
  label,
  value,
  detail,
  color,
  onClick,
}: {
  label: string
  value: number | string
  detail: string
  color: string
  onClick: () => void
}) {
  return (
    <Card
      role="link"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onClick()
        }
      }}
      sx={{
        cursor: 'pointer',
        height: '100%',
        borderTop: '4px solid',
        borderColor: color,
        transition: 'transform 160ms ease, box-shadow 160ms ease',
        '&:hover': { transform: 'translateY(-3px)', boxShadow: 7 },
        '&:focus-visible': { outline: '3px solid', outlineColor: color, outlineOffset: 2 },
      }}
    >
      <CardContent>
        <Typography variant="overline" color="text.secondary" sx={{ fontWeight: 800, letterSpacing: 1 }}>{label}</Typography>
        <Typography variant="h3" sx={{ fontWeight: 800, color, lineHeight: 1.1 }}>{value}</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>{detail}</Typography>
      </CardContent>
    </Card>
  )
}

export default function NOCDashboard() {
  const navigate = useNavigate()
  const [darkMode, setDarkMode] = useState(false)
  const [expandedRouterID, setExpandedRouterID] = useState<number | null>(null)
  const [selectedRouterID, setSelectedRouterID] = useState('')
  const [selectedPort, setSelectedPort] = useState('')
  const [manualPortName, setManualPortName] = useState('')
  const [savedDefaults, setSavedDefaults] = useState<Record<string, string>>({})
  const [trafficHistory, setTrafficHistory] = useState<{
    key: string
    samples: { sampledAt: string; time: string; download: number; upload: number }[]
  }>({ key: '', samples: [] })
  const devices = useQuery({ queryKey: ['network-devices'], queryFn: getNetworkDevices, refetchInterval: 30000 })
  const routers = useQuery({ queryKey: ['network-routers'], queryFn: getNetworkRouters, refetchInterval: 30000 })
  const pppoeSessions = useQuery({
    queryKey: ['network-pppoe-sessions', 'noc-capacity'],
    queryFn: () => getNetworkPPPoESessions(true),
    refetchInterval: 30000,
  })
  const activeSessions = pppoeSessions.data ?? []
  const pppoeTrafficRouterIDs = [...new Set(activeSessions.map((session) => session.router_id))]
  const pppoeTrafficQueries = useQueries({
    queries: pppoeTrafficRouterIDs.map((id) => ({
      queryKey: ['network-router-pppoe-live-traffic', id],
      queryFn: () => getNetworkRouterPPPoELiveTraffic(id),
      refetchInterval: 15000,
    })),
  })
  const alerts = useQuery({
    queryKey: ['network-router-alerts', 'ACTIVE'],
    queryFn: () => getNetworkRouterAlerts('ACTIVE'),
    refetchInterval: 30000,
  })
  const oltDashboard = useQuery({ queryKey: ['network-olt-dashboard'], queryFn: getOLTDashboard, refetchInterval: 30000 })

  const routerRows = routers.data ?? []
  const routerID = routerRows.some((router) => String(router.id) === selectedRouterID)
    ? selectedRouterID
    : String(routerRows.find((router) => router.status === 'ACTIVE' && router.credentials_configured)?.id ?? '')
  const savedDefaultPort = savedDefaults[routerID] ?? getDefaultRouterInterface(routerID)

  const interfaces = useQuery({
    queryKey: ['network-router-interfaces', routerID],
    queryFn: () => getNetworkRouterInterfaces(Number(routerID)),
    enabled: Boolean(routerID),
    refetchInterval: 60000,
  })
  const routerPorts = interfaces.data ?? []
  const activePort = manualPortName.trim() ||
    (routerPorts.includes(selectedPort) ? selectedPort : '') ||
    (routerPorts.includes(savedDefaultPort) ? savedDefaultPort : (routerPorts[0] ?? ''))
  const vlanTraffic = useQuery({
    queryKey: ['network-router-vlan-traffic', routerID, activePort],
    queryFn: () => getNetworkRouterVLANTraffic(Number(routerID), activePort || undefined),
    enabled: Boolean(routerID && activePort),
    refetchInterval: 5000,
  })
  const selectedInterfaceTraffic = (vlanTraffic.data?.interfaces ?? []).find(
    (iface) => iface.name === activePort,
  )
  const selectedPortVlans = (vlanTraffic.data?.vlans ?? []).filter(
    (vlan) => vlan.parent_interface === activePort,
  )
  const trafficHistoryKey = `${routerID}:${activePort}`
  useEffect(() => {
    if (!selectedInterfaceTraffic || !vlanTraffic.data?.sampled_at) return
    const sampledAt = vlanTraffic.data.sampled_at
    const timer = window.setTimeout(() => {
      setTrafficHistory((current) => {
        const samples = current.key === trafficHistoryKey ? current.samples : []
        if (samples.at(-1)?.sampledAt === sampledAt) return current
        return {
          key: trafficHistoryKey,
          samples: [
            ...samples.slice(-29),
            {
              sampledAt,
              time: new Date(sampledAt).toLocaleTimeString(),
              download: selectedInterfaceTraffic.rx_bps / 1_000_000,
              upload: selectedInterfaceTraffic.tx_bps / 1_000_000,
            },
          ],
        }
      })
    }, 0)
    return () => window.clearTimeout(timer)
  }, [selectedInterfaceTraffic, trafficHistoryKey, vlanTraffic.data?.sampled_at])
  const trafficSamples = trafficHistory.key === trafficHistoryKey ? trafficHistory.samples : []

  const rows = devices.data ?? []
  const monitored = rows.filter((device) => device.monitoring_enabled)
  const online = monitored.filter((device) => device.monitoring_status === 'ONLINE')
  const offline = monitored.filter((device) => device.monitoring_status === 'OFFLINE')
  const availability = monitored.length ? Math.round((online.length / monitored.length) * 100) : 0
  const downloadRate = vlanTraffic.isLoading
    ? 'Sampling…'
    : vlanTraffic.isError
      ? 'Unavailable'
      : selectedInterfaceTraffic
        ? formatRate(selectedInterfaceTraffic.rx_bps)
        : '—'
  const uploadRate = vlanTraffic.isLoading
    ? 'Sampling…'
    : vlanTraffic.isError
      ? 'Unavailable'
      : selectedInterfaceTraffic
        ? formatRate(selectedInterfaceTraffic.tx_bps)
        : '—'
  const reachableRouters = routerRows.filter((router) => router.connectivity_status === 'ONLINE')
  const averageCPU = reachableRouters.length
    ? Math.round(reachableRouters.reduce((total, router) => total + router.cpu_load, 0) / reachableRouters.length)
    : 0
  const totalMemory = reachableRouters.reduce((total, router) => total + router.total_memory, 0)
  const memoryUsed = reachableRouters.reduce(
    (total, router) => total + Math.max(0, router.total_memory - router.free_memory),
    0,
  )
  const memoryPercent = totalMemory ? Math.round(memoryUsed * 100 / totalMemory) : 0
  const deviceHealth = [
    { name: 'Online', value: online.length, color: onlineColor },
    { name: 'Offline', value: offline.length, color: offlineColor },
    { name: 'Unknown', value: Math.max(0, monitored.length - online.length - offline.length), color: neutralColor },
  ].filter((item) => item.value > 0)
  const devicesByType = ['MIKROTIK', 'SWITCH', 'OLT'].map((type) => {
    const group = type === 'MIKROTIK'
      ? routerRows.map((router) => ({ monitoring_status: router.connectivity_status }))
      : monitored.filter((device) => device.device_type === type)
    return {
      type: type === 'MIKROTIK' ? 'Router' : type,
      online: group.filter((device) => device.monitoring_status === 'ONLINE').length,
      offline: group.filter((device) => device.monitoring_status === 'OFFLINE').length,
    }
  })
  const liveRatesBySession = new Map<string, NetworkRouterPPPoESessionTraffic>()
  pppoeTrafficQueries.forEach((query, index) => {
    for (const sample of query.data?.traffic ?? []) {
      liveRatesBySession.set(`${pppoeTrafficRouterIDs[index]}:${sample.username.toLowerCase()}`, sample)
    }
  })
  const topUsers = activeSessions.map((session) => {
    const liveRate = liveRatesBySession.get(`${session.router_id}:${session.username.toLowerCase()}`)
    return {
      ...session,
      liveDownloadBps: liveRate?.download_bps ?? session.rx_rate_bps,
      liveUploadBps: liveRate?.upload_bps ?? session.tx_rate_bps,
      rateSource: liveRate?.source ?? 'session-snapshot',
    }
  })
    .sort((left, right) => right.liveDownloadBps + right.liveUploadBps - left.liveDownloadBps - left.liveUploadBps)
    .slice(0, 5)
  const pppoeTrafficLoading = pppoeTrafficQueries.some((query) => query.isLoading)
  const pppoeTrafficError = pppoeTrafficQueries.some((query) => query.isError)
  const hasFallbackTraffic = pppoeTrafficError ||
    topUsers.some((session) => session.rateSource === 'ppp-active-rate' || session.rateSource === 'session-snapshot')
  const priorityDevices = [...monitored]
    .filter((device) => device.device_type !== 'MIKROTIK')
    .sort((left, right) => {
      const typePriority = (device: typeof left) => device.device_type === 'SWITCH' ? 0 : device.device_type === 'OLT' ? 1 : 2
      return typePriority(left) - typePriority(right) ||
        Number(right.monitoring_status === 'OFFLINE') - Number(left.monitoring_status === 'OFFLINE') ||
        left.code.localeCompare(right.code, undefined, { numeric: true })
    })

  if (devices.isLoading) {
    return <Box sx={{ display: 'grid', placeItems: 'center', minHeight: 320 }}><CircularProgress aria-label="Loading network dashboard" /></Box>
  }
  if (devices.isError) return <Alert severity="error">Unable to load the network monitoring dashboard.</Alert>

  return (
    <Box sx={{
      maxWidth: 1680,
      mx: 'auto',
      pb: 3,
      p: { xs: 1, md: 2 },
      borderRadius: 3,
      color: darkMode ? '#e7eefc' : 'text.primary',
      background: darkMode ? 'radial-gradient(circle at 50% -20%, #113c70 0%, #071321 42%, #030914 100%)' : 'transparent',
      '& .MuiCard-root': darkMode ? {
        bgcolor: 'rgba(9, 24, 43, 0.9)',
        color: '#e7eefc',
        border: '1px solid rgba(100, 166, 235, 0.16)',
        boxShadow: 'none',
      } : {},
    }}>
      <Box sx={{ textAlign: 'center', mb: 2.5 }}>
        <Typography variant="h3" sx={{ fontWeight: 900, letterSpacing: 2 }}>
          TS-<Box component="span" sx={{ color: '#35a7ff' }}>CLOUD</Box>
        </Typography>
        <Typography color="text.secondary">Unified ISP Management &amp; Automation Platform</Typography>
      </Box>
      <Grid container spacing={1.25} sx={{ mb: 2.5 }}>
        {modules.map(([title, detail, path]) => (
          <Grid key={title} size={{ xs: 6, sm: 4, lg: 1 }}>
            <Box
              role="link"
              tabIndex={0}
              onClick={() => navigate(path)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  navigate(path)
                }
              }}
              sx={{
                minHeight: 72,
                p: 1.25,
                borderRadius: 2,
                border: '1px solid rgba(100, 166, 235, 0.18)',
                bgcolor: darkMode ? 'rgba(12, 31, 55, 0.88)' : 'background.paper',
                cursor: 'pointer',
                '&:hover': { borderColor: '#35a7ff', transform: 'translateY(-2px)' },
              }}
            >
              <Typography sx={{ fontSize: 12, fontWeight: 900 }}>{title}</Typography>
              <Typography variant="caption" color="text.secondary">{detail}</Typography>
            </Box>
          </Grid>
        ))}
      </Grid>
      <Box sx={{ mb: 3, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap' }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 900, color: '#35a7ff' }}>NOC COMMAND CENTER</Typography>
          <Typography color="text.secondary">Real-time health of monitored routers, OLTs, switches and network devices.</Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Button size="small" onClick={() => navigate('/dashboard')}>Back to Admin Dashboard</Button>
          <Typography variant="caption">Dark</Typography>
          <Switch checked={darkMode} onChange={(event) => setDarkMode(event.target.checked)} />
          <Chip size="small" label="LIVE · refreshes every 30s" color="success" variant="outlined" />
        </Box>
      </Box>
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6, lg: 3 }}><DashboardCard label="Monitored devices" value={monitored.length} detail="Under active monitoring" color="primary.main" onClick={() => navigate('/network/devices')} /></Grid>
        <Grid size={{ xs: 12, sm: 6, lg: 3 }}><DashboardCard label="Online devices" value={online.length} detail="Reachable in latest poll" color={onlineColor} onClick={() => navigate('/network/devices?status=ONLINE')} /></Grid>
        <Grid size={{ xs: 12, sm: 6, lg: 3 }}><DashboardCard label="Problem devices" value={offline.length} detail="Needs attention" color={offline.length ? offlineColor : onlineColor} onClick={() => navigate('/network/devices?status=OFFLINE')} /></Grid>
        <Grid size={{ xs: 12, sm: 6, lg: 3 }}><DashboardCard label="Network availability" value={`${availability}%`} detail="Online share of monitored devices" color={offline.length ? offlineColor : onlineColor} onClick={() => navigate('/network/olt-dashboard')} /></Grid>
      </Grid>
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Typography align="center" sx={{ fontWeight: 900, color: '#35a7ff', letterSpacing: 1 }}>TS-CLOUD LIVE TOPOLOGY</Typography>
          <Grid container spacing={1.5} sx={{ mt: 1 }}>
            {[
              ['NOC CONTROL', 'Live monitoring', '/dashboard?view=noc'],
              ['ROUTERS', `${routerRows.length} MikroTik`, '/network/routers'],
              ['OLT / ONU', `${oltDashboard.data?.summary.total_olts ?? 0} OLT · ${oltDashboard.data?.summary.total_onus ?? 0} ONU`, '/network/olt-dashboard'],
              ['SWITCHES', `${monitored.filter((device) => device.device_type === 'SWITCH').length} monitored`, '/network/devices?type=SWITCH'],
              ['MONITORING', `${monitored.length} monitored devices`, '/network/devices'],
            ].map(([title, detail, path]) => (
              <Grid key={title} size={{ xs: 6, sm: 4, md: 2.4 }}>
                <Button fullWidth variant="outlined" onClick={() => navigate(path)} sx={{ height: '100%', minHeight: 70, flexDirection: 'column' }}>
                  <Typography variant="caption" sx={{ fontWeight: 900 }}>{title}</Typography>
                  <Typography variant="caption" color="text.secondary">{detail}</Typography>
                </Button>
              </Grid>
            ))}
          </Grid>
        </CardContent>
      </Card>
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
            <Typography sx={{ fontWeight: 900 }}>ACTIVE ALERTS</Typography>
            <Chip size="small" label={`${offline.length + (alerts.data?.length ?? 0)} active`} color={offline.length || (alerts.data?.length ?? 0) ? 'error' : 'success'} />
          </Box>
          {[...offline.map((device) => ({
            key: `device-${device.id}`,
            label: `${device.code} · OFFLINE`,
            detail: device.last_error || device.name,
            path: `/network/devices?type=${device.device_type}&status=OFFLINE`,
          })), ...(alerts.data ?? []).map((alert) => ({
            key: `router-${alert.id}`,
            label: `${alert.router_code} · ${alert.type.replaceAll('_', ' ')}`,
            detail: alert.message,
            path: '/network/routers',
          }))].slice(0, 8).map((alert) => (
            <Box key={alert.key} onClick={() => navigate(alert.path)} sx={{ py: 1, borderTop: '1px solid', borderColor: 'divider', cursor: 'pointer' }}>
              <Typography variant="body2" sx={{ fontWeight: 700, color: 'error.main' }}>{alert.label}</Typography>
              <Typography variant="caption" color="text.secondary">{alert.detail}</Typography>
            </Box>
          ))}
          {offline.length === 0 && (alerts.data?.length ?? 0) === 0 && <Typography color="text.secondary">No active network alerts.</Typography>}
        </CardContent>
      </Card>
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 1, flexWrap: 'wrap', mb: 1.5 }}>
            <Box><Typography sx={{ fontWeight: 900 }}>LIVE TRAFFIC &amp; CAPACITY</Typography><Typography variant="caption" color="text.secondary">Live selected-interface throughput (RX download / TX upload) and router resources.</Typography></Box>
            <Chip size="small" label={`${activeSessions.length} active PPPoE`} color="primary" onClick={() => navigate('/network/pppoe-sessions')} />
          </Box>
          <Grid container spacing={1.5} sx={{ mb: 2 }}>
            <Grid size={{ xs: 6, md: 3 }}><Card variant="outlined" sx={{ borderLeft: `5px solid ${downloadTrafficColor}`, bgcolor: 'rgba(0, 180, 216, 0.08)' }}><CardContent><Typography variant="caption" sx={{ color: downloadTrafficColor, fontWeight: 900, letterSpacing: 0.8 }}>DOWNLOAD · RX</Typography><Typography variant="h5" sx={{ fontWeight: 900, color: downloadTrafficColor }}>{downloadRate}</Typography></CardContent></Card></Grid>
            <Grid size={{ xs: 6, md: 3 }}><Card variant="outlined" sx={{ borderLeft: `5px solid ${uploadTrafficColor}`, bgcolor: 'rgba(255, 107, 0, 0.08)' }}><CardContent><Typography variant="caption" sx={{ color: uploadTrafficColor, fontWeight: 900, letterSpacing: 0.8 }}>UPLOAD · TX</Typography><Typography variant="h5" sx={{ fontWeight: 900, color: uploadTrafficColor }}>{uploadRate}</Typography></CardContent></Card></Grid>
            <Grid size={{ xs: 6, md: 3 }}><Card variant="outlined"><CardContent><Typography variant="caption" color="text.secondary">AVG ROUTER CPU</Typography><Typography sx={{ fontWeight: 900, color: averageCPU >= 85 ? offlineColor : '#e6b800' }}>{averageCPU}%</Typography></CardContent></Card></Grid>
            <Grid size={{ xs: 6, md: 3 }}><Card variant="outlined"><CardContent><Typography variant="caption" color="text.secondary">ROUTER RAM USED</Typography><Typography sx={{ fontWeight: 900, color: memoryPercent >= 85 ? offlineColor : '#ba68c8' }}>{memoryPercent}%</Typography></CardContent></Card></Grid>
          </Grid>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1, mt: 1 }}>
            <Typography variant="h6" sx={{ fontWeight: 900, letterSpacing: 0.7 }}>LIVE TRAFFIC</Typography>
            {activePort && <Chip size="small" variant="outlined" label={`Interface: ${activePort}`} />}
          </Box>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
            Updates every 5 seconds · showing the latest 30 live samples.
          </Typography>
          <Box sx={{ width: '100%', height: 300 }}>
            {trafficSamples.length ? (
              <ResponsiveContainer>
                <AreaChart data={trafficSamples} margin={{ top: 12, right: 20, left: 8, bottom: 4 }}>
                  <defs>
                    <linearGradient id="noc-download-gradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={downloadChartColor} stopOpacity={0.42} />
                      <stop offset="95%" stopColor={downloadChartColor} stopOpacity={0.03} />
                    </linearGradient>
                    <linearGradient id="noc-upload-gradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={uploadChartColor} stopOpacity={0.38} />
                      <stop offset="95%" stopColor={uploadChartColor} stopOpacity={0.03} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" interval="preserveStartEnd" minTickGap={36} tick={{ fontSize: 11 }} />
                  <YAxis allowDecimals tick={{ fontSize: 12, fontWeight: 700 }} tickFormatter={(value: number) => `${value} Mbps`} />
                  <Tooltip formatter={(value) => `${Number(value ?? 0).toFixed(2)} Mbps`} />
                  <Legend verticalAlign="top" height={36} wrapperStyle={{ fontSize: 15, fontWeight: 800 }} />
                  <Area type="monotone" dataKey="download" name="Download · RX" stroke={downloadChartColor} strokeWidth={3} fill="url(#noc-download-gradient)" activeDot={{ r: 5 }} />
                  <Area type="monotone" dataKey="upload" name="Upload · TX" stroke={uploadChartColor} strokeWidth={3} fill="url(#noc-upload-gradient)" activeDot={{ r: 5 }} />
                </AreaChart>
              </ResponsiveContainer>
            ) : <Box sx={{ display: 'grid', placeItems: 'center', height: '100%' }}><Typography color="text.secondary">{vlanTraffic.isError ? 'Interface traffic is unavailable; check the report below for details.' : 'Collecting live traffic samples…'}</Typography></Box>}
          </Box>
        </CardContent>
      </Card>
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap', mb: 1.5 }}>
            <Box><Typography sx={{ fontWeight: 900 }}>INTERFACE TRAFFIC REPORT</Typography><Typography variant="caption" color="text.secondary">Live interface RX/TX, with VLAN counters when configured.</Typography></Box>
            <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
              <TextField select size="small" label="Router" value={routerID} onChange={(event) => { setSelectedRouterID(event.target.value); setSelectedPort(''); setManualPortName('') }} sx={{ minWidth: 200 }}>
                {routerRows.map((router) => <MenuItem key={router.id} value={String(router.id)}>{router.code} — {router.name}</MenuItem>)}
              </TextField>
              <Autocomplete
                freeSolo
                options={routerPorts}
                value={activePort}
                inputValue={manualPortName || activePort}
                onChange={(_, value) => {
                  const port = typeof value === 'string' ? value : ''
                  setSelectedPort(port)
                  setManualPortName(port)
                }}
                onInputChange={(_, value, reason) => {
                  if (reason === 'input' || reason === 'clear') setManualPortName(value)
                }}
                disabled={!routerID}
                sx={{ minWidth: 210 }}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    size="small"
                    label="MikroTik port / interface"
                    placeholder="Type or select, e.g. ether1"
                    helperText={interfaces.isError
                      ? 'Interface list unavailable; enter the exact RouterOS name.'
                      : 'Select a detected interface or enter its exact RouterOS name.'}
                    slotProps={{ ...params.slotProps, htmlInput: { ...params.slotProps.htmlInput, maxLength: 120 } }}
                  />
                )}
              />
              <Button
                variant={savedDefaultPort === activePort ? 'outlined' : 'contained'}
                disabled={!activePort || savedDefaultPort === activePort}
                onClick={() => {
                  saveDefaultRouterInterface(routerID, activePort)
                  setSavedDefaults((current) => ({ ...current, [routerID]: activePort }))
                }}
              >
                {savedDefaultPort === activePort ? 'Default saved' : 'Save as default'}
              </Button>
            </Box>
          </Box>
          {interfaces.isError && <Alert severity="warning" sx={{ mb: 1 }}>Port list unavailable; type the exact RouterOS interface name above.</Alert>}
          {vlanTraffic.isLoading ? <Box sx={{ display: 'grid', placeItems: 'center', minHeight: 90 }}><CircularProgress size={24} aria-label="Loading interface traffic" /></Box>
            : vlanTraffic.isError ? <Alert severity="error">Unable to load interface traffic. Check RouterOS API connectivity and interface monitoring permissions.</Alert>
              : selectedInterfaceTraffic ? <>
                <Typography variant="caption" color="text.secondary">Sampled {new Date(vlanTraffic.data!.sampled_at).toLocaleString()} · auto-refreshes every 30 seconds</Typography>
                <Table size="small">
                  <TableHead><TableRow><TableCell>Interface</TableCell><TableCell align="right">RX</TableCell><TableCell align="right">TX</TableCell></TableRow></TableHead>
                  <TableBody><TableRow><TableCell>{selectedInterfaceTraffic.name}</TableCell><TableCell align="right">{formatRate(selectedInterfaceTraffic.rx_bps)}</TableCell><TableCell align="right">{formatRate(selectedInterfaceTraffic.tx_bps)}</TableCell></TableRow></TableBody>
                </Table>
                {selectedPortVlans.length > 0
                  ? <Table size="small" sx={{ mt: 1.5 }}>
                    <TableHead><TableRow><TableCell>VLAN ID</TableCell><TableCell>VLAN interface</TableCell><TableCell align="right">RX</TableCell><TableCell align="right">TX</TableCell></TableRow></TableHead>
                    <TableBody>{selectedPortVlans.map((vlan) => <TableRow key={`${vlan.parent_interface}-${vlan.vlan_id}-${vlan.name}`}><TableCell>{vlan.vlan_id}</TableCell><TableCell>{vlan.name}</TableCell><TableCell align="right">{formatRate(vlan.rx_bps)}</TableCell><TableCell align="right">{formatRate(vlan.tx_bps)}</TableCell></TableRow>)}</TableBody>
                  </Table>
                  : <Alert severity="info" sx={{ mt: 1.5 }}>{activePort} has no configured VLAN interfaces; interface traffic is shown above.</Alert>}
              </> : <Alert severity="info">{routerID ? 'No interface traffic sample is available for this port.' : 'No MikroTik router is configured.'}</Alert>}
        </CardContent>
      </Card>
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 5 }}>
          <Card sx={{ height: '100%' }}><CardContent>
            <Typography variant="h6" sx={{ fontWeight: 800 }}>Device health</Typography>
            <Typography variant="body2" color="text.secondary">Live status distribution</Typography>
            <Box sx={{ height: 290, position: 'relative' }}>
              <ResponsiveContainer>
                <PieChart>
                  <Pie data={deviceHealth} dataKey="value" nameKey="name" innerRadius="58%" outerRadius="82%" paddingAngle={3}>
                    {deviceHealth.map((entry) => <Cell key={entry.name} fill={entry.color} />)}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
              <Box sx={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', pointerEvents: 'none' }}>
                <Box sx={{ textAlign: 'center' }}><Typography variant="h4" sx={{ fontWeight: 800 }}>{monitored.length}</Typography><Typography variant="caption" color="text.secondary">devices</Typography></Box>
              </Box>
            </Box>
            <Box sx={{ display: 'flex', justifyContent: 'center', gap: 1, flexWrap: 'wrap' }}>
              {deviceHealth.map((item) => <Chip key={item.name} size="small" icon={<FiberManualRecordIcon sx={{ color: `${item.color} !important` }} />} label={`${item.name}: ${item.value}`} />)}
            </Box>
          </CardContent></Card>
        </Grid>
        <Grid size={{ xs: 12, md: 7 }}>
          <Card sx={{ height: '100%' }}><CardContent>
            <Typography variant="h6" sx={{ fontWeight: 800 }}>Devices by type</Typography>
            <Typography variant="body2" color="text.secondary">Online and offline Router, Switch and OLT counts</Typography>
            <Box sx={{ height: 340 }}>
              <ResponsiveContainer>
                <BarChart data={devicesByType}>
                  <XAxis dataKey="type" /><YAxis allowDecimals={false} /><Tooltip />
                  <Bar dataKey="online" name="Online" stackId="status" fill={onlineColor} />
                  <Bar dataKey="offline" name="Offline" stackId="status" fill={offlineColor} />
                </BarChart>
              </ResponsiveContainer>
            </Box>
          </CardContent></Card>
        </Grid>
        <Grid size={{ xs: 12 }}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
              <Box><Typography variant="h6" sx={{ fontWeight: 800 }}>Router resources</Typography><Typography variant="body2" color="text.secondary">CPU and RAM at a glance. Select a router card to expand details.</Typography></Box>
              <MonitorHeartIcon color="primary" />
            </Box>
            <Grid container spacing={1.5}>
              {routerRows.map((router) => {
                const ramUsed = router.total_memory > 0
                  ? Math.round((router.total_memory - router.free_memory) * 100 / router.total_memory)
                  : 0
                const expanded = expandedRouterID === router.id
                return (
                  <Grid key={router.id} size={{ xs: 12, sm: 6, lg: 4 }}>
                    <Card variant="outlined" onClick={() => setExpandedRouterID(expanded ? null : router.id)} sx={{ cursor: 'pointer' }}>
                      <CardContent>
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1 }}>
                          <Box><Typography sx={{ fontWeight: 800 }}>{router.code}</Typography><Typography variant="caption" color="text.secondary">{router.name || router.router_identity || 'MikroTik router'}</Typography></Box>
                          <Chip size="small" color={router.connectivity_status === 'ONLINE' ? 'success' : 'error'} label={router.connectivity_status || 'UNKNOWN'} />
                        </Box>
                        <Grid container spacing={1} sx={{ mt: 1 }}>
                          <Grid size={6}><Typography variant="caption" color="text.secondary">CPU</Typography><Typography sx={{ fontWeight: 800 }}>{router.cpu_load}%</Typography></Grid>
                          <Grid size={6}><Typography variant="caption" color="text.secondary">RAM used</Typography><Typography sx={{ fontWeight: 800 }}>{ramUsed}%</Typography></Grid>
                        </Grid>
                        {expanded && <Box sx={{ mt: 1, pt: 1, borderTop: '1px solid', borderColor: 'divider' }}>
                          <Typography variant="body2">Uptime: {router.router_uptime || '—'}</Typography>
                          <Typography variant="body2">Memory: {formatBytes(router.free_memory)} free / {formatBytes(router.total_memory)}</Typography>
                          <Typography variant="body2">RouterOS: {router.routeros_version || 'Unavailable'}</Typography>
                        </Box>}
                      </CardContent>
                    </Card>
                  </Grid>
                )
              })}
              {!routers.isLoading && routerRows.length === 0 && <Grid size={{ xs: 12 }}><Typography color="text.secondary" align="center">No router resource data is available.</Typography></Grid>}
            </Grid>
          </CardContent></Card>
        </Grid>
        <Grid size={{ xs: 12 }}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}><Box><Typography variant="h6" sx={{ fontWeight: 800 }}>Priority devices</Typography><Typography variant="body2" color="text.secondary">Switches and OLTs, with offline devices shown first.</Typography></Box><DnsIcon color="primary" /></Box>
            <Grid container spacing={1.5}>
              {priorityDevices.slice(0, 12).map((device) => {
                const isOffline = device.monitoring_status === 'OFFLINE'
                const devicePath = `/network/devices?type=${device.device_type}&device=${device.id}${isOffline ? '&status=OFFLINE' : ''}`
                return (
                  <Grid key={device.id} size={{ xs: 12, sm: 6, lg: 3 }}>
                    <Box
                      role="link"
                      tabIndex={0}
                      onClick={() => navigate(devicePath)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          navigate(devicePath)
                        }
                      }}
                      sx={{
                        p: 1.5,
                        borderRadius: 2,
                        border: '1px solid',
                        borderColor: isOffline ? 'error.light' : 'success.light',
                        bgcolor: isOffline ? 'error.50' : 'success.50',
                        cursor: 'pointer',
                        transition: 'transform 160ms ease, box-shadow 160ms ease',
                        '&:hover': { transform: 'translateY(-2px)', boxShadow: 3 },
                        '&:focus-visible': { outline: '3px solid', outlineColor: isOffline ? 'error.main' : 'success.main', outlineOffset: 2 },
                      }}
                    >
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1 }}>
                        <Box sx={{ minWidth: 0 }}>
                          <Typography sx={{ fontWeight: 800 }} noWrap>{device.code}</Typography>
                          <Typography variant="caption" color="text.secondary" noWrap>
                            {device.device_type} · {device.name}
                          </Typography>
                        </Box>
                        {isOffline
                          ? <WarningAmberIcon color="error" />
                          : <DnsIcon color={device.monitoring_status === 'ONLINE' ? 'success' : 'disabled'} />}
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 1, mt: 1 }}>
                        <Chip
                          size="small"
                          color={device.monitoring_status === 'ONLINE' ? 'success' : isOffline ? 'error' : 'default'}
                          label={device.monitoring_status || 'UNKNOWN'}
                        />
                        <Typography variant="caption" color="text.secondary" noWrap>
                          {device.pop_name || 'Unassigned'}
                        </Typography>
                      </Box>
                      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.75 }}>
                        Last checked: {device.last_polled_at ? new Date(device.last_polled_at).toLocaleString() : 'Never'}
                      </Typography>
                    </Box>
                  </Grid>
                )
              })}
              {!priorityDevices.length && (
                <Grid size={{ xs: 12 }}>
                  <Typography color="text.secondary" align="center">No monitored Switch or OLT devices.</Typography>
                </Grid>
              )}
            </Grid>
          </CardContent></Card>
        </Grid>
        <Grid size={{ xs: 12 }}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}><Box><Typography variant="h6" sx={{ fontWeight: 800 }}>Top bandwidth users</Typography><Typography variant="body2" color="text.secondary">{hasFallbackTraffic ? 'Live interface counters unavailable; showing latest PPPoE session rates where available.' : 'Live per-user RouterOS traffic · refreshed every 15 seconds.'}</Typography></Box><Button size="small" onClick={() => navigate('/network/pppoe-sessions')}>View all</Button></Box>
            {pppoeTrafficError && <Alert severity="warning" sx={{ mb: 1 }}>Live RouterOS interface sampling failed for at least one router. User list remains available using the latest session rates.</Alert>}
            {topUsers.length ? topUsers.map((session, index) => <Box key={session.id} sx={{ display: 'grid', gridTemplateColumns: '32px minmax(0, 1fr) auto auto', gap: 1.5, alignItems: 'center', py: 1, borderTop: '1px solid', borderColor: 'divider' }}><Typography sx={{ fontWeight: 900, color: '#35a7ff' }}>{index + 1}</Typography><Box sx={{ minWidth: 0 }}><Typography sx={{ fontWeight: 700 }} noWrap>{session.username}</Typography><Typography variant="caption" color="text.secondary" noWrap>{session.router_code}</Typography></Box><Typography sx={{ color: downloadTrafficColor, fontWeight: 800, whiteSpace: 'nowrap' }}>↓ {formatRate(session.liveDownloadBps)}</Typography><Typography sx={{ color: uploadTrafficColor, fontWeight: 800, whiteSpace: 'nowrap' }}>↑ {formatRate(session.liveUploadBps)}</Typography></Box>) : <Typography color={pppoeTrafficError ? 'error.main' : 'text.secondary'}>{pppoeTrafficError ? 'Could not load active PPPoE sessions.' : pppoeTrafficLoading ? 'Sampling live PPPoE traffic from RouterOS…' : 'No active PPPoE users are currently listed.'}</Typography>}
          </CardContent></Card>
        </Grid>
      </Grid>
    </Box>
  )
}
