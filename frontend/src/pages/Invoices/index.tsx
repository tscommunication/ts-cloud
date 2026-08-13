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
import DeleteIcon from '@mui/icons-material/Delete'
import EditIcon from '@mui/icons-material/Edit'
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'

import {
  getCustomers,
  type Customer,
} from '../../api/customers'
import {
  getPackages,
  type Package,
} from '../../api/packages'
import {
  getSubscriptions,
  type Subscription,
} from '../../api/subscriptions'
import {
  createInvoice,
  deleteInvoice,
  getInvoices,
  updateInvoice,
  type CreateInvoiceRequest,
  type Invoice,
} from '../../api/invoices'
import { getAPIErrorMessage } from '../../api/errors'

const getToday = () =>
  new Date().toISOString().slice(0, 10)

const getDueDate = () => {
  const date = new Date()
  date.setDate(date.getDate() + 7)

  return date.toISOString().slice(0, 10)
}

const createInitialForm = (): CreateInvoiceRequest => ({
  subscription_id: 0,
  bill_month: new Date().getMonth() + 1,
  bill_year: new Date().getFullYear(),
  issue_date: getToday(),
  due_date: getDueDate(),
  package_price: 0,
  discount: 0,
  vat: 0,
  remarks: '',
})

const toDateInput = (value: string) => {
  return value ? value.slice(0, 10) : getToday()
}

const formatDate = (value: string) => {
  if (!value) {
    return '-'
  }

  return new Date(value).toLocaleDateString()
}

