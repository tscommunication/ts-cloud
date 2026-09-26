import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  Autocomplete,
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  IconButton,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import DeleteIcon from '@mui/icons-material/Delete'
import EditIcon from '@mui/icons-material/Edit'
import VisibilityIcon from '@mui/icons-material/Visibility'

import { getNetworkDevices, getNetworkDevicePorts } from '../../api/networkDevices'
import {
  createNetworkVLANEntry,
  deleteNetworkVLANEntry,
  getNetworkRouterInterfaces,
  getNetworkRouterVLANTraffic,
  getNetworkRouters,
  getNetworkVLANEntries,
  updateNetworkVLANEntry,
  type NetworkVLANEntry,
  type NetworkVLANEntryInput,
} from '../../api/networkRouters'
import { getAPIErrorMessage } from '../../api/errors'
import { getDefaultRouterInterface, saveDefaultRouterInterface } from '../../api/networkRouterTrafficPreference'

type DeviceKind = 'router' | 'switch' | 'olt'

function formatRate(bitsPerSecond: number) {
  if (bitsPerSecond < 1000) return `${bitsPerSecond} bps`
  const units = ['Kbps', 'Mbps', 'Gbps']
  const unitIndex = Math.min(Math.floor(Math.log10(bitsPerSecond) / 3) - 1, units.length - 1)
  return `${(bitsPerSecond / 1000 ** (unitIndex + 1)).toFixed(1)} ${units[unitIndex]}`
}

