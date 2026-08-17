import { useCallback, useEffect, useState } from 'react'
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
  Divider,
  Grid,
  IconButton,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'

import AddIcon from '@mui/icons-material/Add'
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'
import EditIcon from '@mui/icons-material/Edit'
import ToggleOffIcon from '@mui/icons-material/ToggleOff'
import ToggleOnIcon from '@mui/icons-material/ToggleOn'
import VisibilityIcon from '@mui/icons-material/Visibility'
import ArchiveIcon from '@mui/icons-material/Archive'

import {
  createCustomer,
  archiveCustomer,
  getCustomers,
  getCustomerSummary,
  getCustomerLedger,
  updateCustomer,
  updateCustomerStatus,
  type CreateCustomerRequest,
  type Customer,
  type CustomerSummary,
  type CustomerLedgerEntry,
} from '../../api/customers'
import { getAPIErrorMessage } from '../../api/errors'
import { getStoredUser } from '../../api/auth'
import { getAgents, getPOPs, type Agent, type POP } from '../../api/distribution'
import {
  getDivisions,
  getDistricts,
  getUpazilas,
  type Division,
  type District,
  type Upazila,
} from '../../api/locations'

const initialForm: CreateCustomerRequest = {
  full_name: '',
  mobile: '',
  father_name: '',
  mother_name: '',
  alt_mobile: '',
  email: '',
  nid: '',
  country: 'Bangladesh',
  division: '',
  district: '',
  upazila: '',
  post_office: '',
  postal_code: '',
  road_or_area: '',
  village_or_holding: '',
  union: '',
  village: '',
  address: '',
  billing_day: 1,
}

const bangladeshMobileRegex = /^01[3-9][0-9]{8}$/
const customerNIDRegex = /^[0-9]{10,17}$/

const isValidBangladeshMobile = (value: string) =>
  bangladeshMobileRegex.test(value.trim())

const isValidCustomerNID = (value: string) =>
  customerNIDRegex.test(value.trim())