function Invoices() {
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [subscriptions, setSubscriptions] = useState<
    Subscription[]
  >([])
  const [customers, setCustomers] = useState<Customer[]>([])
  const [packages, setPackages] = useState<Package[]>([])

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')

  const [open, setOpen] = useState(false)
  const [editingInvoice, setEditingInvoice] =
    useState<Invoice | null>(null)
  const [form, setForm] =
    useState<CreateInvoiceRequest>(createInitialForm)

  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deletingInvoice, setDeletingInvoice] =
    useState<Invoice | null>(null)

  const loadData = async () => {
    try {
      setLoading(true)
      setError('')

      const [
        invoiceData,
        subscriptionData,
        customerData,
        packageData,
      ] = await Promise.all([
        getInvoices(),
        getSubscriptions(),
        getCustomers(),
        getPackages(),
      ])

      setInvoices(invoiceData)
      setSubscriptions(subscriptionData.subscriptions)
      setCustomers(customerData.customers)
      setPackages(packageData.packages)
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load invoice data.'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadData()
  }, [])

  const customerMap = useMemo(
    () =>
      new Map(
        customers.map((customer) => [
          customer.id,
          customer,
        ]),
      ),
    [customers],
  )

  const packageMap = useMemo(
    () =>
      new Map(
        packages.map((pkg) => [pkg.id, pkg]),
      ),
    [packages],
  )

  const subscriptionMap = useMemo(
    () =>
      new Map(
        subscriptions.map((subscription) => [
          subscription.id,
          subscription,
        ]),
      ),
    [subscriptions],
  )

  const filteredInvoices = useMemo(() => {
    const query = search.trim().toLowerCase()

    if (!query) {
      return invoices
    }

    return invoices.filter((invoice) => {
      const customer = customerMap.get(invoice.customer_id)
      const pkg = packageMap.get(invoice.package_id)

      return [
        invoice.invoice_no,
        invoice.status,
        invoice.remarks,
        customer?.customer_code,
        customer?.full_name,
        pkg?.package_code,
        pkg?.name,
      ]
        .join(' ')
        .toLowerCase()
        .includes(query)
    })
  }, [invoices, search, customerMap, packageMap])

  const handleOpenCreate = () => {
    setEditingInvoice(null)
    setForm(createInitialForm())
    setError('')
    setOpen(true)
  }

  const handleOpenEdit = (invoice: Invoice) => {
    setEditingInvoice(invoice)
    setError('')

    setForm({
      subscription_id: invoice.subscription_id,
      bill_month: invoice.bill_month,
      bill_year: invoice.bill_year,
      issue_date: toDateInput(invoice.issue_date),
      due_date: toDateInput(invoice.due_date),
      package_price: invoice.package_price,
      discount: invoice.discount,
      vat: invoice.vat,
      remarks: invoice.remarks,
    })

    setOpen(true)
  }

  const handleCloseDialog = () => {
    if (saving) {
      return
    }

    setOpen(false)
    setEditingInvoice(null)
    setForm(createInitialForm())
  }

  const handleChange = (
    field: keyof CreateInvoiceRequest,
    value: string | number,
  ) => {
    setForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleSubscriptionChange = (
    subscriptionId: number,
  ) => {
    const subscription = subscriptionMap.get(subscriptionId)
    const pkg = subscription
      ? packageMap.get(subscription.package_id)
      : undefined

    setForm((current) => ({
      ...current,
      subscription_id: subscriptionId,
      package_price: pkg?.price ?? current.package_price,
    }))
  }

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault()

    if (!form.subscription_id) {
      return
    }

    try {
      setSaving(true)
      setError('')

      const payload: CreateInvoiceRequest = {
        ...form,
        remarks: form.remarks.trim(),
      }

      if (editingInvoice) {
        await updateInvoice(editingInvoice.id, payload)
      } else {
        await createInvoice(payload)
      }

      setOpen(false)
      setEditingInvoice(null)
      setForm(createInitialForm())

      await loadData()
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          `Failed to ${editingInvoice ? 'update' : 'create'} invoice.`,
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const handleOpenDelete = (invoice: Invoice) => {
    setDeletingInvoice(invoice)
    setDeleteOpen(true)
  }

  const handleCloseDelete = () => {
    if (deleting) {
      return
    }

    setDeleteOpen(false)
    setDeletingInvoice(null)
  }

  const handleDelete = async () => {
    if (!deletingInvoice) {
      return
    }

    try {
      setDeleting(true)
      setError('')

      await deleteInvoice(deletingInvoice.id)

      setDeleteOpen(false)
      setDeletingInvoice(null)

      await loadData()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to delete invoice.'))
    } finally {
      setDeleting(false)
    }
  }

  const getCustomerName = (customerId: number) => {
    const customer = customerMap.get(customerId)

    return customer
      ? customer.full_name
      : `Customer #${customerId}`
  }

  const getPackageName = (packageId: number) => {
    const pkg = packageMap.get(packageId)

    return pkg ? pkg.name : `Package #${packageId}`
  }

  const getSubscriptionLabel = (
    subscription: Subscription,
  ) => {
    const customer = customerMap.get(
      subscription.customer_id,
    )

    return `${subscription.subscription_code} - ${
      customer?.full_name || 'Unknown Customer'
    }`
  }

  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: {
            xs: 'column',
            sm: 'row',
          },
          justifyContent: {
            xs: 'flex-start',
            sm: 'space-between',
          },
          alignItems: {
            xs: 'stretch',
            sm: 'center',
          },
          gap: 2,
          mb: 3,
        }}
      >
        <Box>
          <Typography
            variant="h4"
            component="h1"
            sx={{ fontWeight: 700 }}
          >
            Invoices
          </Typography>

          <Typography
            variant="body1"
            color="text.secondary"
          >
            Create and manage customer billing invoices.
          </Typography>
        </Box>

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleOpenCreate}
          disabled={loading}
        >
          Add Invoice
        </Button>
      </Box>

      {error && (
        <Alert
          severity="error"
          sx={{ mb: 3 }}
          onClose={() => setError('')}
        >
          {error}
        </Alert>
      )}

      <Card>
        <CardContent>
          <Box
            sx={{
              display: 'flex',
              flexDirection: {
                xs: 'column',
                sm: 'row',
              },
              justifyContent: {
                xs: 'flex-start',
                sm: 'space-between',
              },
              alignItems: {
                xs: 'stretch',
                sm: 'center',
              },
              gap: 2,
              mb: 2,
            }}
          >
            <TextField
              size="small"
              placeholder="Search invoices..."
              value={search}
              onChange={(event) =>
                setSearch(event.target.value)
              }
              sx={{
                maxWidth: 400,
                width: '100%',
              }}
              slotProps={{
                input: {
                  startAdornment: (
                    <SearchIcon sx={{ mr: 1 }} />
                  ),
                },
              }}
            />

            <IconButton
              onClick={() => void loadData()}
              disabled={loading}
              title="Refresh"
            >
              <RefreshIcon />
            </IconButton>
          </Box>

          {loading ? (
            <Box
              sx={{
                py: 8,
                display: 'flex',
                justifyContent: 'center',
              }}
            >
              <CircularProgress />
            </Box>
          ) : filteredInvoices.length === 0 ? (
            <Box
              sx={{
                py: 8,
                textAlign: 'center',
              }}
            >
              <Typography
                variant="h6"
                color="text.secondary"
              >
                No invoices found
              </Typography>

              <Typography
                variant="body2"
                color="text.secondary"
              >
                {search
                  ? 'Try a different search term.'
                  : 'Add your first invoice to get started.'}
              </Typography>
            </Box>
          ) : (
            <TableContainer sx={{ overflowX: 'auto' }}>
              <Table sx={{ minWidth: 1100 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Invoice</TableCell>
                    <TableCell>Customer</TableCell>
                    <TableCell>Package</TableCell>
                    <TableCell>Bill Period</TableCell>
                    <TableCell>Due Date</TableCell>
                    <TableCell>Amount</TableCell>
                    <TableCell>Due</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell
                      align="right"
                      sx={{
                        minWidth: 112,
                        whiteSpace: 'nowrap',
                      }}
                    >
                      Actions
                    </TableCell>
                  </TableRow>
                </TableHead>

                <TableBody>
                  {filteredInvoices.map((invoice) => (
                    <TableRow key={invoice.id} hover>
                      <TableCell>
                        <Typography
                          sx={{ fontWeight: 600 }}
                        >
                          {invoice.invoice_no}
                        </Typography>
                      </TableCell>

                      <TableCell>
                        {getCustomerName(invoice.customer_id)}
                      </TableCell>

                      <TableCell>
                        {getPackageName(invoice.package_id)}
                      </TableCell>

                      <TableCell>
                        {invoice.bill_month}/{invoice.bill_year}
                      </TableCell>

                      <TableCell>
                        {formatDate(invoice.due_date)}
                      </TableCell>

                      <TableCell>
                        BDT {invoice.total_amount.toLocaleString()}
                      </TableCell>

                      <TableCell>
                        BDT {invoice.due_amount.toLocaleString()}
                      </TableCell>

                      <TableCell>
                        <Typography
                          component="span"
                          sx={{
                            fontWeight: 600,
                            color:
                              invoice.status === 'PAID'
                                ? 'success.main'
                                : invoice.status === 'UNPAID'
                                  ? 'warning.main'
                                  : 'text.secondary',
                          }}
                        >
                          {invoice.status}
                        </Typography>
                      </TableCell>

                      <TableCell
                        align="right"
                        sx={{
                          minWidth: 112,
                          whiteSpace: 'nowrap',
                        }}
                      >
                        <IconButton
                          color="primary"
                          title="Edit"
                          onClick={() =>
                            handleOpenEdit(invoice)
                          }
                        >
                          <EditIcon />
                        </IconButton>

                        <IconButton
                          color="error"
                          title="Delete"
                          onClick={() =>
                            handleOpenDelete(invoice)
                          }
                        >
                          <DeleteIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={open}
        onClose={handleCloseDialog}
        fullWidth
        maxWidth="md"
      >
        <Box component="form" onSubmit={handleSubmit}>
          <DialogTitle>
            {editingInvoice
              ? 'Edit Invoice'
              : 'Add Invoice'}
          </DialogTitle>

          <DialogContent dividers>
            <Grid
              container
              spacing={2}
              sx={{ pt: 1 }}
            >
              <Grid size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  required
                  select
                  label="Subscription"
                  value={form.subscription_id || ''}
                  onChange={(event) =>
                    handleSubscriptionChange(
                      Number(event.target.value),
                    )
                  }
                >
                  {subscriptions.map((subscription) => (
                    <MenuItem
                      key={subscription.id}
                      value={subscription.id}
                    >
                      {getSubscriptionLabel(subscription)}
                    </MenuItem>
                  ))}
                </TextField>
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  label="Bill Month"
                  value={form.bill_month}
                  onChange={(event) =>
                    handleChange(
                      'bill_month',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      min: 1,
                      max: 12,
                    },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  label="Bill Year"
                  value={form.bill_year}
                  onChange={(event) =>
                    handleChange(
                      'bill_year',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      min: 2020,
                    },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  label="Package Price"
                  value={form.package_price}
                  onChange={(event) =>
                    handleChange(
                      'package_price',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      min: 0,
                      step: '0.01',
                    },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  type="date"
                  label="Issue Date"
                  value={form.issue_date}
                  onChange={(event) =>
                    handleChange(
                      'issue_date',
                      event.target.value,
                    )
                  }
                  slotProps={{
                    inputLabel: { shrink: true },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  type="date"
                  label="Due Date"
                  value={form.due_date}
                  onChange={(event) =>
                    handleChange(
                      'due_date',
                      event.target.value,
                    )
                  }
                  slotProps={{
                    inputLabel: { shrink: true },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  type="number"
                  label="Discount"
                  value={form.discount}
                  onChange={(event) =>
                    handleChange(
                      'discount',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      min: 0,
                      step: '0.01',
                    },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  type="number"
                  label="VAT"
                  value={form.vat}
                  onChange={(event) =>
                    handleChange(
                      'vat',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      min: 0,
                      step: '0.01',
                    },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  multiline
                  minRows={3}
                  label="Remarks"
                  value={form.remarks}
                  onChange={(event) =>
                    handleChange(
                      'remarks',
                      event.target.value,
                    )
                  }
                />
              </Grid>
            </Grid>
          </DialogContent>

          <DialogActions sx={{ px: 3, py: 2 }}>
            <Button
              onClick={handleCloseDialog}
              disabled={saving}
            >
              Cancel
            </Button>

            <Button
              type="submit"
              variant="contained"
              disabled={saving || !form.subscription_id}
              startIcon={
                saving ? (
                  <CircularProgress size={18} />
                ) : (
                  <AddIcon />
                )
              }
            >
              {saving
                ? 'Saving...'
                : editingInvoice
                  ? 'Update Invoice'
                  : 'Create Invoice'}
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog
        open={deleteOpen}
        onClose={handleCloseDelete}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>Delete Invoice</DialogTitle>

        <DialogContent>
          <Typography>
            Delete{' '}
            <strong>{deletingInvoice?.invoice_no}</strong>?
            This action cannot be undone.
          </Typography>
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleCloseDelete}
            disabled={deleting}
          >
            Cancel
          </Button>

          <Button
            variant="contained"
            color="error"
            onClick={() => void handleDelete()}
            disabled={deleting}
            startIcon={
              deleting ? (
                <CircularProgress size={18} />
              ) : (
                <DeleteIcon />
              )
            }
          >
            {deleting ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default Invoices
