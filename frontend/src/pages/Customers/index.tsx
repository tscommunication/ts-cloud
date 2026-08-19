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
  Tabs,
  Tab,
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
  getCustomerTechnicalProfile,
  updateCustomer,
  updateCustomerStatus,
  updateCustomerTechnicalProfile,
  type CreateCustomerRequest,
  type Customer,
  type CustomerSummary,
  type CustomerLedgerEntry,
  type CustomerTechnicalProfile,
  type UpdateCustomerTechnicalProfileRequest,
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
  date_of_birth: '',
  joining_date: '',
  occupation: '',
  company_name: '',
  designation: '',
  nid_birth_date: '',
  nid_issue_date: '',
  nid_address: '',
  present_address: '',
  permanent_address: '',
  tin: '',
  customer_note: '',
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

const initialTechnicalForm: UpdateCustomerTechnicalProfileRequest = {
  onu_mac: '',
  olt_pon: '',
  olt_slot: '',
  olt_port: '',
  onu_type: '',
  onu_model: '',
  onu_ip: '',
  onu_password: '',
  onu_serial: '',
  onu_sn: '',
  router_brand: '',
  router_model: '',
  router_ip: '',
  router_password: '',
  cable_type: '',
  cable_length: 0,
  media_converter_mac: '',
  media_converter_ip: '',
  media_converter_password: '',
  switch_model: '',
  switch_port: '',
  switch_ip: '',
  switch_password: '',
  additional_note: '',
}

const bangladeshMobileRegex = /^01[3-9][0-9]{8}$/
const customerNIDRegex = /^[0-9]{10,17}$/

const isValidBangladeshMobile = (value: string) =>
  bangladeshMobileRegex.test(value.trim())

const isValidCustomerNID = (value: string) =>
  customerNIDRegex.test(value.trim())

const isValidOptionalCustomerDate = (value?: string) => {
  const normalized = value?.trim() ?? ''

  if (!normalized) return true

  const match = normalized.match(
    /^(\d{2})-(\d{2})-(\d{4})$/,
  )

  if (!match) return false

  const day = Number(match[1])
  const month = Number(match[2])
  const year = Number(match[3])

  const parsed = new Date(
    Date.UTC(year, month - 1, day),
  )

  return (
    parsed.getUTCFullYear() === year &&
    parsed.getUTCMonth() === month - 1 &&
    parsed.getUTCDate() === day
  )
}

