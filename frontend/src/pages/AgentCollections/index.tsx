import { useCallback, useEffect, useState } from 'react'
import { Alert, Box, Card, CardContent, Chip, Grid, MenuItem, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { getAgentCollections, type AgentCollection, type AgentCollectionReport } from '../../api/agentCollections'
import { getAgents, type Agent } from '../../api/distribution'
import { getAPIErrorMessage } from '../../api/errors'

const money = new Intl.NumberFormat('en-BD', { style: 'currency', currency: 'BDT', maximumFractionDigits: 2 })

export default function AgentCollections() {
  const [report, setReport] = useState<AgentCollectionReport>({ collections: [], count: 0, total_amount: 0, total_commission: 0 })
  const [agents, setAgents] = useState<Agent[]>([])
  const [agentID, setAgentID] = useState<number | ''>('')
  const [status, setStatus] = useState<'' | 'ACTIVE' | 'VOID'>('')
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    try { setError(''); setReport(await getAgentCollections({ agent_id: agentID || undefined, status })) }
    catch (err) { setError(getAPIErrorMessage(err, 'Failed to load collection report.')) }
  }, [agentID, status])
  useEffect(() => { const start = async () => { try { setAgents(await getAgents()) } catch (err) { setError(getAPIErrorMessage(err, 'Failed to load agents.')) } }; void start() }, [])
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load()
  }, [load])

  const cards = [
    ['Collection Entries', String(report.count)],
    ['Total Collected', money.format(report.total_amount)],
    ['Agent Commission', money.format(report.total_commission)],
  ]
  return <Box>
    <Box sx={{ mb: 3 }}><Typography variant="h4">Agent Collections</Typography><Typography color="text.secondary">Track collections and commission snapshots generated from customer payments.</Typography></Box>
    {error && <Alert severity="error" onClose={() => setError('')} sx={{ mb: 2 }}>{error}</Alert>}
    <Grid container spacing={2} sx={{ mb: 3 }}>{cards.map(([label, value]) => <Grid key={label} size={{ xs: 12, md: 4 }}><Card><CardContent><Typography color="text.secondary" variant="body2">{label}</Typography><Typography variant="h5" sx={{ mt: 1 }}>{value}</Typography></CardContent></Card></Grid>)}</Grid>
    <Card><CardContent>
      <Box sx={{ display: 'flex', gap: 2, mb: 2, flexWrap: 'wrap' }}>
        <TextField select size="small" label="Agent" value={agentID} onChange={(event) => setAgentID(event.target.value ? Number(event.target.value) : '')} sx={{ minWidth: 240 }}><MenuItem value="">All Agents</MenuItem>{agents.map((agent) => <MenuItem key={agent.id} value={agent.id}>{agent.code} — {agent.name}</MenuItem>)}</TextField>
        <TextField select size="small" label="Status" value={status} onChange={(event) => setStatus(event.target.value as '' | 'ACTIVE' | 'VOID')} sx={{ minWidth: 160 }}><MenuItem value="">All Statuses</MenuItem><MenuItem value="ACTIVE">Active</MenuItem><MenuItem value="VOID">Void</MenuItem></TextField>
      </Box>
      <Table><TableHead><TableRow><TableCell>Date / Receipt</TableCell><TableCell>Agent</TableCell><TableCell>Customer</TableCell><TableCell align="right">Collection</TableCell><TableCell align="right">Rate</TableCell><TableCell align="right">Commission</TableCell><TableCell>Status</TableCell></TableRow></TableHead><TableBody>
        {report.collections.map((row: AgentCollection) => <TableRow key={row.id}><TableCell>{new Date(row.collected_at).toLocaleDateString('en-BD')}<br /><Typography variant="caption">{row.receipt_no}</Typography></TableCell><TableCell>{row.agent_name}</TableCell><TableCell>{row.customer_code}<br />{row.customer_name}</TableCell><TableCell align="right">{money.format(row.amount)}</TableCell><TableCell align="right">{row.commission_rate}%</TableCell><TableCell align="right">{money.format(row.commission_amount)}</TableCell><TableCell><Chip size="small" color={row.status === 'ACTIVE' ? 'success' : 'default'} label={row.status} /></TableCell></TableRow>)}
        {!report.collections.length && <TableRow><TableCell colSpan={7} align="center">No agent collections recorded yet.</TableCell></TableRow>}
      </TableBody></Table>
    </CardContent></Card>
  </Box>
}