function Customers() {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'ACTIVE' | 'INACTIVE' | 'ARCHIVED' | ''>('')
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [open, setOpen] = useState(false)
  const [editingCustomer, setEditingCustomer] = useState<Customer | null>(null)
  const [viewingCustomer, setViewingCustomer] = useState<Customer | null>(null)
  const [summary, setSummary] = useState<CustomerSummary | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)
  const [ledger, setLedger] = useState<CustomerLedgerEntry[]>([])
  const [archivingCustomer, setArchivingCustomer] = useState<Customer | null>(null)
  const isSuperadmin = getStoredUser()?.role === 'superadmin'
  const isAgent = getStoredUser()?.role === 'agent'
  const [form, setForm] =
    useState<CreateCustomerRequest>(initialForm)
  const [pops, setPOPs] = useState<POP[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [divisions, setDivisions] = useState<Division[]>([])
  const [districts, setDistricts] = useState<District[]>([])
  const [upazilas, setUpazilas] = useState<Upazila[]>([])
  const [locationLoading, setLocationLoading] = useState(false)

  const loadCustomers = useCallback(async () => {
    try {
      setLoading(true)
      setError('')

      const data = await getCustomers({
        search: debouncedSearch || undefined,
        status: statusFilter,
        page: page + 1,
        page_size: pageSize,
      })

      setCustomers(data.customers)
      setTotal(data.count)
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load customers.'))
    } finally {
      setLoading(false)
    }
  }, [debouncedSearch, page, pageSize, statusFilter])

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedSearch(search.trim()), 350)
    return () => window.clearTimeout(timer)
  }, [search])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadCustomers()
  }, [loadCustomers])

  useEffect(() => {
    if (isAgent) return

    const loadDistribution = async () => {
      try {
        const [popRows, agentRows] = await Promise.all([getPOPs(), getAgents()])
        setPOPs(popRows)
        setAgents(agentRows)
      } catch (error: unknown) {
        setError(getAPIErrorMessage(error, 'Failed to load POP and agent options.'))
      }
    }
    void loadDistribution()
  }, [isAgent])

  useEffect(() => {
    const loadDivisions = async () => {
      try {
        const rows = await getDivisions()
        setDivisions(rows)
      } catch (error: unknown) {
        setError(
          getAPIErrorMessage(
            error,
            'Failed to load division options.',
          ),
        )
      }
    }

    void loadDivisions()
  }, [])

  const handleDivisionChange = async (divisionName: string) => {
    setForm((current) => ({
      ...current,
      division: divisionName,
      district: '',
      upazila: '',
      post_office: '',
      postal_code: '',
    }))
    setDistricts([])
    setUpazilas([])

    const selected = divisions.find(
      (item) => item.name === divisionName,
    )
    if (!selected) return

    try {
      setLocationLoading(true)
      const rows = await getDistricts(selected.id)
      setDistricts(rows)
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          'Failed to load district options.',
        ),
      )
    } finally {
      setLocationLoading(false)
    }
  }

  const handleDistrictChange = async (districtName: string) => {
    setForm((current) => ({
      ...current,
      district: districtName,
      upazila: '',
      post_office: '',
      postal_code: '',
    }))
    setUpazilas([])

    const selected = districts.find(
      (item) => item.name === districtName,
    )
    if (!selected) return

    try {
      setLocationLoading(true)
      const rows = await getUpazilas(selected.id)
      setUpazilas(rows)
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          'Failed to load upazila options.',
        ),
      )
    } finally {
      setLocationLoading(false)
    }
  }

  const handleUpazilaChange = (upazilaName: string) => {
    setForm((current) => ({
      ...current,
      upazila: upazilaName,
      post_office: '',
      postal_code: '',
    }))
  }

  const handleChange = (
    field: keyof CreateCustomerRequest,
    value: string | number | undefined,
  ) => {
    setForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const openCreateDialog = () => {
    setEditingCustomer(null)
    setForm(initialForm)
    setOpen(true)
  }

  const openEditDialog = async (customer: Customer) => {
    setEditingCustomer(customer)
    setDistricts([])
    setUpazilas([])

    setForm({
      full_name: customer.full_name,
      mobile: customer.mobile,
      father_name: customer.father_name,
      mother_name: customer.mother_name,
      alt_mobile: customer.alt_mobile,
      email: customer.email,
      nid: customer.nid,
      country: customer.country,
      division: customer.division,
      district: customer.district,
      upazila: customer.upazila,
      post_office: customer.post_office,
      postal_code: customer.postal_code,
      road_or_area: customer.road_or_area,
      village_or_holding: customer.village_or_holding,
      union: customer.union,
      village: customer.village,
      address: customer.address,
      billing_day: customer.billing_day,
      pop_id: customer.pop_id,
      agent_id: customer.agent_id,
    })

    setOpen(true)

    const selectedDivision = divisions.find(
      (item) => item.name === customer.division,
    )

    if (!selectedDivision) return

    try {
      setLocationLoading(true)

      const districtRows = await getDistricts(
        selectedDivision.id,
      )
      setDistricts(districtRows)

      const selectedDistrict = districtRows.find(
        (item) => item.name === customer.district,
      )

      if (!selectedDistrict) return

      const upazilaRows = await getUpazilas(
        selectedDistrict.id,
      )
      setUpazilas(upazilaRows)

    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          'Failed to load saved customer location options.',
        ),
      )
    } finally {
      setLocationLoading(false)
    }
  }

  const openDetailDialog = async (customer: Customer) => {
    setViewingCustomer(customer)
    setSummary(null)
    setLedger([])
    setSummaryLoading(true)
    try {
      const [summaryData, ledgerData] = await Promise.all([
        getCustomerSummary(customer.id),
        getCustomerLedger(customer.id),
      ])
      setSummary(summaryData)
      setLedger(ledgerData)
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load customer summary.'))
    } finally {
      setSummaryLoading(false)
    }
  }

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault()

    const mobile = form.mobile.trim()
    const altMobile = form.alt_mobile?.trim() ?? ''
    const nid = form.nid?.trim() ?? ''

    if (!form.full_name.trim()) {
      setError('Full Name is required.')
      return
    }

    if (!isValidBangladeshMobile(mobile)) {
      setError('Mobile must be a valid 11-digit Bangladesh mobile number starting with 013-019.')
      return
    }

    if (altMobile && !isValidBangladeshMobile(altMobile)) {
      setError('Alternative Mobile must be a valid 11-digit Bangladesh mobile number starting with 013-019.')
      return
    }

    if (!isValidCustomerNID(nid)) {
      setError('NID is required and must contain only 10 to 17 digits.')
      return
    }

    try {
      setSaving(true)
      setError('')

      const payload = {
        ...form,
        full_name: form.full_name.trim(),
        mobile,
        alt_mobile: altMobile,
        nid,
      }

      if (editingCustomer) await updateCustomer(editingCustomer.id, payload)
      else await createCustomer(payload)

      setForm(initialForm)
      setEditingCustomer(null)
      setOpen(false)

      await loadCustomers()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to create customer.'))
    } finally {
      setSaving(false)
    }
  }

  const toggleStatus = async (customer: Customer) => {
    try {
      setError('')
      await updateCustomerStatus(
        customer.id,
        customer.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE',
      )
      await loadCustomers()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to update customer status.'))
    }
  }

  const confirmArchive = async () => {
    if (!archivingCustomer) return
    try {
      setSaving(true)
      setError('')
      await archiveCustomer(archivingCustomer.id)
      setArchivingCustomer(null)
      await loadCustomers()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to archive customer.'))
      setArchivingCustomer(null)
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

        {!isAgent && <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openCreateDialog}
        >
          Add Customer
        </Button>}
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
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(0)
              }}
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

            <TextField
              select
              size="small"
              label="Status"
              value={statusFilter}
              onChange={(event) => {
                setStatusFilter(event.target.value as 'ACTIVE' | 'INACTIVE' | 'ARCHIVED' | '')
                setPage(0)
              }}
              sx={{ minWidth: 150 }}
            >
              <MenuItem value="">All statuses</MenuItem>
              <MenuItem value="ACTIVE">Active</MenuItem>
              <MenuItem value="INACTIVE">Inactive</MenuItem>
              <MenuItem value="ARCHIVED">Archived</MenuItem>
            </TextField>

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
          ) : customers.length === 0 ? (
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
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>

                <TableBody>
                  {customers.map((customer) => (
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

                      <TableCell align="right">
                        <Tooltip title="View customer">
                          <IconButton onClick={() => void openDetailDialog(customer)}>
                            <VisibilityIcon />
                          </IconButton>
                        </Tooltip>
                        {!isAgent && <Tooltip title="Edit customer">
                          <IconButton
                            onClick={() => openEditDialog(customer)}
                            disabled={customer.status === 'ARCHIVED'}
                          >
                            <EditIcon />
                          </IconButton>
                        </Tooltip>}
                        {!isAgent && <Tooltip title={customer.status === 'ACTIVE' ? 'Deactivate customer' : 'Activate customer'}>
                          <IconButton
                            color={customer.status === 'ACTIVE' ? 'warning' : 'success'}
                            onClick={() => void toggleStatus(customer)}
                            disabled={customer.status === 'ARCHIVED'}
                          >
                            {customer.status === 'ACTIVE' ? <ToggleOffIcon /> : <ToggleOnIcon />}
                          </IconButton>
                        </Tooltip>}
                        {isSuperadmin && customer.status !== 'ARCHIVED' && (
                          <Tooltip title="Archive customer">
                            <IconButton
                              color="error"
                              onClick={() => setArchivingCustomer(customer)}
                            >
                              <ArchiveIcon />
                            </IconButton>
                          </Tooltip>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
          <TablePagination
            component="div"
            count={total}
            page={page}
            rowsPerPage={pageSize}
            rowsPerPageOptions={[10, 20, 50, 100]}
            onPageChange={(_, nextPage) => setPage(nextPage)}
            onRowsPerPageChange={(event) => {
              setPageSize(Number(event.target.value))
              setPage(0)
            }}
          />
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
          onSubmit={handleSubmit}
        >
          <DialogTitle>
            {editingCustomer ? `Edit ${editingCustomer.customer_code}` : 'Add Customer'}
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
                  error={Boolean(form.mobile) && !isValidBangladeshMobile(form.mobile)}
                  helperText={
                    form.mobile && !isValidBangladeshMobile(form.mobile)
                      ? 'Enter a valid Bangladesh mobile number: 013-019, exactly 11 digits.'
                      : 'Bangladesh mobile number, e.g. 01712345678'
                  }
                  onChange={(event) =>
                    handleChange(
                      'mobile',
                      event.target.value.replace(/\D/g, '').slice(0, 11),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      inputMode: 'numeric',
                      maxLength: 11,
                    },
                  }}
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
                  error={
                    Boolean(form.alt_mobile) &&
                    !isValidBangladeshMobile(form.alt_mobile ?? '')
                  }
                  helperText={
                    form.alt_mobile && !isValidBangladeshMobile(form.alt_mobile)
                      ? 'Enter a valid Bangladesh mobile number: 013-019, exactly 11 digits.'
                      : 'Optional Bangladesh mobile number'
                  }
                  onChange={(event) =>
                    handleChange(
                      'alt_mobile',
                      event.target.value.replace(/\D/g, '').slice(0, 11),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      inputMode: 'numeric',
                      maxLength: 11,
                    },
                  }}
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
                  required
                  label="NID"
                  value={form.nid}
                  error={Boolean(form.nid) && !isValidCustomerNID(form.nid ?? '')}
                  helperText={
                    form.nid && !isValidCustomerNID(form.nid)
                      ? 'NID must contain only 10 to 17 digits.'
                      : 'Required: 10 to 17 digits'
                  }
                  onChange={(event) =>
                    handleChange(
                      'nid',
                      event.target.value.replace(/\D/g, '').slice(0, 17),
                    )
                  }
                  slotProps={{
                    htmlInput: {
                      inputMode: 'numeric',
                      minLength: 10,
                      maxLength: 17,
                    },
                  }}
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

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  select
                  fullWidth
                  label="POP"
                  value={form.pop_id ?? ''}
                  onChange={(event) => {
                    const popID = event.target.value ? Number(event.target.value) : undefined
                    setForm((current) => ({ ...current, pop_id: popID, agent_id: undefined }))
                  }}
                >
                  <MenuItem value="">Unassigned</MenuItem>
                  {pops.filter((row) => row.status === 'ACTIVE' || row.id === form.pop_id).map((row) => <MenuItem key={row.id} value={row.id}>{row.code} — {row.name}</MenuItem>)}
                </TextField>
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  select
                  fullWidth
                  label="Agent / Reseller"
                  disabled={!form.pop_id}
                  value={form.agent_id ?? ''}
                  onChange={(event) => handleChange('agent_id', event.target.value ? Number(event.target.value) : undefined)}
                >
                  <MenuItem value="">Unassigned</MenuItem>
                  {agents.filter((row) => row.pop_id === form.pop_id && (row.status === 'ACTIVE' || row.id === form.agent_id)).map((row) => <MenuItem key={row.id} value={row.id}>{row.code} — {row.name}</MenuItem>)}
                </TextField>
              </Grid>

              {/* Country */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Country"
                  value={form.country ?? ''}
                  onChange={(event) =>
                    handleChange(
                      'country',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Division */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  select
                  fullWidth
                  label="Division"
                  value={form.division ?? ''}
                  disabled={divisions.length === 0}
                  onChange={(event) =>
                    void handleDivisionChange(event.target.value)
                  }
                >
                  {form.division &&
                    !divisions.some(
                      (item) => item.name === form.division,
                    ) && (
                      <MenuItem value={form.division}>
                        {form.division}
                      </MenuItem>
                    )}
                  {divisions.map((item) => (
                    <MenuItem key={item.id} value={item.name}>
                      {item.name}
                    </MenuItem>
                  ))}
                </TextField>
              </Grid>

              {/* District */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  select
                  fullWidth
                  label="District"
                  value={form.district ?? ''}
                  disabled={!form.division || locationLoading}
                  onChange={(event) =>
                    void handleDistrictChange(event.target.value)
                  }
                >
                  {form.district &&
                    !districts.some(
                      (item) => item.name === form.district,
                    ) && (
                      <MenuItem value={form.district}>
                        {form.district}
                      </MenuItem>
                    )}
                  {districts.map((item) => (
                    <MenuItem key={item.id} value={item.name}>
                      {item.name}
                    </MenuItem>
                  ))}
                </TextField>
              </Grid>

              {/* Thana / Upazila */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  select
                  fullWidth
                  label="Thana / Upazila"
                  value={form.upazila ?? ''}
                  disabled={!form.district || locationLoading}
                  onChange={(event) =>
                    handleUpazilaChange(event.target.value)
                  }
                >
                  {form.upazila &&
                    !upazilas.some(
                      (item) => item.name === form.upazila,
                    ) && (
                      <MenuItem value={form.upazila}>
                        {form.upazila}
                      </MenuItem>
                    )}
                  {upazilas.map((item) => (
                    <MenuItem key={item.id} value={item.name}>
                      {item.name}
                    </MenuItem>
                  ))}
                </TextField>
              </Grid>

              {/* Post Office / Dakghor */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Post Office / Dakghor"
                  value={form.post_office ?? ''}
                  onChange={(event) =>
                    handleChange(
                      'post_office',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Postal Code */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Postal Code"
                  value={form.postal_code ?? ''}
                  onChange={(event) =>
                    handleChange(
                      'postal_code',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Village Name / Holding Number */}
              <Grid size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  label="Village Name / Holding Number"
                  value={form.village_or_holding ?? ''}
                  onChange={(event) =>
                    handleChange(
                      'village_or_holding',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              {/* Road Number / Para / Mohalla */}
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Road Number / Para / Mohalla"
                  value={form.road_or_area ?? ''}
                  onChange={(event) =>
                    handleChange(
                      'road_or_area',
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
                !isValidBangladeshMobile(form.mobile) ||
                (Boolean(form.alt_mobile) &&
                  !isValidBangladeshMobile(form.alt_mobile ?? '')) ||
                !isValidCustomerNID(form.nid ?? '')
              }
              startIcon={
                saving ? (
                  <CircularProgress size={18} />
                ) : (
                  editingCustomer ? <EditIcon /> : <AddIcon />
                )
              }
            >
              {saving
                ? 'Saving...'
                : editingCustomer ? 'Save Changes' : 'Create Customer'}
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog
        open={Boolean(archivingCustomer)}
        onClose={() => !saving && setArchivingCustomer(null)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Archive customer?</DialogTitle>
        <DialogContent>
          <Typography>
            {archivingCustomer?.full_name} ({archivingCustomer?.customer_code})
            will be removed from active operations, but billing history and
            related records will be preserved.
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            Customers with active subscriptions cannot be archived.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setArchivingCustomer(null)} disabled={saving}>
            Cancel
          </Button>
          {!isAgent && <Button
            color="error"
            variant="contained"
            onClick={() => void confirmArchive()}
            disabled={saving}
            startIcon={saving ? <CircularProgress size={18} /> : <ArchiveIcon />}
          >
            {saving ? 'Archiving...' : 'Archive Customer'}
          </Button>}
        </DialogActions>
      </Dialog>

      <Dialog
        open={Boolean(viewingCustomer)}
        onClose={() => setViewingCustomer(null)}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>
          {viewingCustomer?.full_name} · {viewingCustomer?.customer_code}
        </DialogTitle>
        <DialogContent dividers>
          <Grid container spacing={2}>
            {[
              ['Mobile', viewingCustomer?.mobile],
              ['Alternative Mobile', viewingCustomer?.alt_mobile],
              ['Email', viewingCustomer?.email],
              ['NID', viewingCustomer?.nid],
              ['Billing Day', viewingCustomer?.billing_day],
              ['Status', viewingCustomer?.status],
              ['Father Name', viewingCustomer?.father_name],
              ['Mother Name', viewingCustomer?.mother_name],
            ].map(([label, value]) => (
              <Grid key={String(label)} size={{ xs: 12, sm: 6 }}>
                <Typography variant="caption" color="text.secondary">{label}</Typography>
                <Typography>{value || '—'}</Typography>
              </Grid>
            ))}
            <Grid size={{ xs: 12 }}>
              <Typography variant="caption" color="text.secondary">Address</Typography>
              <Typography>
                {[
                  viewingCustomer?.village_or_holding,
                  viewingCustomer?.road_or_area,
                  viewingCustomer?.post_office,
                  viewingCustomer?.upazila,
                  viewingCustomer?.district,
                  viewingCustomer?.division,
                  viewingCustomer?.country,
                ].filter(Boolean).join(', ') || viewingCustomer?.address || '—'}
              </Typography>
            </Grid>
          </Grid>

          <Divider sx={{ my: 3 }} />
          <Typography variant="h6" sx={{ mb: 2 }}>Billing Summary</Typography>
          {summaryLoading ? (
            <Box sx={{ py: 3, textAlign: 'center' }}><CircularProgress /></Box>
          ) : summary ? (
            <Grid container spacing={2}>
              {[
                ['Subscriptions', summary.subscriptions],
                ['Active Subscriptions', summary.active_subscriptions],
                ['Invoices', summary.invoices],
                ['Successful Payments', summary.successful_payments],
				['Cancelled Invoices', summary.cancelled_invoices],
				['Voided Payments', `${summary.voided_payments} · ৳${summary.voided_amount.toFixed(2)}`],
                ['Outstanding', `৳${summary.outstanding_amount.toFixed(2)}`],
                ['Total Paid', `৳${summary.total_paid.toFixed(2)}`],
              ].map(([label, value]) => (
                <Grid key={String(label)} size={{ xs: 12, sm: 6, md: 4 }}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography variant="caption" color="text.secondary">{label}</Typography>
                      <Typography variant="h6">{value}</Typography>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          ) : null}
          <Divider sx={{ my: 3 }} />
          <Typography variant="h6" sx={{ mb: 2 }}>Customer Ledger</Typography>
          {ledger.length === 0 ? <Typography color="text.secondary">No ledger entries.</Typography> : (
            <TableContainer><Table size="small"><TableHead><TableRow><TableCell>Date</TableCell><TableCell>Reference</TableCell><TableCell>Description</TableCell><TableCell align="right">Debit</TableCell><TableCell align="right">Credit</TableCell></TableRow></TableHead><TableBody>
              {ledger.slice(0, 20).map((entry, index) => <TableRow key={`${entry.type}-${entry.reference}-${index}`}><TableCell>{new Date(entry.date).toLocaleDateString()}</TableCell><TableCell>{entry.reference}</TableCell><TableCell>{entry.description}</TableCell><TableCell align="right">{entry.debit ? `৳${entry.debit.toFixed(2)}` : '—'}</TableCell><TableCell align="right">{entry.credit ? `৳${entry.credit.toFixed(2)}` : '—'}</TableCell></TableRow>)}
            </TableBody></Table></TableContainer>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewingCustomer(null)}>Close</Button>
          <Button
            variant="contained"
            onClick={() => {
              if (viewingCustomer) openEditDialog(viewingCustomer)
              setViewingCustomer(null)
            }}
          >
            Edit Customer
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default Customers
