import { useEffect, useMemo, useState } from 'react'
import {
  Alert, Box, Button, ButtonGroup, Card, CardContent, Chip, CircularProgress,
  Table, TableBody, TableCell, TableHead, TableRow, Typography,
} from '@mui/material'
import { getNetworkDeviceSLA, type NetworkDeviceSLA } from '../../api/networkDeviceSla'

const duration = (seconds: number) =>
  seconds >= 3600
    ? `${(seconds / 3600).toFixed(1)}h`
    : `${Math.round(seconds / 60)}m`

export default function NetworkSLA() {
  const [days, setDays] = useState<1 | 7 | 30>(7)
  const [rows, setRows] = useState<NetworkDeviceSLA[]>([])
  const [deviceType, setDeviceType] = useState<'ALL' | 'OLT' | 'SWITCH' | 'MIKROTIK'>('ALL')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    void getNetworkDeviceSLA(days)
      .then((nextRows) => {
        if (active) setRows(nextRows)
      })
      .catch(() => {
        if (active) setError('SLA report could not be loaded. Please try again.')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [days])

  const selectPeriod = (nextDays: 1 | 7 | 30) => {
    if (nextDays === days) return
    setLoading(true)
    setError('')
    setDays(nextDays)
  }

  const visibleRows = useMemo(
    () => rows.filter((row) => deviceType === 'ALL' || row.device_type === deviceType),
    [deviceType, rows],
  )

  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700 }}>
        Device Uptime / SLA
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        Availability based on recorded monitoring observations.
      </Typography>
      <ButtonGroup sx={{ mb: 2 }}>
        {([1, 7, 30] as const).map((value) => (
          <Button key={value} variant={days === value ? 'contained' : 'outlined'} onClick={() => selectPeriod(value)}>
            {value === 1 ? '24h' : `${value}d`}
          </Button>
        ))}
      </ButtonGroup>
      <ButtonGroup sx={{ mb: 2, ml: { xs: 0, sm: 1 }, display: { xs: 'flex', sm: 'inline-flex' } }}>
        {(['ALL', 'OLT', 'SWITCH', 'MIKROTIK'] as const).map((value) => (
          <Button key={value} variant={deviceType === value ? 'contained' : 'outlined'} onClick={() => setDeviceType(value)}>
            {value === 'ALL' ? 'All devices' : value === 'MIKROTIK' ? 'MikroTik' : value}
          </Button>
        ))}
      </ButtonGroup>
      <Card>
        <CardContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          {loading && <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}><CircularProgress size={28} /></Box>}
          {!loading && !error && rows.length === 0 && <Alert severity="info">No monitoring observations have been recorded for this period yet.</Alert>}
          {!loading && !error && rows.length > 0 && visibleRows.length === 0 && <Alert severity="info">No {deviceType === 'MIKROTIK' ? 'MikroTik' : deviceType.toLowerCase()} device is available for this filter.</Alert>}
          {!loading && !error && visibleRows.length > 0 && (
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Device</TableCell><TableCell>Type</TableCell><TableCell>Status</TableCell>
                  <TableCell align="right">Uptime</TableCell><TableCell align="right">Downtime</TableCell><TableCell align="right">Samples</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {visibleRows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell><b>{row.code}</b><br />{row.name}</TableCell>
                    <TableCell>{row.device_type}</TableCell>
                    <TableCell><Chip size="small" label={row.status} color={row.status === 'ONLINE' ? 'success' : 'error'} /></TableCell>
                    <TableCell align="right">{row.observed_seconds > 0 ? `${row.uptime_percent.toFixed(2)}%` : 'Insufficient data'}</TableCell>
                    <TableCell align="right">{duration(row.downtime_seconds)}</TableCell>
                    <TableCell align="right">{row.samples}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </Box>
  )
}