function Customers() {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [customerTab, setCustomerTab] = useState(0)
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

  const [technicalForm, setTechnicalForm] =
    useState<UpdateCustomerTechnicalProfileRequest>(
      initialTechnicalForm,
    )
  const [technicalProfile, setTechnicalProfile] =
    useState<CustomerTechnicalProfile | null>(null)
  const [technicalLoading, setTechnicalLoading] = useState(false)
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

  const handleTechnicalChange = (
  field: keyof UpdateCustomerTechnicalProfileRequest,
  value: string | number | undefined,
) => {
  setTechnicalForm((current) => ({
    ...current,
    [field]: value,
  }))
}

const openCreateDialog = () => {
  setEditingCustomer(null)
  setForm(initialForm)
  setTechnicalForm(initialTechnicalForm)
  setTechnicalProfile(null)
  setTechnicalLoading(false)
  setCustomerTab(0)
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
      date_of_birth: customer.date_of_birth ?? '',
      joining_date: customer.joining_date ?? '',
      occupation: customer.occupation ?? '',
      company_name: customer.company_name ?? '',
      designation: customer.designation ?? '',
      nid_birth_date: customer.nid_birth_date ?? '',
      nid_issue_date: customer.nid_issue_date ?? '',
      nid_address: customer.nid_address ?? '',
      present_address: customer.present_address ?? '',
      permanent_address: customer.permanent_address ?? '',
      tin: customer.tin ?? '',
      customer_note: customer.customer_note ?? '',
      country: customer.country?.trim() || 'Bangladesh',
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

setCustomerTab(0)

setTechnicalLoading(true)
try {
  const profile = await getCustomerTechnicalProfile(customer.id)

  setTechnicalProfile(profile)

  if (profile) {
    setTechnicalForm({
      onu_mac: profile.onu_mac ?? '',
      olt_pon: profile.olt_pon ?? '',
      olt_slot: profile.olt_slot ?? '',
      olt_port: profile.olt_port ?? '',
      onu_type: profile.onu_type ?? '',
      onu_model: profile.onu_model ?? '',
      onu_ip: profile.onu_ip ?? '',
      onu_password: '',
      onu_serial: profile.onu_serial ?? '',
      onu_sn: profile.onu_sn ?? '',
      router_brand: profile.router_brand ?? '',
      router_model: profile.router_model ?? '',
      router_ip: profile.router_ip ?? '',
      router_password: '',
      cable_type: profile.cable_type ?? '',
      cable_length: profile.cable_length ?? 0,
      media_converter_mac: profile.media_converter_mac ?? '',
      media_converter_ip: profile.media_converter_ip ?? '',
      media_converter_password: '',
      switch_model: profile.switch_model ?? '',
      switch_port: profile.switch_port ?? '',
      switch_ip: profile.switch_ip ?? '',
      switch_password: '',
      additional_note: profile.additional_note ?? '',
    })
  } else {
    setTechnicalForm(initialTechnicalForm)
  }
} catch (error: unknown) {
  setTechnicalProfile(null)
  setTechnicalForm(initialTechnicalForm)
  setError(
    getAPIErrorMessage(
      error,
      'Failed to load customer technical profile.',
    ),
  )
} finally {
  setTechnicalLoading(false)
}


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

    const customerDates = [
  ['Date of Birth', form.date_of_birth],
  ['Joining Date', form.joining_date],
  ['NID Birth Date', form.nid_birth_date],
  ['NID Issue Date', form.nid_issue_date],
] as const

for (const [label, value] of customerDates) {
  if (!isValidOptionalCustomerDate(value)) {
    setError(`${label} must use DD-MM-YYYY format.`)
    return
  }
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

      let savedCustomer: Customer

if (editingCustomer) {
  savedCustomer = await updateCustomer(
    editingCustomer.id,
    payload,
  )
} else {
  savedCustomer = await createCustomer(payload)
}

// The customer is persisted at this point. Switch to edit mode
// immediately so a technical-profile failure can be retried
// without creating a duplicate customer.
setEditingCustomer(savedCustomer)

let savedTechnicalProfile: CustomerTechnicalProfile

try {
  savedTechnicalProfile =
    await updateCustomerTechnicalProfile(
      savedCustomer.id,
      technicalForm,
    )
} catch (technicalError: unknown) {
  setError(
    getAPIErrorMessage(
      technicalError,
      'Customer saved, but the technical profile could not be saved. Retry Save Changes to try again.',
    ),
  )

  await loadCustomers()
  return
}

setTechnicalProfile(savedTechnicalProfile)
setForm(initialForm)
setTechnicalForm(initialTechnicalForm)
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
            <Tabs
              value={customerTab}
              onChange={(_, value) => setCustomerTab(value)}
              variant="scrollable"
              scrollButtons="auto"
              sx={{ mb: 2 }}
            >
              <Tab label="Basic Information" />
              <Tab label="Service Information" />
              <Tab label="Technical Information" />
              <Tab label="Billing Information" />
            </Tabs>

            <Grid
              container
              spacing={2}
              sx={{
                pt: 1,
                display: customerTab === 0 ? 'flex' : 'none',
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

              {/* Country */}
              {/* Date of Birth */}
          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label="Date of Birth"
              placeholder="DD-MM-YYYY"
              value={form.date_of_birth ?? ''}
              error={
                Boolean(form.date_of_birth) &&
                !isValidOptionalCustomerDate(form.date_of_birth)
              }
              helperText={
                form.date_of_birth &&
                !isValidOptionalCustomerDate(form.date_of_birth)
                  ? 'Use DD-MM-YYYY, e.g. 19-08-1990'
                  : 'DD-MM-YYYY'
              }
              onChange={(event) =>
                handleChange(
                  'date_of_birth',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Joining Date */}
          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label="Joining Date"
              placeholder="DD-MM-YYYY"
              value={form.joining_date ?? ''}
              error={
                Boolean(form.joining_date) &&
                !isValidOptionalCustomerDate(form.joining_date)
              }
              helperText={
                form.joining_date &&
                !isValidOptionalCustomerDate(form.joining_date)
                  ? 'Use DD-MM-YYYY, e.g. 19-08-2026'
                  : 'DD-MM-YYYY'
              }
              onChange={(event) =>
                handleChange(
                  'joining_date',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Occupation */}
          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label="Occupation"
              value={form.occupation ?? ''}
              onChange={(event) =>
                handleChange(
                  'occupation',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Company Name */}
          <Grid size={{ xs: 12, md: 6 }}>
            <TextField
              fullWidth
              label="Company Name"
              value={form.company_name ?? ''}
              onChange={(event) =>
                handleChange(
                  'company_name',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Designation */}
          <Grid size={{ xs: 12, md: 6 }}>
            <TextField
              fullWidth
              label="Designation"
              value={form.designation ?? ''}
              onChange={(event) =>
                handleChange(
                  'designation',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* NID Birth Date */}
          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label="NID Birth Date"
              placeholder="DD-MM-YYYY"
              value={form.nid_birth_date ?? ''}
              error={
                Boolean(form.nid_birth_date) &&
                !isValidOptionalCustomerDate(form.nid_birth_date)
              }
              helperText={
                form.nid_birth_date &&
                !isValidOptionalCustomerDate(form.nid_birth_date)
                  ? 'Use DD-MM-YYYY'
                  : 'DD-MM-YYYY'
              }
              onChange={(event) =>
                handleChange(
                  'nid_birth_date',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* NID Issue Date */}
          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label="NID Issue Date"
              placeholder="DD-MM-YYYY"
              value={form.nid_issue_date ?? ''}
              error={
                Boolean(form.nid_issue_date) &&
                !isValidOptionalCustomerDate(form.nid_issue_date)
              }
              helperText={
                form.nid_issue_date &&
                !isValidOptionalCustomerDate(form.nid_issue_date)
                  ? 'Use DD-MM-YYYY'
                  : 'DD-MM-YYYY'
              }
              onChange={(event) =>
                handleChange(
                  'nid_issue_date',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* TIN */}
          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label="TIN"
              value={form.tin ?? ''}
              onChange={(event) =>
                handleChange(
                  'tin',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* NID Address */}
          <Grid size={{ xs: 12 }}>
            <TextField
              fullWidth
              multiline
              minRows={2}
              label="NID Address"
              value={form.nid_address ?? ''}
              onChange={(event) =>
                handleChange(
                  'nid_address',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Present Address */}
          <Grid size={{ xs: 12, md: 6 }}>
            <TextField
              fullWidth
              multiline
              minRows={2}
              label="Present Address"
              value={form.present_address ?? ''}
              onChange={(event) =>
                handleChange(
                  'present_address',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Permanent Address */}
          <Grid size={{ xs: 12, md: 6 }}>
            <TextField
              fullWidth
              multiline
              minRows={2}
              label="Permanent Address"
              value={form.permanent_address ?? ''}
              onChange={(event) =>
                handleChange(
                  'permanent_address',
                  event.target.value,
                )
              }
            />
          </Grid>

          {/* Customer Note */}
          <Grid size={{ xs: 12 }}>
            <TextField
              fullWidth
              multiline
              minRows={2}
              label="Customer Note"
              value={form.customer_note ?? ''}
              onChange={(event) =>
                handleChange(
                  'customer_note',
                  event.target.value,
                )
              }
            />
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Country"
                  value="Bangladesh"
                  disabled
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

              {/* Road Number / Para / Mohalla */}
              <Grid size={{ xs: 12, md: 6 }}>
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

              {/* Village Name / Holding Number */}
              <Grid size={{ xs: 12, md: 6 }}>
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
            </Grid>

            {customerTab === 1 && (
              <Box sx={{ py: 2 }}>
                <Typography variant="h6" sx={{ mb: 2 }}>
                  Service Information
                </Typography>

                <Grid container spacing={{ xs: 2, md: 2 }}>
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


                </Grid>
              </Box>
            )}

            {customerTab === 2 && (
  <Box sx={{ py: 2 }}>
    <Typography variant="h6" sx={{ mb: 2 }}>
      Technical Information
    </Typography>

    {technicalLoading ? (
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'center',
          py: 5,
        }}
      >
        <CircularProgress size={28} />
      </Box>
    ) : (
      <Grid container spacing={{ xs: 2, md: 2 }}>
        {/* ONU */}
        <Grid size={{ xs: 12 }}>
          <Typography
            variant="subtitle1"
            sx={{ fontWeight: 600 }}
          >
            ONU Information
          </Typography>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="ONU MAC"
            value={technicalForm.onu_mac ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'onu_mac',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="ONU Type"
            value={technicalForm.onu_type ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'onu_type',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="ONU Model"
            value={technicalForm.onu_model ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'onu_model',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="ONU IP"
            value={technicalForm.onu_ip ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'onu_ip',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="ONU Serial"
            value={technicalForm.onu_serial ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'onu_serial',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="ONU SN"
            value={technicalForm.onu_sn ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'onu_sn',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <TextField
            fullWidth
            type="password"
            label="ONU Password"
            value={technicalForm.onu_password ?? ''}
            helperText={
              technicalProfile?.onu_password_configured
                ? 'Password configured. Leave blank to keep existing password.'
                : 'Optional. Stored encrypted.'
            }
            onChange={(event) =>
              handleTechnicalChange(
                'onu_password',
                event.target.value,
              )
            }
          />
        </Grid>

        {/* OLT */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }} />
          <Typography
            variant="subtitle1"
            sx={{ fontWeight: 600 }}
          >
            OLT Information
          </Typography>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="OLT PON"
            value={technicalForm.olt_pon ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'olt_pon',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="OLT Slot"
            value={technicalForm.olt_slot ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'olt_slot',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="OLT Port"
            value={technicalForm.olt_port ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'olt_port',
                event.target.value,
              )
            }
          />
        </Grid>

        {/* Router */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }} />
          <Typography
            variant="subtitle1"
            sx={{ fontWeight: 600 }}
          >
            Customer Router
          </Typography>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="Router Brand"
            value={technicalForm.router_brand ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'router_brand',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="Router Model"
            value={technicalForm.router_model ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'router_model',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="Router IP"
            value={technicalForm.router_ip ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'router_ip',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <TextField
            fullWidth
            type="password"
            label="Router Password"
            value={technicalForm.router_password ?? ''}
            helperText={
              technicalProfile?.router_password_configured
                ? 'Password configured. Leave blank to keep existing password.'
                : 'Optional. Stored encrypted.'
            }
            onChange={(event) =>
              handleTechnicalChange(
                'router_password',
                event.target.value,
              )
            }
          />
        </Grid>

        {/* Cable */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }} />
          <Typography
            variant="subtitle1"
            sx={{ fontWeight: 600 }}
          >
            Cable Information
          </Typography>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <TextField
            fullWidth
            label="Cable Type"
            value={technicalForm.cable_type ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'cable_type',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <TextField
            fullWidth
            type="number"
            label="Cable Length"
            value={technicalForm.cable_length ?? 0}
            onChange={(event) =>
              handleTechnicalChange(
                'cable_length',
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

        {/* Media Converter */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }} />
          <Typography
            variant="subtitle1"
            sx={{ fontWeight: 600 }}
          >
            Media Converter
          </Typography>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="Media Converter MAC"
            value={technicalForm.media_converter_mac ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'media_converter_mac',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            label="Media Converter IP"
            value={technicalForm.media_converter_ip ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'media_converter_ip',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <TextField
            fullWidth
            type="password"
            label="Media Converter Password"
            value={
              technicalForm.media_converter_password ?? ''
            }
            helperText={
              technicalProfile
                ?.media_converter_password_configured
                ? 'Password configured. Leave blank to keep existing password.'
                : 'Optional. Stored encrypted.'
            }
            onChange={(event) =>
              handleTechnicalChange(
                'media_converter_password',
                event.target.value,
              )
            }
          />
        </Grid>

        {/* Switch */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }} />
          <Typography
            variant="subtitle1"
            sx={{ fontWeight: 600 }}
          >
            Switch Information
          </Typography>
        </Grid>

        <Grid size={{ xs: 12, md: 3 }}>
          <TextField
            fullWidth
            label="Switch Model"
            value={technicalForm.switch_model ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'switch_model',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 3 }}>
          <TextField
            fullWidth
            label="Switch Port"
            value={technicalForm.switch_port ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'switch_port',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 3 }}>
          <TextField
            fullWidth
            label="Switch IP"
            value={technicalForm.switch_ip ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'switch_ip',
                event.target.value,
              )
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 3 }}>
          <TextField
            fullWidth
            type="password"
            label="Switch Password"
            value={technicalForm.switch_password ?? ''}
            helperText={
              technicalProfile?.switch_password_configured
                ? 'Password configured. Leave blank to keep existing password.'
                : 'Optional. Stored encrypted.'
            }
            onChange={(event) =>
              handleTechnicalChange(
                'switch_password',
                event.target.value,
              )
            }
          />
        </Grid>

        {/* Additional Note */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }} />
          <TextField
            fullWidth
            multiline
            minRows={3}
            label="Additional Note"
            value={technicalForm.additional_note ?? ''}
            onChange={(event) =>
              handleTechnicalChange(
                'additional_note',
                event.target.value,
              )
            }
          />
        </Grid>
      </Grid>
    )}
  </Box>
)}

            {customerTab === 3 && (
              <Box sx={{ py: 2 }}>
                <Typography variant="h6" sx={{ mb: 2 }}>
                  Billing Information
                </Typography>

                <Grid container spacing={{ xs: 2, md: 2 }}>
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


                </Grid>
              </Box>
            )}

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
