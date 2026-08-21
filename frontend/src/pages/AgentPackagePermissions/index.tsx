import { useCallback, useEffect, useMemo, useState } from 'react'

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Divider,
  FormControlLabel,
  Grid,
  List,
  ListItemButton,
  ListItemText,
  TextField,
  Typography,
} from '@mui/material'

import SaveIcon from '@mui/icons-material/Save'
import SearchIcon from '@mui/icons-material/Search'

import {
  getAgents,
  updateAgentPackages,
  type Agent,
} from '../../api/distribution'
import { getPackages, type Package } from '../../api/packages'
import { getAPIErrorMessage } from '../../api/errors'

export default function AgentPackagePermissions() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [packages, setPackages] = useState<Package[]>([])
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)
  const [selectedPackageIDs, setSelectedPackageIDs] = useState<number[]>([])
  const [search, setSearch] = useState('')
  const [packageSearch, setPackageSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      const [agentRows, packageRows] = await Promise.all([
        getAgents(),
        getPackages(),
      ])
      setAgents(agentRows)
      setPackages(packageRows.packages.filter((pkg) => pkg.status === 'ACTIVE'))
      setSelectedAgent((current) => {
        if (!current) return null
        return agentRows.find((row) => row.id === current.id) ?? null
      })
    } catch (err) {
      setError(getAPIErrorMessage(err, 'Failed to load package permissions.'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load()
  }, [load])

  const filteredAgents = useMemo(() => {
    const query = search.trim().toLowerCase()
    return agents.filter((agent) =>
      !query || [agent.code, agent.name, agent.pop_names?.join(' ')].join(' ').toLowerCase().includes(query),
    ).sort((left, right) => left.code.localeCompare(right.code, undefined, { numeric: true }))
  }, [agents, search])

  const filteredPackages = useMemo(() => {
    const query = packageSearch.trim().toLowerCase()
    return packages.filter((pkg) => !query || [pkg.package_code, pkg.name, pkg.mikrotik_profile].join(' ').toLowerCase().includes(query))
  }, [packages, packageSearch])

  const chooseAgent = (agent: Agent) => {
    setSelectedAgent(agent)
    setSelectedPackageIDs(agent.package_ids ?? [])
    setError('')
    setSuccess('')
  }

  const togglePackage = (packageID: number) => {
    setSelectedPackageIDs((current) =>
      current.includes(packageID)
        ? current.filter((id) => id !== packageID)
        : [...current, packageID],
    )
  }

  const save = async () => {
    if (!selectedAgent) return
    try {
      setSaving(true)
      setError('')
      setSuccess('')
      const updated = await updateAgentPackages(selectedAgent.id, selectedPackageIDs)
      setAgents((current) => current.map((agent) => agent.id === updated.id ? updated : agent))
      setSelectedAgent(updated)
      setSelectedPackageIDs(updated.package_ids ?? [])
      setSuccess(`${updated.code} — ${updated.name} package permissions updated.`)
    } catch (err) {
      setError(getAPIErrorMessage(err, 'Failed to update package permissions.'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box>
      <Typography variant="h4" component="h1" sx={{ fontWeight: 700, mb: 1 }}>
        Assign Package Permission
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Select an Agent / Reseller, then choose which packages they may offer to customers.
      </Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}><CircularProgress /></Box>
      ) : (
        <Grid container spacing={3}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <TextField
                  fullWidth
                  size="small"
                  label="Search Agent / Reseller"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  slotProps={{ input: { startAdornment: <SearchIcon sx={{ mr: 1 }} /> } }}
                  sx={{ mb: 2 }}
                />
                <List disablePadding>
                  {filteredAgents.map((agent, index) => (
                    <ListItemButton
                      key={agent.id}
                      selected={selectedAgent?.id === agent.id}
                      onClick={() => chooseAgent(agent)}
                      sx={{ gap: 1.5, borderRadius: 1 }}
                    >
                      <Box
                        sx={{
                          width: 30,
                          height: 30,
                          flexShrink: 0,
                          borderRadius: '50%',
                          display: 'grid',
                          placeItems: 'center',
                          fontSize: 13,
                          fontWeight: 700,
                          color: selectedAgent?.id === agent.id ? 'primary.contrastText' : 'text.secondary',
                          bgcolor: selectedAgent?.id === agent.id ? 'primary.main' : 'action.hover',
                        }}
                      >
                        {index + 1}
                      </Box>
                      <ListItemText
                        primary={`${agent.code} — ${agent.name}`}
                        secondary={`${agent.pop_names?.join(', ') || agent.pop_name} | ${agent.package_ids?.length ?? 0} package(s)`}
                      />
                    </ListItemButton>
                  ))}
                </List>
              </CardContent>
            </Card>
          </Grid>

          <Grid size={{ xs: 12, md: 8 }}>
            <Card>
              <CardContent>
                {!selectedAgent ? (
                  <Alert severity="info">Click an Agent / Reseller to view and edit assigned packages.</Alert>
                ) : (
                  <>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap' }}>
                      <Box>
                        <Typography variant="h6">{selectedAgent.code} — {selectedAgent.name}</Typography>
                        <Typography color="text.secondary">Assigned {selectedPackageIDs.length} of {packages.length} active packages</Typography>
                      </Box>
                      <Box>
                        <Button onClick={() => setSelectedPackageIDs(packages.map((pkg) => pkg.id))}>Select All</Button>
                        <Button onClick={() => setSelectedPackageIDs([])}>Clear All</Button>
                      </Box>
                    </Box>
                    <Divider sx={{ my: 2 }} />
                    <TextField fullWidth size="small" label="Search packages by code, name or MikroTik profile" value={packageSearch} onChange={(event) => setPackageSearch(event.target.value)} slotProps={{ input: { startAdornment: <SearchIcon sx={{ mr: 1 }} /> } }} sx={{ mb: 2 }} />
                    <Grid container spacing={1}>
                      {filteredPackages.map((pkg) => (
                        <Grid key={pkg.id} size={{ xs: 12, sm: 6 }}>
                          <FormControlLabel
                            control={<Checkbox checked={selectedPackageIDs.includes(pkg.id)} onChange={() => togglePackage(pkg.id)} />}
                            label={
                              <Box>
                                <Typography sx={{ fontWeight: 600 }}>{pkg.package_code} — {pkg.name}</Typography>
                                <Typography variant="caption" color="text.secondary">
                                  BDT {pkg.price.toLocaleString()} | MikroTik: {pkg.mikrotik_profile || '—'}
                                </Typography>
                              </Box>
                            }
                          />
                        </Grid>
                      ))}
                    </Grid>
                    <Divider sx={{ my: 2 }} />
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 2 }}>
                      <Box>{selectedPackageIDs.map((id) => {
                        const pkg = packages.find((item) => item.id === id)
                        return pkg ? <Chip key={id} size="small" label={pkg.name} sx={{ mr: 0.5, mb: 0.5 }} /> : null
                      })}</Box>
                      <Button variant="contained" startIcon={saving ? <CircularProgress size={18} /> : <SaveIcon />} disabled={saving} onClick={() => void save()}>
                        {saving ? 'Saving...' : 'Save Permissions'}
                      </Button>
                    </Box>
                  </>
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}
    </Box>
  )
}
