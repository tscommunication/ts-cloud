import { useCallback, useEffect, useState } from 'react'
import { Alert, Box, Button, Card, CardContent, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Grid, MenuItem, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography } from '@mui/material'
import { getAgentCollections, type AgentCollection, type AgentCollectionReport } from '../../api/agentCollections'
import { getAgents, type Agent } from '../../api/distribution'
import { getAPIErrorMessage } from '../../api/errors'
import { createAgentSettlement, getAgentSettlements, voidAgentSettlement, type AgentSettlement, type AgentSettlementReport } from '../../api/agentSettlements'
import { getStoredUser } from '../../api/auth'

const money = new Intl.NumberFormat('en-BD', { style: 'currency', currency: 'BDT', maximumFractionDigits: 2 })

export default function AgentCollections() {
  const [report, setReport] = useState<AgentCollectionReport>({ collections: [], count: 0, total_amount: 0, total_commission: 0 })
  const [agents, setAgents] = useState<Agent[]>([])
  const storedUser = getStoredUser()
  const [agentID, setAgentID] = useState<number | ''>(storedUser?.role === 'agent' ? storedUser.agent_id ?? '' : '')
  const [status, setStatus] = useState<'' | 'ACTIVE' | 'VOID'>('')
  const [error, setError] = useState('')
  const [settlements, setSettlements] = useState<AgentSettlementReport>({ settlements: [], earned: 0, paid: 0, payable: 0 })
  const [dialog, setDialog] = useState(false)
  const [saving, setSaving] = useState(false)
  const [settlementForm, setSettlementForm] = useState({ amount: 0, method: 'CASH', transaction_id: '', paid_at: new Date().toISOString().slice(0, 10), remarks: '' })
  const isSuperadmin = storedUser?.role === 'superadmin'
  const isAgent = storedUser?.role === 'agent'

  const load = useCallback(async () => {
    try { setError(''); setReport(await getAgentCollections({ agent_id: agentID || undefined, status })) }
    catch (err) { setError(getAPIErrorMessage(err, 'Failed to load collection report.')) }
  }, [agentID, status])
  useEffect(() => { if (isAgent) return; const start = async () => { try { setAgents(await getAgents()) } catch (err) { setError(getAPIErrorMessage(err, 'Failed to load agents.')) } }; void start() }, [isAgent])
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load()
  }, [load])
  useEffect(() => { const loadSettlements = async () => { if (!agentID) { setSettlements({ settlements: [], earned: 0, paid: 0, payable: 0 }); return }; try { setSettlements(await getAgentSettlements(agentID)) } catch (err) { setError(getAPIErrorMessage(err, 'Failed to load settlements.')) } }; void loadSettlements() }, [agentID])

  const refreshSettlements = async () => { if (agentID) setSettlements(await getAgentSettlements(agentID)) }
  const saveSettlement = async () => { if (!agentID) return; try { setSaving(true); setError(''); await createAgentSettlement({ agent_id: agentID, ...settlementForm }); setDialog(false); await refreshSettlements() } catch (err) { setError(getAPIErrorMessage(err, 'Failed to save settlement.')) } finally { setSaving(false) } }
  const voidSettlement = async (id: number) => { try { setError(''); await voidAgentSettlement(id); await refreshSettlements() } catch (err) { setError(getAPIErrorMessage(err, 'Failed to void settlement.')) } }

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
        {!isAgent && <TextField select size="small" label="Agent" value={agentID} onChange={(event) => setAgentID(event.target.value ? Number(event.target.value) : '')} sx={{ minWidth: 240 }}><MenuItem value="">All Agents</MenuItem>{agents.map((agent) => <MenuItem key={agent.id} value={agent.id}>{agent.code} — {agent.name}</MenuItem>)}</TextField>}
        <TextField select size="small" label="Status" value={status} onChange={(event) => setStatus(event.target.value as '' | 'ACTIVE' | 'VOID')} sx={{ minWidth: 160 }}><MenuItem value="">All Statuses</MenuItem><MenuItem value="ACTIVE">Active</MenuItem><MenuItem value="VOID">Void</MenuItem></TextField>
      </Box>
      <Table><TableHead><TableRow><TableCell>Date / Receipt</TableCell><TableCell>Agent</TableCell><TableCell>Customer</TableCell><TableCell align="right">Collection</TableCell><TableCell align="right">Rate</TableCell><TableCell align="right">Commission</TableCell><TableCell>Status</TableCell></TableRow></TableHead><TableBody>
        {report.collections.map((row: AgentCollection) => <TableRow key={row.id}><TableCell>{new Date(row.collected_at).toLocaleDateString('en-BD')}<br /><Typography variant="caption">{row.receipt_no}</Typography></TableCell><TableCell>{row.agent_name}</TableCell><TableCell>{row.customer_code}<br />{row.customer_name}</TableCell><TableCell align="right">{money.format(row.amount)}</TableCell><TableCell align="right">{row.commission_rate}%</TableCell><TableCell align="right">{money.format(row.commission_amount)}</TableCell><TableCell><Chip size="small" color={row.status === 'ACTIVE' ? 'success' : 'default'} label={row.status} /></TableCell></TableRow>)}
        {!report.collections.length && <TableRow><TableCell colSpan={7} align="center">No agent collections recorded yet.</TableCell></TableRow>}
      </TableBody></Table>
    </CardContent></Card>
    {agentID && <Card sx={{ mt: 3 }}><CardContent>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, mb: 2, alignItems: 'center' }}><Box><Typography variant="h6">Commission Settlements</Typography><Typography color="text.secondary">Earned {money.format(settlements.earned)} · Paid {money.format(settlements.paid)} · Payable {money.format(settlements.payable)}</Typography></Box>{isSuperadmin && <Button variant="contained" disabled={settlements.payable <= 0} onClick={() => { setSettlementForm((current) => ({ ...current, amount: settlements.payable })); setDialog(true) }}>Pay Commission</Button>}</Box>
      <Table size="small"><TableHead><TableRow><TableCell>Date / Settlement</TableCell><TableCell>Method</TableCell><TableCell>Transaction</TableCell><TableCell align="right">Amount</TableCell><TableCell>Status</TableCell><TableCell align="right">Action</TableCell></TableRow></TableHead><TableBody>
        {settlements.settlements.map((row: AgentSettlement) => <TableRow key={row.id}><TableCell>{new Date(row.paid_at).toLocaleDateString('en-BD')}<br />{row.settlement_no}</TableCell><TableCell>{row.method}</TableCell><TableCell>{row.transaction_id || '—'}</TableCell><TableCell align="right">{money.format(row.amount)}</TableCell><TableCell><Chip size="small" color={row.status === 'PAID' ? 'success' : 'default'} label={row.status} /></TableCell><TableCell align="right">{isSuperadmin && row.status === 'PAID' && <Button color="error" size="small" onClick={() => voidSettlement(row.id)}>Void</Button>}</TableCell></TableRow>)}
        {!settlements.settlements.length && <TableRow><TableCell colSpan={6} align="center">No commission settlements.</TableCell></TableRow>}
      </TableBody></Table>
    </CardContent></Card>}
    <Dialog open={dialog} onClose={() => setDialog(false)} fullWidth maxWidth="sm"><DialogTitle>Pay Agent Commission</DialogTitle><DialogContent><Grid container spacing={2} sx={{ mt: 0 }}><Grid size={6}><TextField fullWidth required type="number" label="Amount" value={settlementForm.amount} slotProps={{ htmlInput: { min: 0.01, max: settlements.payable, step: 0.01 } }} onChange={(event) => setSettlementForm({ ...settlementForm, amount: Number(event.target.value) })} /></Grid><Grid size={6}><TextField fullWidth select label="Method" value={settlementForm.method} onChange={(event) => setSettlementForm({ ...settlementForm, method: event.target.value })}>{['CASH','BKASH','NAGAD','ROCKET','BANK'].map((method) => <MenuItem key={method} value={method}>{method}</MenuItem>)}</TextField></Grid><Grid size={6}><TextField fullWidth type="date" label="Paid Date" value={settlementForm.paid_at} slotProps={{ inputLabel: { shrink: true } }} onChange={(event) => setSettlementForm({ ...settlementForm, paid_at: event.target.value })} /></Grid><Grid size={6}><TextField fullWidth required={settlementForm.method !== 'CASH'} label="Transaction ID" value={settlementForm.transaction_id} onChange={(event) => setSettlementForm({ ...settlementForm, transaction_id: event.target.value })} /></Grid><Grid size={12}><TextField fullWidth multiline label="Remarks" value={settlementForm.remarks} onChange={(event) => setSettlementForm({ ...settlementForm, remarks: event.target.value })} /></Grid></Grid></DialogContent><DialogActions><Button onClick={() => setDialog(false)}>Cancel</Button><Button variant="contained" disabled={saving || settlementForm.amount <= 0 || settlementForm.amount > settlements.payable || (settlementForm.method !== 'CASH' && !settlementForm.transaction_id.trim())} onClick={saveSettlement}>Confirm Payment</Button></DialogActions></Dialog>
  </Box>
}