export default function NetworkVLANTraffic() {
  const [deviceKind, setDeviceKind] = useState<DeviceKind>('router')
  const [deviceSelection, setDeviceSelection] = useState('')
  const [portSelection, setPortSelection] = useState('')
  const [manualPortName, setManualPortName] = useState('')
  const [savedDefaults, setSavedDefaults] = useState<Record<string, string>>({})
  const [vlanIDInput, setVLANIDInput] = useState('')
  const [vlanNameInput, setVLANNameInput] = useState('')
  const [editingEntryID, setEditingEntryID] = useState<number | null>(null)
  const [selectedVLANEntryID, setSelectedVLANEntryID] = useState<number | null>(null)
  const queryClient = useQueryClient()
  const routers = useQuery({
    queryKey: ['network-routers'],
    queryFn: getNetworkRouters,
    refetchInterval: 30000,
  })
  const devices = useQuery({
    queryKey: ['network-devices', 'vlan-report'],
    queryFn: getNetworkDevices,
    refetchInterval: 30000,
  })
  const availableRouters = (routers.data ?? []).filter(
    (router) => router.status === 'ACTIVE' && router.credentials_configured,
  )
  const availableDevices = (devices.data ?? []).filter(
    (device) => device.device_type === (deviceKind === 'switch' ? 'SWITCH' : 'OLT') &&
      device.monitoring_enabled,
  )
  const selectedDevices = deviceKind === 'router' ? availableRouters : availableDevices
  const deviceID = selectedDevices.some((device) => String(device.id) === deviceSelection)
    ? deviceSelection
    : String(selectedDevices[0]?.id ?? '')

  const savedDefaultPort = deviceKind === 'router'
    ? savedDefaults[deviceID] ?? getDefaultRouterInterface(deviceID)
    : ''
  const routerInterfaces = useQuery({
    queryKey: ['network-router-interfaces', deviceID],
    queryFn: () => getNetworkRouterInterfaces(Number(deviceID)),
    enabled: deviceKind === 'router' && Boolean(deviceID),
    refetchInterval: 60000,
  })
  const routerPorts = routerInterfaces.data ?? []
  const selectedPort = manualPortName.trim() ||
    (routerPorts.includes(portSelection) ? portSelection : '') ||
    (routerPorts.includes(savedDefaultPort) ? savedDefaultPort : (routerPorts[0] ?? ''))

  const vlanReport = useQuery({
    queryKey: ['network-router-vlan-traffic', deviceID, selectedPort],
    queryFn: () => getNetworkRouterVLANTraffic(
      Number(deviceID),
      selectedPort || undefined,
    ),
    enabled: deviceKind === 'router' && Boolean(deviceID && selectedPort),
    refetchInterval: 30000,
  })
  const switchPorts = useQuery({
    queryKey: ['network-device-ports', 'vlan-report', deviceKind, deviceID],
    queryFn: () => getNetworkDevicePorts(Number(deviceID)),
    enabled: deviceKind !== 'router' && Boolean(deviceID),
    refetchInterval: 30000,
  })

  const switchPortRows = switchPorts.data ?? []
  const activePort = deviceKind === 'router'
    ? selectedPort
    : (switchPortRows.some((port) => String(port.id) === portSelection)
      ? portSelection
      : String(switchPortRows[0]?.id ?? ''))
  const selectedSwitchPort = switchPortRows.find((port) => String(port.id) === activePort)
  const entryDeviceType: NetworkVLANEntry['device_type'] = deviceKind === 'router'
    ? 'ROUTER'
    : deviceKind === 'switch' ? 'SWITCH' : 'OLT'
  const entryPortName = deviceKind === 'router' ? activePort : (selectedSwitchPort?.port_key ?? '')
  const vlanEntries = useQuery({
    queryKey: ['network-vlan-entries', entryDeviceType, deviceID, entryPortName],
    queryFn: () => getNetworkVLANEntries(entryDeviceType, Number(deviceID), entryPortName),
    enabled: Boolean(deviceID && entryPortName),
  })
  const selectedVLANEntry = vlanEntries.data?.find((entry) => entry.id === selectedVLANEntryID)
  const saveEntry = useMutation({
    mutationFn: ({ id, entry }: { id: number | null; entry: NetworkVLANEntryInput }) => id === null
      ? createNetworkVLANEntry(entry)
      : updateNetworkVLANEntry(id, entry),
    onSuccess: async (entry) => {
      setEditingEntryID(null)
      setSelectedVLANEntryID(entry.id)
      setVLANIDInput('')
      setVLANNameInput('')
      await queryClient.invalidateQueries({
        queryKey: ['network-vlan-entries', entryDeviceType, deviceID, entryPortName],
      })
    },
  })
  const removeEntry = useMutation({
    mutationFn: deleteNetworkVLANEntry,
    onSuccess: async (_, deletedID) => {
      if (selectedVLANEntryID === deletedID) setSelectedVLANEntryID(null)
      if (editingEntryID === deletedID) {
        setEditingEntryID(null)
        setVLANIDInput('')
        setVLANNameInput('')
      }
      await queryClient.invalidateQueries({
        queryKey: ['network-vlan-entries', entryDeviceType, deviceID, entryPortName],
      })
    },
  })

  const routerVLANRows = (vlanReport.data?.vlans ?? []).filter((vlan) =>
    vlan.parent_interface === activePort &&
    (!selectedVLANEntry ||
      (vlan.vlan_id === selectedVLANEntry.vlan_id &&
        vlan.name === selectedVLANEntry.vlan_name)),
  )
  const routerInterfaceTraffic = (vlanReport.data?.interfaces ?? []).find(
    (iface) => iface.name === activePort,
  )

  const deviceLoading = deviceKind === 'router'
    ? routers.isLoading || routerInterfaces.isLoading || (Boolean(activePort) && vlanReport.isLoading)
    : devices.isLoading || switchPorts.isLoading
  const deviceError = deviceKind === 'router'
    ? routers.isError
    : devices.isError || switchPorts.isError

  return (
    <Box sx={{ maxWidth: 1400, mx: 'auto' }}>
      <Typography variant="h4" sx={{ fontWeight: 800, mb: 0.5 }}>Interface Traffic Report</Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        View traffic on the selected interface even when no VLAN is configured; manage VLAN IDs and names below.
      </Typography>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mb: 2 }}>
            <TextField
              select
              label="Device type"
              value={deviceKind}
              onChange={(event) => {
                const value = event.target.value
                setDeviceKind(value === 'switch' || value === 'olt' ? value : 'router')
                setDeviceSelection('')
                setPortSelection('')
                setManualPortName('')
                setSelectedVLANEntryID(null)
                setEditingEntryID(null)
                setVLANIDInput('')
                setVLANNameInput('')
              }}
              sx={{ minWidth: 170 }}
            >
              <MenuItem value="router">Router / MikroTik</MenuItem>
              <MenuItem value="switch">Switch</MenuItem>
              <MenuItem value="olt">OLT</MenuItem>
            </TextField>
            <TextField
              select
              label={deviceKind === 'router' ? 'Router' : deviceKind === 'switch' ? 'Switch' : 'OLT'}
              value={deviceID}
              onChange={(event) => {
                setDeviceSelection(event.target.value)
                setPortSelection('')
                setManualPortName('')
                setSelectedVLANEntryID(null)
                setEditingEntryID(null)
                setVLANIDInput('')
                setVLANNameInput('')
              }}
              disabled={!selectedDevices.length}
              sx={{ minWidth: 240 }}
            >
              {selectedDevices.map((device) => (
                <MenuItem key={device.id} value={String(device.id)}>{device.code} — {device.name}</MenuItem>
              ))}
            </TextField>
            {deviceKind === 'router' ? (
              <Autocomplete
                freeSolo
                options={routerPorts}
                value={activePort}
                inputValue={manualPortName || activePort}
                onChange={(_, value) => {
                  const port = typeof value === 'string' ? value : ''
                  setPortSelection(port)
                  setManualPortName(port)
                  setSelectedVLANEntryID(null)
                  setEditingEntryID(null)
                  setVLANIDInput('')
                  setVLANNameInput('')
                }}
                onInputChange={(_, value, reason) => {
                  if (reason === 'input' || reason === 'clear') {
                    setManualPortName(value)
                    setSelectedVLANEntryID(null)
                    setEditingEntryID(null)
                    setVLANIDInput('')
                    setVLANNameInput('')
                  }
                }}
                disabled={!deviceID}
                sx={{ minWidth: 260 }}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="MikroTik port / interface"
                    placeholder="Type or select, e.g. ether1"
                    helperText={routerInterfaces.isError
                      ? 'Interface list unavailable; enter the exact RouterOS name.'
                      : 'Select a detected interface or enter its exact RouterOS name.'}
                    slotProps={{ ...params.slotProps, htmlInput: { ...params.slotProps.htmlInput, maxLength: 120 } }}
                  />
                )}
              />
            ) : (
              <TextField
                select
                label="Port"
                value={activePort}
                onChange={(event) => {
                  setPortSelection(event.target.value)
                  setSelectedVLANEntryID(null)
                  setEditingEntryID(null)
                  setVLANIDInput('')
                  setVLANNameInput('')
                }}
                disabled={switchPorts.isLoading || !switchPortRows.length}
                sx={{ minWidth: 200 }}
              >
                {switchPortRows.map((port) => (
                  <MenuItem key={port.id} value={String(port.id)}>
                    {port.name || port.description || port.port_key}
                  </MenuItem>
                ))}
              </TextField>
            )}
            <TextField
              select
              label="Saved VLAN filter"
              value={selectedVLANEntryID ?? ''}
              onChange={(event) => setSelectedVLANEntryID(event.target.value ? Number(event.target.value) : null)}
              disabled={!vlanEntries.data?.length}
              sx={{ minWidth: 220 }}
            >
              <MenuItem value="">All VLANs on this port</MenuItem>
              {(vlanEntries.data ?? []).map((entry) => (
                <MenuItem key={entry.id} value={entry.id}>
                  {entry.vlan_id} — {entry.vlan_name}
                </MenuItem>
              ))}
            </TextField>
            {deviceKind === 'router' && (
              <Button
              variant={savedDefaultPort === activePort ? 'outlined' : 'contained'}
              disabled={!activePort || savedDefaultPort === activePort}
              onClick={() => {
                saveDefaultRouterInterface(deviceID, activePort)
                setSavedDefaults((current) => ({ ...current, [deviceID]: activePort }))
              }}
              sx={{ alignSelf: 'center' }}
              >
              {savedDefaultPort === activePort ? 'Default saved' : 'Save as default'}
              </Button>
            )}
          </Box>
          {deviceKind === 'router' && savedDefaultPort && (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1.5 }}>
              Saved default interface: {savedDefaultPort}
            </Typography>
          )}
          {deviceLoading ? (
            <Box sx={{ display: 'grid', placeItems: 'center', minHeight: 120 }}>
              <CircularProgress aria-label="Loading interface traffic report" />
            </Box>
          ) : deviceError ? (
            <Alert severity="error">
              Unable to load {deviceKind === 'router' ? 'router interface traffic' : `${deviceKind === 'switch' ? 'switch' : 'OLT'} ports`}.
              Check device connectivity and monitoring permissions.
            </Alert>
          ) : deviceKind === 'router' ? (
            <>
              {routerInterfaces.isError && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                  {getAPIErrorMessage(routerInterfaces.error, 'Could not load the MikroTik interface list. Type the exact port name above to continue.')}
                </Alert>
              )}
              {vlanReport.isError ? (
                <Alert severity="error">
                  {getAPIErrorMessage(vlanReport.error, 'Unable to load VLAN traffic from this router.')}
                </Alert>
              ) : routerInterfaceTraffic ? (
                <>
                  <Typography variant="caption" color="text.secondary">
                    Sampled {new Date(vlanReport.data!.sampled_at).toLocaleString()} · auto-refreshes every 30 seconds
                  </Typography>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Interface</TableCell>
                        <TableCell align="right">RX</TableCell>
                        <TableCell align="right">TX</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      <TableRow>
                        <TableCell>{routerInterfaceTraffic.name}</TableCell>
                        <TableCell align="right">{formatRate(routerInterfaceTraffic.rx_bps)}</TableCell>
                        <TableCell align="right">{formatRate(routerInterfaceTraffic.tx_bps)}</TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>
                  {routerVLANRows.length === 0 && (
                    <Alert severity="info" sx={{ mt: 2 }}>
                      {activePort} has no configured VLAN interfaces; interface traffic is shown above.
                    </Alert>
                  )}
                  {routerVLANRows.length > 0 && (
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>VLAN ID</TableCell>
                        <TableCell>VLAN interface</TableCell>
                        <TableCell align="right">RX</TableCell>
                        <TableCell align="right">TX</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {routerVLANRows.map((vlan) => (
                        <TableRow key={`${vlan.parent_interface}-${vlan.vlan_id}-${vlan.name}`}>
                          <TableCell>{vlan.vlan_id}</TableCell>
                          <TableCell>{vlan.name}</TableCell>
                          <TableCell align="right">{formatRate(vlan.rx_bps)}</TableCell>
                          <TableCell align="right">{formatRate(vlan.tx_bps)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                  )}
                </>
              ) : (
                <Alert severity="info">
                  {activePort
                    ? 'No RouterOS traffic sample was found for this interface.'
                    : 'Choose a detected interface or type a MikroTik port name.'}
                </Alert>
              )}
            </>
          ) : selectedSwitchPort ? (
            <>
              <Alert severity="info" sx={{ mb: 2 }}>
                This {deviceKind === 'switch' ? 'switch' : 'OLT'} currently provides SNMP port totals only. The entered VLAN is not represented as a per-VLAN counter.
              </Alert>
              {selectedSwitchPort.latest_sample ? (
                <>
                  <Typography variant="caption" color="text.secondary">
                    Port sample: {new Date(selectedSwitchPort.latest_sample.sampled_at).toLocaleString()} · auto-refreshes every 30 seconds
                  </Typography>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>{deviceKind === 'switch' ? 'Switch port' : 'OLT port'}</TableCell>
                        <TableCell>VLAN reference</TableCell>
                        <TableCell align="right">RX</TableCell>
                        <TableCell align="right">TX</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      <TableRow>
                        <TableCell>{selectedSwitchPort.name || selectedSwitchPort.description || selectedSwitchPort.port_key}</TableCell>
                        <TableCell>
                          {selectedVLANEntry
                            ? `${selectedVLANEntry.vlan_id} — ${selectedVLANEntry.vlan_name}`
                            : (vlanEntries.data?.length ? 'Saved VLAN metadata only' : '—')}
                        </TableCell>
                        <TableCell align="right">{selectedSwitchPort.latest_sample.in_mbps.toFixed(2)} Mbps</TableCell>
                        <TableCell align="right">{selectedSwitchPort.latest_sample.out_mbps.toFixed(2)} Mbps</TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>
                </>
              ) : (
                <Alert severity="warning">No traffic sample is available for this switch port yet.</Alert>
              )}
            </>
          ) : (
            <Alert severity="info">
              {deviceID
                ? 'No monitored switch port is available.'
                : `No monitored ${deviceKind === 'switch' ? 'switch' : 'OLT'} is available.`}
            </Alert>
          )}
          <Box sx={{ mt: 3, pt: 2, borderTop: 1, borderColor: 'divider' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              {editingEntryID === null ? 'Save VLAN on this device port' : 'Edit saved VLAN'}
            </Typography>
            <Typography color="text.secondary" variant="body2" sx={{ mb: 2 }}>
              Save the VLAN ID and name for the selected device and port. Switch/OLT traffic remains port-level until per-VLAN counters are available.
            </Typography>
            {saveEntry.isError && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {getAPIErrorMessage(saveEntry.error, 'Unable to save VLAN entry.')}
              </Alert>
            )}
            {removeEntry.isError && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {getAPIErrorMessage(removeEntry.error, 'Unable to delete VLAN entry.')}
              </Alert>
            )}
            <Box
              component="form"
              onSubmit={(event) => {
                event.preventDefault()
                if (!deviceID || !entryPortName) return
                saveEntry.mutate({
                  id: editingEntryID,
                  entry: {
                    device_type: entryDeviceType,
                    device_id: Number(deviceID),
                    port_name: entryPortName,
                    vlan_id: Number(vlanIDInput),
                    vlan_name: vlanNameInput.trim(),
                  },
                })
              }}
              sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'flex-start', mb: 2 }}
            >
              <TextField
                label="VLAN ID"
                type="number"
                value={vlanIDInput}
                onChange={(event) => setVLANIDInput(event.target.value)}
                slotProps={{ htmlInput: { min: 1, max: 4094, step: 1 } }}
                required
                disabled={!deviceID || !entryPortName || saveEntry.isPending}
                sx={{ width: 150 }}
              />
              <TextField
                label="VLAN name"
                value={vlanNameInput}
                onChange={(event) => setVLANNameInput(event.target.value)}
                slotProps={{ htmlInput: { maxLength: 120 } }}
                required
                disabled={!deviceID || !entryPortName || saveEntry.isPending}
                sx={{ minWidth: 240 }}
              />
              <Button
                type="submit"
                variant="contained"
                startIcon={editingEntryID === null ? <AddIcon /> : undefined}
                disabled={!deviceID || !entryPortName || saveEntry.isPending}
              >
                {saveEntry.isPending ? 'Saving…' : editingEntryID === null ? 'Add VLAN' : 'Save changes'}
              </Button>
              {editingEntryID !== null && (
                <Button
                  onClick={() => {
                    setEditingEntryID(null)
                    setVLANIDInput('')
                    setVLANNameInput('')
                  }}
                  disabled={saveEntry.isPending}
                >
                  Cancel
                </Button>
              )}
            </Box>
            {vlanEntries.isError ? (
              <Alert severity="error">
                {getAPIErrorMessage(vlanEntries.error, 'Unable to load saved VLAN entries.')}
              </Alert>
            ) : vlanEntries.isLoading ? (
              <CircularProgress size={24} aria-label="Loading saved VLAN entries" />
            ) : vlanEntries.data?.length ? (
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>VLAN ID</TableCell>
                    <TableCell>VLAN name</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {vlanEntries.data.map((entry) => (
                    <TableRow key={entry.id}>
                      <TableCell>{entry.vlan_id}</TableCell>
                      <TableCell>{entry.vlan_name}</TableCell>
                      <TableCell align="right">
                        <IconButton
                          aria-label={`Use VLAN ${entry.vlan_id} as report filter`}
                          onClick={() => setSelectedVLANEntryID(entry.id)}
                        >
                          <VisibilityIcon />
                        </IconButton>
                        <IconButton
                          aria-label={`Edit VLAN ${entry.vlan_id}`}
                          onClick={() => {
                            setEditingEntryID(entry.id)
                            setVLANIDInput(String(entry.vlan_id))
                            setVLANNameInput(entry.vlan_name)
                          }}
                        >
                          <EditIcon />
                        </IconButton>
                        <IconButton
                          aria-label={`Delete VLAN ${entry.vlan_id}`}
                          disabled={removeEntry.isPending}
                          onClick={() => {
                            if (window.confirm(`Delete VLAN ${entry.vlan_id} (${entry.vlan_name}) from this port?`)) {
                              removeEntry.mutate(entry.id)
                            }
                          }}
                        >
                          <DeleteIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <Alert severity="info">No VLANs saved for this device port yet.</Alert>
            )}
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
