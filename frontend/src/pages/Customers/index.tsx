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
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'

import {
  createCustomer,
  getCustomers,
  type CreateCustomerRequest,
  type Customer,
} from '../../api/customers'
import { getAPIErrorMessage } from '../../api/errors'

const initialForm: CreateCustomerRequest = {
  full_name: '',
  mobile: '',
  father_name: '',
  mother_name: '',
  alt_mobile: '',
  email: '',
  nid: '',
  division: '',
  district: '',
  upazila: '',
  union: '',
  village: '',
  address: '',
  billing_day: 1,
}

function Customers() {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [form, setForm] =
    useState<CreateCustomerRequest>(initialForm)

  const loadCustomers = async () => {
    try {
      setLoading(true)
      setError('')

      const data = await getCustomers()

      setCustomers(data.customers)
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load customers.'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadCustomers()
  }, [])

  const filteredCustomers = useMemo(() => {
    const query = search.trim().toLowerCase()

    if (!query) {
      return customers
    }

    return customers.filter((customer) =>
      [
        customer.customer_code,
        customer.full_name,
        customer.mobile,
        customer.email,
        customer.status,
      ]
        .join(' ')
        .toLowerCase()
        .includes(query),
    )
  }, [customers, search])

  const handleChange = (
    field: keyof CreateCustomerRequest,
    value: string | number,
  ) => {
    setForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleCreate = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault()

    if (
      !form.full_name.trim() ||
      !form.mobile.trim()
    ) {
      return
    }

    try {
      setSaving(true)
      setError('')

      await createCustomer({
        ...form,
        full_name: form.full_name.trim(),
        mobile: form.mobile.trim(),
      })

      setForm(initialForm)
      setOpen(false)

      await loadCustomers()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to create customer.'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box>
      {/* Page Header */}
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
            sx={{
              fontWeight: 700,
            }}
          >
            Customers
          </Typography>

          <Typography
            variant="body1"
            color="text.secondary"
          >
            Manage ISP customers and their billing information.
          </Typography>
        </Box>

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setOpen(true)}
        >
          Add Customer
        </Button>
      </Box>

      {/* Error Message */}
      {error && (
        <Alert
          severity="error"
          sx={{ mb: 3 }}
          onClose={() => setError('')}
        >
          {error}
        </Alert>
      )}

      {/* Customer Card */}
      <Card>
        <CardContent>
          {/* Search / Refresh */}
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
              placeholder="Search customers..."
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
              onClick={() => void loadCustomers()}
              disabled={loading}
              title="Refresh"
            >
              <RefreshIcon />
            </IconButton>
          </Box>

          {/* Loading */}
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
          ) : filteredCustomers.length === 0 ? (
            /* Empty State */
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
                No customers found
              </Typography>

              <Typography
                variant="body2"
                color="text.secondary"
              >
                {search
                  ? 'Try a different search term.'
                  : 'Add your first customer to get started.'}
              </Typography>
            </Box>
          ) : (
            /* Customer Table */
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Code</TableCell>
                    <TableCell>Customer</TableCell>
                    <TableCell>Mobile</TableCell>
                    <TableCell>Email</TableCell>
                    <TableCell>Billing Day</TableCell>
                    <TableCell>Status</TableCell>
                  </TableRow>
                </TableHead>

                <TableBody>
                  {filteredCustomers.map((customer) => (
                    <TableRow
                      key={customer.id}
                      hover
                    >
                      <TableCell>
                        <Typography
                          sx={{
                            fontWeight: 600,
                          }}
                        >
                          {customer.customer_code}
                        </Typography>
                      </TableCell>

                      <TableCell>
                        {customer.full_name}
                      </TableCell>

                      <TableCell>
                        {customer.mobile}
                      </TableCell>

                      <TableCell>
                        {customer.email || '—'}
                      </TableCell>

                      <TableCell>
                        {customer.billing_day}
                      </TableCell>

                      <TableCell>
                        <Typography
                          component="span"
                          sx={{
                            fontWeight: 600,
                            color:
                              customer.status ===
                              'ACTIVE'
                                ? 'success.main'
                                : 'text.secondary',
                          }}
                        >
                          {customer.status}
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      {/* Add Customer Dialog */}
      <Dialog
        open={open}
        onClose={() =>
          !saving && setOpen(false)
        }
        fullWidth
        maxWidth="md"
      >
        <Box
          component="form"
          onSubmit={handleCreate}
        >
          <DialogTitle>
            Add Customer
          </DialogTitle>

          <DialogContent dividers>
            <Grid
              container
              spacing={2}
              sx={{
                pt: 1,
              }}
            >
              {/* Full Name */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  required
                  label="Full Name"
                  value={form.full_name}
                  onChange={(event) =>
                    handleChange(
                      'full_name',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Mobile */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  required
                  label="Mobile"
                  value={form.mobile}
                  onChange={(event) =>
                    handleChange(
                      'mobile',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Father Name */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  label="Father Name"
                  value={form.father_name}
                  onChange={(event) =>
                    handleChange(
                      'father_name',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Mother Name */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  label="Mother Name"
                  value={form.mother_name}
                  onChange={(event) =>
                    handleChange(
                      'mother_name',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Alternative Mobile */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  label="Alternative Mobile"
                  value={form.alt_mobile}
                  onChange={(event) =>
                    handleChange(
                      'alt_mobile',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Email */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  type="email"
                  label="Email"
                  value={form.email}
                  onChange={(event) =>
                    handleChange(
                      'email',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* NID */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  label="NID"
                  value={form.nid}
                  onChange={(event) =>
                    handleChange(
                      'nid',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Billing Day */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  type="number"
                  label="Billing Day"
                  value={form.billing_day}
                  onChange={(event) =>
                    handleChange(
                      'billing_day',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      min: 1,
                      max: 31,
                    },
                  }}
                />
              </Grid>

              {/* Division */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Division"
                  value={form.division}
                  onChange={(event) =>
                    handleChange(
                      'division',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* District */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="District"
                  value={form.district}
                  onChange={(event) =>
                    handleChange(
                      'district',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Upazila */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Upazila"
                  value={form.upazila}
                  onChange={(event) =>
                    handleChange(
                      'upazila',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Union */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  label="Union"
                  value={form.union}
                  onChange={(event) =>
                    handleChange(
                      'union',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Village */}
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  label="Village"
                  value={form.village}
                  onChange={(event) =>
                    handleChange(
                      'village',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Address */}
              <Grid size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  multiline
                  minRows={3}
                  label="Address"
                  value={form.address}
                  onChange={(event) =>
                    handleChange(
                      'address',
                      event.target.value,
                    )
                  }
                />
              </Grid>
            </Grid>
          </DialogContent>

          {/* Dialog Actions */}
          <DialogActions
            sx={{
              px: 3,
              py: 2,
            }}
          >
            <Button
              onClick={() => setOpen(false)}
              disabled={saving}
            >
              Cancel
            </Button>

            <Button
              type="submit"
              variant="contained"
              disabled={
                saving ||
                !form.full_name.trim() ||
                !form.mobile.trim()
              }
              startIcon={
                saving ? (
                  <CircularProgress size={18} />
                ) : (
                  <AddIcon />
                )
              }
            >
              {saving
                ? 'Creating...'
                : 'Create Customer'}
            </Button>
          </DialogActions>
        </Box>
      </Dialog>
    </Box>
  )
}

export default Customers
