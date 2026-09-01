import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  MenuItem,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import { getStoredUser } from '../../api/auth'
import { getCustomers } from '../../api/customers'
import {
  approveCustomerChangeRequest,
  createCustomerChangeRequest,
  getCustomerChangeRequestOptions,
  getCustomerChangeRequests,
  rejectCustomerChangeRequest,
  type CustomerChangeRequest,
  type CustomerChangeType,
} from '../../api/customerChangeRequests'
import { getAPIErrorMessage } from '../../api/errors'

const labels: Record<CustomerChangeType, string> = {
  BILLING_CYCLE: 'Billing Cycle Change',
  PACKAGE: 'Package Change',
  LINE_SHIFT: 'Line Shift / Router Change',
  CLOSE: 'Close Customer',
}

function requestedValueLabel(row: CustomerChangeRequest) {
  try {
    const value = JSON.parse(row.requested_value) as Record<string, number>
    if (row.type === 'BILLING_CYCLE') return `Billing day: ${value.billing_day}`
    if (row.type === 'PACKAGE') return `Package ID: ${value.package_id}`
    if (row.type === 'LINE_SHIFT') return `Router ID: ${value.router_id}`
    return 'Disconnect customer service'
  } catch {
    return row.requested_value || '—'
  }
}

export default function CustomerChangeRequests() {
  const queryClient = useQueryClient()
  const isAgent = getStoredUser()?.role === 'agent'
  const [searchParams, setSearchParams] = useSearchParams()
  const [open, setOpen] = useState(false)
  const [customerID, setCustomerID] = useState(0)
  const [type, setType] = useState<CustomerChangeType>('BILLING_CYCLE')
  const [reason, setReason] = useState('')
  const [requestedValue, setRequestedValue] = useState('')
  const [rejecting, setRejecting] = useState<number | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    const requestedCustomerID = Number(searchParams.get('customer_id'))
    if (!isAgent || !Number.isInteger(requestedCustomerID) || requestedCustomerID <= 0) return
    const timer = window.setTimeout(() => {
      setCustomerID(requestedCustomerID)
      setOpen(true)
      setSearchParams({}, { replace: true })
    }, 0)
    return () => window.clearTimeout(timer)
  }, [isAgent, searchParams, setSearchParams])

  const requests = useQuery({
    queryKey: ['customer-change-requests'],
    queryFn: () => getCustomerChangeRequests(),
  })
  const customers = useQuery({
    queryKey: ['customers', 'change-request'],
    queryFn: () => getCustomers({ page: 1, page_size: 100 }),
  })
  const options = useQuery({
    queryKey: ['customer-change-request-options'],
    queryFn: getCustomerChangeRequestOptions,
    enabled: isAgent,
  })

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['customer-change-requests'] })
  const numericValue = Number(requestedValue)
  const requestedValueValid = type === 'CLOSE' || (Number.isInteger(numericValue) && numericValue > 0)

  const resetForm = () => {
    setCustomerID(0)
    setType('BILLING_CYCLE')
    setReason('')
    setRequestedValue('')
  }

  const submit = async () => {
    try {
      setBusy(true)
      setError('')
      setNotice('')
      const payload = type === 'CLOSE'
        ? '{}'
        : JSON.stringify(type === 'BILLING_CYCLE'
          ? { billing_day: numericValue }
          : type === 'PACKAGE'
            ? { package_id: numericValue }
            : { router_id: numericValue })
      await createCustomerChangeRequest({
        customer_id: customerID,
        type,
        reason,
        current_value: '',
        requested_value: payload,
      })
      setOpen(false)
      resetForm()
      await refresh()
    } catch (requestError) {
      setError(getAPIErrorMessage(requestError, 'Request could not be submitted.'))
    } finally {
      setBusy(false)
    }
  }

  const approve = async (id: number) => {
    try {
      setBusy(true)
      setError('')
      setNotice('')
      const result = await approveCustomerChangeRequest(id)
      if (result.execution_error) {
        setNotice(`Database change completed, but MikroTik sync needs attention: ${result.execution_error}`)
      }
      await refresh()
    } catch (requestError) {
      setError(getAPIErrorMessage(requestError, 'Request could not be approved.'))
    } finally {
      setBusy(false)
    }
  }

  const reject = async () => {
    if (!rejecting) return
    try {
      setBusy(true)
      setError('')
      setNotice('')
      await rejectCustomerChangeRequest(rejecting, rejectReason)
      setRejecting(null)
      setRejectReason('')
      await refresh()
    } catch (requestError) {
      setError(getAPIErrorMessage(requestError, 'Request could not be rejected.'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ justifyContent: 'space-between', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>Customer Change Requests</Typography>
          <Typography color="text.secondary">
            {isAgent ? 'Submit and track service-change requests.' : 'Approve or reject agent requests. Approval applies the controlled change immediately.'}
          </Typography>
        </Box>
        {isAgent && <Button variant="contained" onClick={() => setOpen(true)}>New Request</Button>}
      </Stack>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {notice && <Alert severity="warning" sx={{ mb: 2 }}>{notice}</Alert>}
      <Card>
        <CardContent>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Request</TableCell>
                  <TableCell>Customer</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Reason / Requested Change</TableCell>
                  <TableCell>Status</TableCell>
                  {!isAgent && <TableCell align="right">Review</TableCell>}
                </TableRow>
              </TableHead>
              <TableBody>
                {(requests.data ?? []).map((row) => {
                  const customer = (customers.data?.customers ?? []).find((item) => item.id === row.customer_id)
                  return (
                    <TableRow key={row.ID}>
                      <TableCell>
                        {row.request_code}<br />
                        <Typography variant="caption">{new Date(row.CreatedAt).toLocaleString()}</Typography>
                      </TableCell>
                      <TableCell>{customer ? `${customer.customer_code} — ${customer.full_name}` : `#${row.customer_id}`}</TableCell>
                      <TableCell>{labels[row.type]}</TableCell>
                      <TableCell>
                        {row.reason}<br />
                        <Typography variant="caption">{requestedValueLabel(row)}</Typography>
                        {row.rejection_reason && <Alert severity="error" sx={{ mt: 1 }}>{row.rejection_reason}</Alert>}
                        {row.execution_error && <Alert severity="warning" sx={{ mt: 1 }}>MikroTik sync: {row.execution_error}</Alert>}
                      </TableCell>
                      <TableCell>
                        <Chip
                          size="small"
                          label={row.status}
                          color={row.status === 'PENDING' ? 'warning' : row.status === 'REJECTED' ? 'error' : 'success'}
                        />
                      </TableCell>
                      {!isAgent && (
                        <TableCell align="right">
                          {row.status === 'PENDING' && (
                            <>
                              <Button disabled={busy} onClick={() => void approve(row.ID)}>Approve</Button>
                              <Button color="error" disabled={busy} onClick={() => setRejecting(row.ID)}>Reject</Button>
                            </>
                          )}
                        </TableCell>
                      )}
                    </TableRow>
                  )
                })}
                {!requests.isLoading && (requests.data?.length ?? 0) === 0 && (
                  <TableRow><TableCell colSpan={isAgent ? 5 : 6} align="center">No change requests.</TableCell></TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      <Dialog open={open} onClose={() => !busy && setOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>New Customer Change Request</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ pt: 1 }}>
            <Grid size={12}>
              <TextField select fullWidth label="Customer" value={customerID || ''} onChange={(event) => setCustomerID(Number(event.target.value))}>
                {(customers.data?.customers ?? []).map((customer) => (
                  <MenuItem key={customer.id} value={customer.id}>{customer.customer_code} — {customer.full_name}</MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={12}>
              <TextField
                select
                fullWidth
                label="Request Type"
                value={type}
                onChange={(event) => { setType(event.target.value as CustomerChangeType); setRequestedValue('') }}
              >
                {Object.entries(labels).map(([value, label]) => <MenuItem key={value} value={value}>{label}</MenuItem>)}
              </TextField>
            </Grid>
            {type === 'BILLING_CYCLE' && (
              <Grid size={12}>
                <TextField fullWidth type="number" label="New billing day (1–31)" value={requestedValue} slotProps={{ htmlInput: { min: 1, max: 31 } }} onChange={(event) => setRequestedValue(event.target.value)} />
              </Grid>
            )}
            {type === 'PACKAGE' && (
              <Grid size={12}>
                <TextField select fullWidth label="New package" value={requestedValue} onChange={(event) => setRequestedValue(event.target.value)}>
                  {(options.data?.packages ?? []).map((item) => <MenuItem key={item.id} value={item.id}>{item.code} — {item.name}</MenuItem>)}
                </TextField>
              </Grid>
            )}
            {type === 'LINE_SHIFT' && (
              <Grid size={12}>
                <TextField select fullWidth label="Target router" value={requestedValue} onChange={(event) => setRequestedValue(event.target.value)}>
                  {(options.data?.routers ?? []).map((item) => <MenuItem key={item.id} value={item.id}>{item.code} — {item.name}</MenuItem>)}
                </TextField>
              </Grid>
            )}
            {type === 'CLOSE' && <Grid size={12}><Alert severity="warning">Approval will disconnect this customer's subscription and disable the MikroTik PPP account.</Alert></Grid>}
            <Grid size={12}>
              <TextField fullWidth multiline minRows={3} label="Reason" value={reason} onChange={(event) => setReason(event.target.value)} />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" disabled={busy || !customerID || !reason.trim() || !requestedValueValid} onClick={() => void submit()}>Submit Request</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={rejecting !== null} onClose={() => !busy && setRejecting(null)} fullWidth maxWidth="sm">
        <DialogTitle>Reject Request</DialogTitle>
        <DialogContent>
          <TextField autoFocus fullWidth multiline minRows={3} label="Rejection reason" value={rejectReason} onChange={(event) => setRejectReason(event.target.value)} sx={{ mt: 1 }} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRejecting(null)}>Cancel</Button>
          <Button color="error" variant="contained" disabled={busy || !rejectReason.trim()} onClick={() => void reject()}>Reject</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
