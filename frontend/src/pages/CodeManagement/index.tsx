import { useCallback, useEffect, useMemo, useState } from 'react'

import {
  Alert, Box, Button, Card, CardContent, CircularProgress, Dialog,
  DialogActions, DialogContent, DialogTitle, Tab, Table, TableBody,
  TableCell, TableHead, TableRow, Tabs, TextField, Typography,
} from '@mui/material'
import EditIcon from '@mui/icons-material/Edit'

import { getAgents, getPOPs, updateManagedCode, type Agent, type POP } from '../../api/distribution'
import { getPackages, type Package } from '../../api/packages'
import { getAPIErrorMessage } from '../../api/errors'

type EntityType = 'agent' | 'pop' | 'package'
type ManagedRow = { id: number; code: string; name: string; detail: string }

const numericCodeSort = (a: ManagedRow, b: ManagedRow) =>
  a.code.localeCompare(b.code, undefined, { numeric: true })

export default function CodeManagement() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [pops, setPOPs] = useState<POP[]>([])
  const [packages, setPackages] = useState<Package[]>([])
  const [tab, setTab] = useState<EntityType>('agent')
  const [editing, setEditing] = useState<ManagedRow | null>(null)
  const [newCode, setNewCode] = useState('')
  const [reason, setReason] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [agentRows, popRows, packageRows] = await Promise.all([getAgents(), getPOPs(), getPackages()])
      setAgents(agentRows)
      setPOPs(popRows)
      setPackages(packageRows.packages)
    } catch (err) {
      setError(getAPIErrorMessage(err, 'Failed to load code management data.'))
    } finally { setLoading(false) }
  }, [])

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load()
  }, [load])

  const rows = useMemo<ManagedRow[]>(() => {
    if (tab === 'agent') return agents.map((row) => ({ id: row.id, code: row.code, name: row.name, detail: row.pop_names?.join(', ') || row.pop_name })).sort(numericCodeSort)
    if (tab === 'pop') return pops.map((row) => ({ id: row.id, code: row.code, name: row.name, detail: row.manager_name || row.address })).sort(numericCodeSort)
    return packages.map((row) => ({ id: row.id, code: row.package_code, name: row.name, detail: `MikroTik: ${row.mikrotik_profile || '—'}` })).sort(numericCodeSort)
  }, [agents, packages, pops, tab])

  const openEdit = (row: ManagedRow) => {
    setEditing(row); setNewCode(row.code); setReason(''); setError(''); setSuccess('')
  }

  const save = async () => {
    if (!editing || !newCode.trim() || !reason.trim()) return
    try {
      setSaving(true); setError(''); setSuccess('')
      await updateManagedCode(tab, editing.id, newCode.trim(), reason.trim())
      setEditing(null)
      setSuccess(`${editing.code} changed to ${newCode.trim().toUpperCase()}.`)
      await load()
    } catch (err) {
      setError(getAPIErrorMessage(err, 'Failed to update code.'))
    } finally { setSaving(false) }
  }

  return (
    <Box>
      <Typography variant="h4" component="h1" sx={{ fontWeight: 700, mb: 1 }}>Code & Serial Management</Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>Super Admin-only controlled correction for Agent, POP and Package codes. Every change requires a reason and is audited.</Typography>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}
      <Card><CardContent>
        <Tabs value={tab} onChange={(_, value: EntityType) => setTab(value)} sx={{ mb: 2 }}>
          <Tab value="agent" label="Agent Codes" />
          <Tab value="pop" label="POP Codes" />
          <Tab value="package" label="Package Codes" />
        </Tabs>
        {loading ? <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}><CircularProgress /></Box> : (
          <Table>
            <TableHead><TableRow><TableCell>S/N</TableCell><TableCell>Current Code</TableCell><TableCell>Name</TableCell><TableCell>Details</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead>
            <TableBody>{rows.map((row, index) => (
              <TableRow key={row.id} hover><TableCell>{index + 1}</TableCell><TableCell sx={{ fontWeight: 700 }}>{row.code}</TableCell><TableCell>{row.name}</TableCell><TableCell>{row.detail || '—'}</TableCell><TableCell align="right"><Button size="small" startIcon={<EditIcon />} onClick={() => openEdit(row)}>Correct Code</Button></TableCell></TableRow>
            ))}</TableBody>
          </Table>
        )}
      </CardContent></Card>

      <Dialog open={Boolean(editing)} onClose={() => !saving && setEditing(null)} fullWidth maxWidth="xs">
        <DialogTitle>Correct {tab.toUpperCase()} Code</DialogTitle>
        <DialogContent dividers>
          <Typography sx={{ mb: 2 }}>{editing?.name}<br /><strong>Current:</strong> {editing?.code}</Typography>
          <TextField fullWidth required label="New Code" value={newCode} onChange={(event) => setNewCode(event.target.value.toUpperCase())} helperText="2-30 characters: A-Z, 0-9, - or _" sx={{ mb: 2 }} />
          <TextField fullWidth required multiline minRows={3} label="Reason" value={reason} onChange={(event) => setReason(event.target.value)} helperText="Required for audit history" />
        </DialogContent>
        <DialogActions><Button onClick={() => setEditing(null)} disabled={saving}>Cancel</Button><Button variant="contained" disabled={saving || !newCode.trim() || !reason.trim() || newCode.trim() === editing?.code} onClick={() => void save()}>{saving ? 'Saving...' : 'Confirm Code Change'}</Button></DialogActions>
      </Dialog>
    </Box>
  )
}
