import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import BlockIcon from '@mui/icons-material/Block'
import PrintIcon from '@mui/icons-material/Print'
import EditIcon from '@mui/icons-material/Edit'
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'
import axios from 'axios'

import { getCustomers, type Customer } from '../../api/customers'
import { getInvoices, type Invoice } from '../../api/invoices'
import {
  createPayment,
  voidPayment,
  getPayments,
  updatePayment,
  type CreatePaymentRequest,
  type Payment,
} from '../../api/payments'
import { getStoredUser } from '../../api/auth'

const paymentMethods = ['CASH', 'BKASH', 'NAGAD', 'ROCKET', 'BANK']
const today = () => new Date().toISOString().slice(0, 10)
const initialForm = (): CreatePaymentRequest => ({
  invoice_id: 0,
  payment_date: today(),
  amount: 0,
  method: 'CASH',
  transaction_id: '',
  reference: '',
  remarks: '',
})
const formatDate = (value: string) =>
  value ? new Date(value).toLocaleDateString() : '-'
const errorMessage = (error: unknown, fallback: string) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error || fallback
  }
  return fallback
}

function Payments() {
  const [payments, setPayments] = useState<Payment[]>([])
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [voiding, setVoiding] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Payment | null>(null)
  const [form, setForm] = useState<CreatePaymentRequest>(initialForm)
  const [voidOpen, setVoidOpen] = useState(false)
  const [voidingPayment, setVoidingPayment] = useState<Payment | null>(null)
  const isSuperadmin = getStoredUser()?.role === 'superadmin'

  const loadData = async () => {
    try {
      setLoading(true)
      setError('')
      const [paymentData, invoiceData, customerData] = await Promise.all([
        getPayments(),
        getInvoices(),
        getCustomers(),
      ])
      setPayments(paymentData)
      setInvoices(invoiceData)
      setCustomers(customerData.customers)
    } catch (error: unknown) {
      setError(errorMessage(error, 'Failed to load payment data.'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadData()
  }, [])

  const invoiceMap = useMemo(
    () => new Map(invoices.map((invoice) => [invoice.id, invoice])),
    [invoices],
  )
  const customerMap = useMemo(
    () => new Map(customers.map((customer) => [customer.id, customer])),
    [customers],
  )
  const filteredPayments = useMemo(() => {
    const query = search.trim().toLowerCase()
    return payments.filter((payment) => {
      const paymentDay = payment.payment_date.slice(0, 10)
      if (dateFrom && paymentDay < dateFrom) return false
      if (dateTo && paymentDay > dateTo) return false
      if (!query) return true
      const invoice = invoiceMap.get(payment.invoice_id)
      const customer = customerMap.get(payment.customer_id)
      return [
        payment.receipt_no,
        payment.method,
        payment.transaction_id,
        payment.reference,
        payment.status,
        invoice?.invoice_no,
        customer?.customer_code,
        customer?.full_name,
      ].join(' ').toLowerCase().includes(query)
    })
  }, [payments, search, dateFrom, dateTo, invoiceMap, customerMap])

  const collectionTotal = useMemo(
    () => filteredPayments.filter((payment) => payment.status === 'SUCCESS').reduce((total, payment) => total + payment.amount, 0),
    [filteredPayments],
  )
  const voidTotal = useMemo(
    () => filteredPayments.filter((payment) => payment.status === 'VOID').reduce((total, payment) => total + payment.amount, 0),
    [filteredPayments],
  )

  const invoiceLabel = (invoice: Invoice) => {
    const customer = customerMap.get(invoice.customer_id)
    return `${invoice.invoice_no} — ${customer?.full_name || customer?.customer_code || `Customer #${invoice.customer_id}`} — Due BDT ${invoice.due_amount.toLocaleString()}`
  }

  const openCreate = () => {
    setEditing(null)
    setForm(initialForm())
    setError('')
    setOpen(true)
  }
  const openEdit = (payment: Payment) => {
    setEditing(payment)
    setForm({
      invoice_id: payment.invoice_id,
      payment_date: payment.payment_date.slice(0, 10),
      amount: payment.amount,
      method: payment.method,
      transaction_id: payment.transaction_id,
      reference: payment.reference,
      remarks: payment.remarks,
    })
    setError('')
    setOpen(true)
  }
  const closeDialog = () => {
    if (!saving) setOpen(false)
  }
  const change = <K extends keyof CreatePaymentRequest>(
    key: K,
    value: CreatePaymentRequest[K],
  ) => setForm((current) => ({ ...current, [key]: value }))

  const selectInvoice = (invoiceId: number) => {
    const invoice = invoiceMap.get(invoiceId)
    setForm((current) => ({
      ...current,
      invoice_id: invoiceId,
      amount: invoice?.due_amount ?? current.amount,
    }))
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    try {
      setSaving(true)
      setError('')
      if (editing) await updatePayment(editing.id, form)
      else await createPayment(form)
      setOpen(false)
      await loadData()
    } catch (error: unknown) {
      setError(errorMessage(error, 'Failed to save payment.'))
    } finally {
      setSaving(false)
    }
  }

  const confirmVoid = (payment: Payment) => {
    setVoidingPayment(payment)
    setVoidOpen(true)
  }
  const handleVoid = async () => {
    if (!voidingPayment) return
    try {
      setVoiding(true)
      setError('')
      await voidPayment(voidingPayment.id)
      setVoidOpen(false)
      setVoidingPayment(null)
      await loadData()
    } catch (error: unknown) {
      setError(errorMessage(error, 'Failed to void payment.'))
    } finally {
      setVoiding(false)
    }
  }

  const printReceipt = (payment: Payment) => {
    const invoice = invoiceMap.get(payment.invoice_id)
    const customer = customerMap.get(payment.customer_id)
    const receipt = window.open('', '_blank', 'width=720,height=800')
    if (!receipt) return
    const safe = (value: string) => value.replace(/[&<>"']/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character] || character)
    receipt.document.write(`<!doctype html><html><head><title>${safe(payment.receipt_no)}</title><style>body{font-family:Arial,sans-serif;max-width:680px;margin:40px auto;color:#111}h1{margin-bottom:4px}.row{display:flex;justify-content:space-between;padding:10px 0;border-bottom:1px solid #ddd}.amount{font-size:24px;font-weight:bold;margin:24px 0}@media print{button{display:none}}</style></head><body><h1>TS-CLOUD</h1><p>Payment Receipt</p><div class="row"><span>Receipt</span><strong>${safe(payment.receipt_no)}</strong></div><div class="row"><span>Customer</span><strong>${safe(customer?.full_name || `Customer #${payment.customer_id}`)}</strong></div><div class="row"><span>Invoice</span><strong>${safe(invoice?.invoice_no || `Invoice #${payment.invoice_id}`)}</strong></div><div class="row"><span>Date</span><strong>${safe(formatDate(payment.payment_date))}</strong></div><div class="row"><span>Method</span><strong>${safe(payment.method)}</strong></div><div class="amount">Received: BDT ${payment.amount.toLocaleString()}</div><button onclick="window.print()">Print Receipt</button></body></html>`)
    receipt.document.close()
  }

  const nonCash = form.method !== 'CASH'
  const selectableInvoices = editing
    ? invoices
    : invoices.filter((invoice) => invoice.due_amount > 0)

  return (
    <Box>
      <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, justifyContent: 'space-between', alignItems: { xs: 'stretch', sm: 'center' }, gap: 2, mb: 3 }}>
        <Box>
          <Typography variant="h4" component="h1" sx={{ fontWeight: 700 }}>Payments</Typography>
          <Typography color="text.secondary">Record and manage customer invoice payments.</Typography>
        </Box>
        <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate} disabled={loading}>Add Payment</Button>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError('')}>{error}</Alert>}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 4 }}><Card><CardContent><Typography color="text.secondary">Successful Collection</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>BDT {collectionTotal.toLocaleString()}</Typography></CardContent></Card></Grid>
        <Grid size={{ xs: 12, sm: 4 }}><Card><CardContent><Typography color="text.secondary">Voided Amount</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>BDT {voidTotal.toLocaleString()}</Typography></CardContent></Card></Grid>
        <Grid size={{ xs: 12, sm: 4 }}><Card><CardContent><Typography color="text.secondary">Records</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>{filteredPayments.length}</Typography></CardContent></Card></Grid>
      </Grid>

      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, justifyContent: 'space-between', gap: 2, mb: 2 }}>
            <TextField size="small" placeholder="Search payments..." value={search} onChange={(event) => setSearch(event.target.value)} sx={{ maxWidth: 400, width: '100%' }} slotProps={{ input: { startAdornment: <SearchIcon sx={{ mr: 1 }} /> } }} />
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              <TextField size="small" type="date" label="From" value={dateFrom} onChange={(event) => setDateFrom(event.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
              <TextField size="small" type="date" label="To" value={dateTo} onChange={(event) => setDateTo(event.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
              <IconButton onClick={() => void loadData()} disabled={loading} title="Refresh"><RefreshIcon /></IconButton>
            </Box>
          </Box>

          {loading ? (
            <Box sx={{ py: 8, display: 'flex', justifyContent: 'center' }}><CircularProgress /></Box>
          ) : filteredPayments.length === 0 ? (
            <Box sx={{ py: 8, textAlign: 'center' }}>
              <Typography variant="h6" color="text.secondary">No payments found</Typography>
              <Typography variant="body2" color="text.secondary">{search ? 'Try a different search term.' : 'Add your first payment to get started.'}</Typography>
            </Box>
          ) : (
            <TableContainer sx={{ overflowX: 'auto' }}>
              <Table sx={{ minWidth: 1050 }}>
                <TableHead><TableRow>
                  <TableCell>Receipt</TableCell><TableCell>Invoice</TableCell><TableCell>Customer</TableCell><TableCell>Date</TableCell><TableCell>Amount</TableCell><TableCell>Method</TableCell><TableCell>Transaction ID</TableCell><TableCell>Status</TableCell><TableCell align="right">Actions</TableCell>
                </TableRow></TableHead>
                <TableBody>{filteredPayments.map((payment) => {
                  const invoice = invoiceMap.get(payment.invoice_id)
                  const customer = customerMap.get(payment.customer_id)
                  return <TableRow key={payment.id} hover>
                    <TableCell><Typography sx={{ fontWeight: 600 }}>{payment.receipt_no || `#${payment.id}`}</Typography></TableCell>
                    <TableCell>{invoice?.invoice_no || `#${payment.invoice_id}`}</TableCell>
                    <TableCell>{customer?.full_name || customer?.customer_code || `#${payment.customer_id}`}</TableCell>
                    <TableCell>{formatDate(payment.payment_date)}</TableCell>
                    <TableCell>BDT {payment.amount.toLocaleString()}</TableCell>
                    <TableCell>{payment.method}</TableCell>
                    <TableCell>{payment.transaction_id || '-'}</TableCell>
                    <TableCell><Typography component="span" sx={{ fontWeight: 600, color: payment.status === 'SUCCESS' ? 'success.main' : 'text.secondary' }}>{payment.status}</Typography></TableCell>
                    <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                      <IconButton title="Print Receipt" onClick={() => printReceipt(payment)}><PrintIcon /></IconButton>
                      <IconButton color="primary" title="Edit" disabled={payment.status !== 'SUCCESS'} onClick={() => openEdit(payment)}><EditIcon /></IconButton>
                      {isSuperadmin && payment.status === 'SUCCESS' && <IconButton color="error" title="Void" onClick={() => confirmVoid(payment)}><BlockIcon /></IconButton>}
                    </TableCell>
                  </TableRow>
                })}</TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onClose={closeDialog} fullWidth maxWidth="md">
        <Box component="form" onSubmit={submit}>
          <DialogTitle>{editing ? 'Edit Payment' : 'Add Payment'}</DialogTitle>
          <DialogContent dividers>
            <Grid container spacing={2} sx={{ pt: 1 }}>
              <Grid size={{ xs: 12 }}><TextField fullWidth required select label="Invoice" value={form.invoice_id || ''} onChange={(event) => selectInvoice(Number(event.target.value))} disabled={Boolean(editing)}>
                {selectableInvoices.map((invoice) => <MenuItem key={invoice.id} value={invoice.id}>{invoiceLabel(invoice)}</MenuItem>)}
              </TextField></Grid>
              <Grid size={{ xs: 12, md: 6 }}><TextField fullWidth required type="date" label="Payment Date" value={form.payment_date} onChange={(event) => change('payment_date', event.target.value)} slotProps={{ inputLabel: { shrink: true } }} /></Grid>
              <Grid size={{ xs: 12, md: 6 }}><TextField fullWidth required type="number" label="Amount" value={form.amount} onChange={(event) => change('amount', Number(event.target.value))} slotProps={{ htmlInput: { min: 0.01, step: '0.01' } }} /></Grid>
              <Grid size={{ xs: 12, md: 6 }}><TextField fullWidth required select label="Method" value={form.method} onChange={(event) => change('method', event.target.value)}>{paymentMethods.map((method) => <MenuItem key={method} value={method}>{method}</MenuItem>)}</TextField></Grid>
              <Grid size={{ xs: 12, md: 6 }}><TextField fullWidth required={nonCash} label="Transaction ID" value={form.transaction_id} onChange={(event) => change('transaction_id', event.target.value)} helperText={nonCash ? 'Required for non-cash payments' : 'Optional for cash payments'} /></Grid>
              <Grid size={{ xs: 12 }}><TextField fullWidth label="Reference" value={form.reference} onChange={(event) => change('reference', event.target.value)} /></Grid>
              <Grid size={{ xs: 12 }}><TextField fullWidth multiline minRows={3} label="Remarks" value={form.remarks} onChange={(event) => change('remarks', event.target.value)} /></Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ px: 3, py: 2 }}>
            <Button onClick={closeDialog} disabled={saving}>Cancel</Button>
            <Button type="submit" variant="contained" disabled={saving || !form.invoice_id || form.amount <= 0 || (nonCash && !form.transaction_id.trim())} startIcon={saving ? <CircularProgress size={18} /> : <AddIcon />}>{saving ? 'Saving...' : editing ? 'Update Payment' : 'Create Payment'}</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={voidOpen} onClose={() => !voiding && setVoidOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Void Payment</DialogTitle>
        <DialogContent><Typography>Void <strong>{voidingPayment?.receipt_no || `payment #${voidingPayment?.id}`}</strong>? The record will remain in financial history and its amount will be restored to the invoice due.</Typography></DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setVoidOpen(false)} disabled={voiding}>Cancel</Button>
          <Button variant="contained" color="error" onClick={() => void handleVoid()} disabled={voiding} startIcon={voiding ? <CircularProgress size={18} /> : <BlockIcon />}>{voiding ? 'Voiding...' : 'Void Payment'}</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default Payments
