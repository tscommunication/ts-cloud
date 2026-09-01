import { useCallback, useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  IconButton,
  InputAdornment,
  MenuItem,
  Snackbar,
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
import VpnKeyIcon from '@mui/icons-material/VpnKey'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth'

import {
  createCustomer,
  archiveCustomer,
  getCustomers,
  getCustomerSummary,
  getCustomerLedger,
  getCustomerTechnicalProfile,
  getCustomerInternetCredential,
  getCustomerReferences,
  updateCustomer,
  updateCustomerStatus,
  updateCustomerTechnicalProfile,
  createCustomerReference,
  updateCustomerReference,
  deleteCustomerReference,
  saveCustomerInternetCredential,
  getTemporaryInternetAccess,
  grantTemporaryInternetAccess,
  cancelTemporaryInternetAccess,
  bulkExtendCustomerExpiry,
  type CreateCustomerRequest,
  type Customer,
  type CustomerSummary,
  type CustomerLedgerEntry,
  type CustomerTechnicalProfile,
  type UpdateCustomerTechnicalProfileRequest,
  type CustomerReference,
  type CustomerReferenceRequest,
  type CustomerInternetCredential,
  type TemporaryInternetAccess,
  type CustomerListParams,
} from '../../api/customers'
import { getAPIErrorMessage } from '../../api/errors'
import GeoLocationPicker from '../../components/GeoLocationPicker'
import { getStoredUser } from '../../api/auth'
import { getAgents, getPOPs, type Agent, type POP } from '../../api/distribution'
import { getNetworkRouters, type NetworkRouter } from '../../api/networkRouters'
import { getPackages, type Package } from '../../api/packages'
import { getProvisionPackages } from '../../api/customerProvisionRequests'
import {
  adjustSubscriptionDate,
  createSubscription,
  getSubscriptions,
  updateSubscription,
  type Subscription,
} from '../../api/subscriptions'
import {
  getDivisions,
  getDistricts,
  getUpazilas,
  getPostOfficesByDistrict,
  type Division,
  type District,
  type Upazila,
  type PostOffice,
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
  latitude: null,
  longitude: null,
  union: '',
  village: '',
  address: '',
  billing_day: 1,
}

const initialReferenceForm: CustomerReferenceRequest = {
  name: '',
  mobile: '',
  relation: '',
  address: '',
  note: '',
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
  mikrotik_port: '',
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

interface CustomerServiceForm {
  router_id: number
  pppoe_username: string
  pppoe_password: string
  mac_address: string
  static_ip_address: string
  sync_interval_minutes: number
  package_id: number
  activation_date: string
  expiry_date: string
  remarks: string
}

const todayISO = () => {
  const today = new Date()
  const year = today.getFullYear()
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const initialServiceForm = (): CustomerServiceForm => ({
  router_id: 0,
  pppoe_username: '',
  pppoe_password: '',
  mac_address: '',
  static_ip_address: '',
  sync_interval_minutes: 30,
  package_id: 0,
  activation_date: todayISO(),
  expiry_date: '',
  remarks: '',
})

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

const customerDateToISO = (value?: string) => {
  const match = (value ?? '').match(/^(\d{2})-(\d{2})-(\d{4})$/)
  return match ? `${match[3]}-${match[2]}-${match[1]}` : ''
}

const isoToCustomerDate = (value: string) => {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  return match ? `${match[3]}-${match[2]}-${match[1]}` : ''
}

interface DDMMYYYYDateFieldProps {
  label: string
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  required?: boolean
  helperText?: string
}

function DDMMYYYYDateField({
  label,
  value,
  onChange,
  disabled = false,
  required = false,
  helperText,
}: DDMMYYYYDateFieldProps) {
  const pickerRef = useRef<HTMLInputElement>(null)

  const openPicker = () => {
    if (disabled) return
    const picker = pickerRef.current
    if (!picker) return
    picker.showPicker()
  }

  return (
    <Box sx={{ position: 'relative' }}>
      <TextField
        fullWidth
        required={required}
        disabled={disabled}
        label={label}
        value={isoToCustomerDate(value)}
        helperText={helperText ? `DD-MM-YYYY · ${helperText}` : 'DD-MM-YYYY'}
        onClick={openPicker}
        slotProps={{
          htmlInput: { readOnly: true },
          input: {
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  aria-label={`Choose ${label}`}
                  disabled={disabled}
                  edge="end"
                  onClick={openPicker}
                >
                  <CalendarMonthIcon />
                </IconButton>
              </InputAdornment>
            ),
          },
        }}
      />
      <input
        ref={pickerRef}
        type="date"
        aria-hidden="true"
        tabIndex={-1}
        value={value || todayISO()}
        onChange={(event) => onChange(event.target.value)}
        style={{ position: 'absolute', width: 1, height: 1, opacity: 0, pointerEvents: 'none' }}
      />
    </Box>
  )
}

function Customers() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const requestedStatus = searchParams.get('status')
  const initialStatus = ['ACTIVE', 'INACTIVE', 'ARCHIVED'].includes(
    requestedStatus ?? '',
  )
    ? (requestedStatus as 'ACTIVE' | 'INACTIVE' | 'ARCHIVED')
    : ''
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [successMessage, setSuccessMessage] = useState('')
  const [geoLoading, setGeoLoading] = useState(false)
  const [customerTab, setCustomerTab] = useState(0)
	const [search, setSearch] = useState(searchParams.get('search') ?? '')
	const [debouncedSearch, setDebouncedSearch] = useState(searchParams.get('search') ?? '')
  const [statusFilter, setStatusFilter] = useState<'ACTIVE' | 'INACTIVE' | 'ARCHIVED' | ''>(initialStatus)
  const [viewFilter, setViewFilter] = useState<CustomerListParams['view']>(
    (searchParams.get('view')?.toUpperCase() as CustomerListParams['view']) || '',
  )
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
  const [credentialCustomer, setCredentialCustomer] = useState<Customer | null>(null)
  const [internetCredential, setInternetCredential] = useState<CustomerInternetCredential | null>(null)
  const [credentialRouterID, setCredentialRouterID] = useState(0)
  const [credentialUsername, setCredentialUsername] = useState('')
  const [credentialPassword, setCredentialPassword] = useState('')
  const [credentialSaving, setCredentialSaving] = useState(false)
  const [temporaryAccessCustomer, setTemporaryAccessCustomer] = useState<Customer | null>(null)
  const [temporaryAccessItems, setTemporaryAccessItems] = useState<TemporaryInternetAccess[]>([])
  const [temporaryAccessDays, setTemporaryAccessDays] = useState(1)
  const [temporaryAccessSource, setTemporaryAccessSource] = useState<'CUSTOMER' | 'RESELLER'>('CUSTOMER')
  const [temporaryAccessAmount, setTemporaryAccessAmount] = useState(0)
  const [temporaryAccessReason, setTemporaryAccessReason] = useState('')
  const [temporaryAccessSaving, setTemporaryAccessSaving] = useState(false)
  const [selectedCustomerIDs, setSelectedCustomerIDs] = useState<Set<number>>(new Set())
  const [bulkExtendOpen, setBulkExtendOpen] = useState(false)
  const [bulkExtendDays, setBulkExtendDays] = useState(1)
  const [bulkExtendReason, setBulkExtendReason] = useState('')
  const [bulkExtendSaving, setBulkExtendSaving] = useState(false)
  const isSuperadmin = getStoredUser()?.role === 'superadmin'
  const isAgent = getStoredUser()?.role === 'agent'
  const [form, setForm] =
    useState<CreateCustomerRequest>(initialForm)
  const [pops, setPOPs] = useState<POP[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [divisions, setDivisions] = useState<Division[]>([])
  const [districts, setDistricts] = useState<District[]>([])
  const [upazilas, setUpazilas] = useState<Upazila[]>([])
  const [postOffices, setPostOffices] = useState<PostOffice[]>([])
  const [locationLoading, setLocationLoading] = useState(false)
  const [routers, setRouters] = useState<NetworkRouter[]>([])
  const [packages, setPackages] = useState<Package[]>([])
  const [serviceForm, setServiceForm] = useState<CustomerServiceForm>(initialServiceForm)
  const [customerSubscription, setCustomerSubscription] = useState<Subscription | null>(null)

  const [technicalForm, setTechnicalForm] =
    useState<UpdateCustomerTechnicalProfileRequest>(
      initialTechnicalForm,
    )
  const [technicalProfile, setTechnicalProfile] =
    useState<CustomerTechnicalProfile | null>(null)
  const [technicalLoading, setTechnicalLoading] = useState(false)

const [references, setReferences] =
  useState<CustomerReference[]>([])
const [referenceForm, setReferenceForm] =
  useState<CustomerReferenceRequest>(
    initialReferenceForm,
  )
const [editingReference, setEditingReference] =
  useState<CustomerReference | null>(null)
const [referencesLoading, setReferencesLoading] =
  useState(false)
const [referenceBusy, setReferenceBusy] =
  useState(false)

  const loadCustomers = useCallback(async () => {
    try {
      setLoading(true)
      setError('')

      const data = await getCustomers({
        search: debouncedSearch || undefined,
        status: statusFilter,
        view: viewFilter,
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
  }, [debouncedSearch, page, pageSize, statusFilter, viewFilter])

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

  useEffect(() => {
    const loadServiceOptions = async () => {
      try {
        const routerRows = await getNetworkRouters()
        const packageRows = isAgent
          ? (await getProvisionPackages()).map((row) => ({
              ...row,
              commission: 0,
              burst_download: 0,
              burst_upload: 0,
              validity_days: 0,
              mikrotik_profile: '',
              radius_profile: '',
              description: '',
            }))
          : (await getPackages()).packages
        setRouters(routerRows)
        setPackages(packageRows)
      } catch (serviceError: unknown) {
        setError(getAPIErrorMessage(serviceError, 'Failed to load router and package options.'))
      }
    }
    void loadServiceOptions()
  }, [isAgent])

  useEffect(() => {
    const selectedAgent = agents.find((row) => row.id === form.agent_id)
    const available = routers.filter((row) =>
      ((isAgent || selectedAgent) || !form.pop_id || row.pop_id === form.pop_id) &&
      (!selectedAgent || selectedAgent.router_ids.includes(row.id)),
    )
    if (available.some((row) => row.id === serviceForm.router_id)) return

    const nextRouterID = available.length === 1 ? available[0].id : 0
    const timer = window.setTimeout(() => {
      setServiceForm((current) => ({ ...current, router_id: nextRouterID }))
    }, 0)

    return () => window.clearTimeout(timer)
  }, [agents, form.agent_id, form.pop_id, isAgent, routers, serviceForm.router_id])

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
    setPostOffices([])

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
    setPostOffices([])

    const selected = districts.find(
      (item) => item.name === districtName,
    )
    if (!selected) return

    try {
      setLocationLoading(true)
      const [upazilaRows, postOfficeRows] = await Promise.all([
        getUpazilas(selected.id),
        getPostOfficesByDistrict(selected.id),
      ])
      setUpazilas(upazilaRows)
      setPostOffices(postOfficeRows)
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

  const useCurrentLocation = () => {
    if (!navigator.geolocation) {
      setError('Geolocation is not supported by this browser.')
      return
    }

    setGeoLoading(true)
    setError('')

    navigator.geolocation.getCurrentPosition(
      (position) => {
        setForm((current) => ({
          ...current,
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        }))
        setGeoLoading(false)
      },
      (geoError) => {
        if (geoError.code === geoError.PERMISSION_DENIED) {
          setError('Location permission was denied.')
        } else if (geoError.code === geoError.POSITION_UNAVAILABLE) {
          setError('Current location is unavailable.')
        } else if (geoError.code === geoError.TIMEOUT) {
          setError('Location request timed out.')
        } else {
          setError('Failed to get current location.')
        }

        setGeoLoading(false)
      },
      {
        enableHighAccuracy: true,
        timeout: 15000,
        maximumAge: 0,
      },
    )
  }

  const handlePostOfficeChange = (postOfficeName: string) => {
    const selected = postOffices.find(
      (item) => item.name === postOfficeName,
    )

    setForm((current) => ({
      ...current,
      post_office: postOfficeName,
      postal_code: selected?.postal_code ?? '',
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

const handleReferenceChange = (
  field: keyof CustomerReferenceRequest,
  value: string,
) => {
  setReferenceForm((current) => ({
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
  setReferences([])
  setReferenceForm(initialReferenceForm)
  setEditingReference(null)
  setReferencesLoading(false)
  setReferenceBusy(false)
  setCustomerTab(0)
  setServiceForm(initialServiceForm())
  setCustomerSubscription(null)
  setOpen(true)
}

  useEffect(() => {
    const status = searchParams.get('status')
    const nextStatus = ['ACTIVE', 'INACTIVE', 'ARCHIVED'].includes(status ?? '')
      ? (status as 'ACTIVE' | 'INACTIVE' | 'ARCHIVED')
      : ''

    const view = searchParams.get('view')?.toUpperCase() ?? ''
    const nextView = ['EXPIRED', 'PENDING', 'RECENT', 'DISABLED', 'ONLINE', 'OFFLINE'].includes(view)
      ? (view as CustomerListParams['view'])
      : ''

    const timer = window.setTimeout(() => {
		setStatusFilter(nextStatus)
		setViewFilter(nextView)
		setSearch(searchParams.get('search') ?? '')
		setDebouncedSearch(searchParams.get('search') ?? '')
      setPage(0)

      if (searchParams.get('action') === 'add') {
        openCreateDialog()
      }
      if (searchParams.get('action') === 'bulk-extend') {
        setSelectedCustomerIDs(new Set())
      }
    }, 0)

    return () => window.clearTimeout(timer)
  }, [searchParams])

  const bulkExtendMode = searchParams.get('action') === 'bulk-extend'

  const submitBulkExtend = async () => {
    if (selectedCustomerIDs.size === 0 || !bulkExtendReason.trim()) return
    setBulkExtendSaving(true)
    setError('')
    try {
      const response = await bulkExtendCustomerExpiry({ customer_ids: [...selectedCustomerIDs], days: bulkExtendDays, reason: bulkExtendReason.trim() })
      const failed = response.results.filter((item) => !item.success)
      if (failed.length) setError(`${response.results.length - failed.length} updated; ${failed.length} failed: ${failed.map((item) => item.error).join(', ')}`)
      setBulkExtendOpen(false)
      setSelectedCustomerIDs(new Set())
      await loadCustomers()
    } catch (requestError: unknown) {
      setError(getAPIErrorMessage(requestError, 'Bulk expiry extension failed.'))
    } finally {
      setBulkExtendSaving(false)
    }
  }

  const openEditDialog = async (customer: Customer) => {
    setEditingCustomer(customer)
    setDistricts([])
    setUpazilas([])
    setPostOffices([])

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
      latitude: customer.latitude ?? null,
      longitude: customer.longitude ?? null,
      union: customer.union,
      village: customer.village,
      address: customer.address,
      billing_day: customer.billing_day,
      pop_id: customer.pop_id,
      agent_id: customer.agent_id,
    })

    setOpen(true)

    try {
      const subscriptionData = await getSubscriptions()
      const linkedSubscription = subscriptionData.subscriptions.find(
        (row) => row.customer_id === customer.id && row.status !== 'DISCONNECTED',
      ) ?? null

      let credential: CustomerInternetCredential | null = null
      try {
        credential = await getCustomerInternetCredential(customer.id)
      } catch {
        // Legacy/adopted RouterOS customers may intentionally have no PPPoE
        // credential stored in TS-Cloud. Their subscription metadata must still
        // remain editable and the existing RouterOS password must stay untouched.
      }

      setCustomerSubscription(linkedSubscription)
      setServiceForm({
        router_id: credential?.router_id ?? linkedSubscription?.router_id ?? routers.find((row) => row.pop_id === customer.pop_id)?.id ?? 0,
        pppoe_username: credential?.pppoe_username ?? linkedSubscription?.pppoe_username ?? '',
        pppoe_password: '',
        mac_address: credential?.mac_address ?? '',
        static_ip_address: credential?.static_ip_address ?? '',
        sync_interval_minutes: 30,
        package_id: linkedSubscription?.package_id ?? 0,
        activation_date: linkedSubscription?.activation_date?.slice(0, 10) ?? todayISO(),
        expiry_date: linkedSubscription?.expiry_date?.slice(0, 10) ?? '',
        remarks: linkedSubscription?.remarks ?? '',
      })
    } catch (serviceError: unknown) {
      setServiceForm(initialServiceForm())
      setCustomerSubscription(null)
      setError(getAPIErrorMessage(serviceError, 'Failed to load customer service information.'))
    }

setCustomerTab(0)

setReferences([])
setReferenceForm(initialReferenceForm)
setEditingReference(null)
setReferenceBusy(false)

setReferencesLoading(true)
try {
  const referenceRows =
    await getCustomerReferences(customer.id)

  setReferences(referenceRows)
} catch (referenceError: unknown) {
  setReferences([])
  setError(
    getAPIErrorMessage(
      referenceError,
      'Failed to load customer references.',
    ),
  )
} finally {
  setReferencesLoading(false)
}

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
      mikrotik_port: profile.mikrotik_port ?? '',
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

      const postOfficeRows = await getPostOfficesByDistrict(
        selectedDistrict.id,
      )
      setPostOffices(postOfficeRows)

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
    setReferences([])
    setSummaryLoading(true)
    try {
      const [summaryData, ledgerData, referenceRows] =
        await Promise.all([
          getCustomerSummary(customer.id),
          getCustomerLedger(customer.id),
          getCustomerReferences(customer.id),
        ])

      setSummary(summaryData)
      setLedger(ledgerData)
      setReferences(referenceRows)
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load customer summary.'))
    } finally {
      setSummaryLoading(false)
    }
  }

  const openCredentialDialog = async (customer: Customer) => {
    setCredentialCustomer(customer)
    setInternetCredential(null)
    setCredentialRouterID(0)
    setCredentialUsername('')
    setCredentialPassword('')
    try {
      const credential = await getCustomerInternetCredential(customer.id)
      setInternetCredential(credential)
      if (credential) {
        setCredentialRouterID(credential.router_id)
        setCredentialUsername(credential.pppoe_username)
        setCredentialPassword(credential.pppoe_password)
      }
    } catch (error: unknown) {
      // A legacy import can have a canonical Internet Account and linked
      // subscription but no stored secret. Keep the dialog open so the
      // assigned agent can set the verified password and create portal login.
      try {
        const subscriptions = await getSubscriptions()
        const linked = subscriptions.subscriptions.find(
          (row) => row.customer_id === customer.id && row.status !== 'DISCONNECTED',
        )
        if (!linked?.router_id || !linked.pppoe_username) {
          throw error
        }
        setCredentialRouterID(linked.router_id)
        setCredentialUsername(linked.pppoe_username)
        setError('This imported customer has no saved PPPoE password. Enter the verified password to create the PPPoE and Customer Portal credential.')
      } catch {
        setError(getAPIErrorMessage(error, 'Failed to load PPPoE credential.'))
        setCredentialCustomer(null)
      }
    }
  }

  const saveInternetCredential = async () => {
    if (!credentialCustomer) return
    try {
      setCredentialSaving(true)
      setError('')
      const saved = await saveCustomerInternetCredential(credentialCustomer.id, {
        router_id: credentialRouterID,
        pppoe_username: credentialUsername.trim(),
        pppoe_password: credentialPassword,
      })
      setInternetCredential(saved)
      setCredentialCustomer(null)
      setSuccessMessage('PPPoE and Customer Portal password updated successfully.')
      await loadCustomers()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to save PPPoE credential.'))
    } finally {
      setCredentialSaving(false)
    }
  }

  const openTemporaryAccessDialog = async (customer: Customer) => {
    setTemporaryAccessCustomer(customer)
    setTemporaryAccessDays(1)
    setTemporaryAccessSource('CUSTOMER')
    setTemporaryAccessAmount(0)
    setTemporaryAccessReason('')
    try {
      setTemporaryAccessItems(await getTemporaryInternetAccess(customer.id))
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load temporary access history.'))
      setTemporaryAccessCustomer(null)
    }
  }

  const grantCustomerTemporaryAccess = async () => {
    if (!temporaryAccessCustomer || !temporaryAccessReason.trim()) return
    try {
      setTemporaryAccessSaving(true)
      setError('')
      await grantTemporaryInternetAccess(temporaryAccessCustomer.id, {
        days: temporaryAccessDays,
        promised_amount: temporaryAccessAmount,
        request_source: temporaryAccessSource,
        reason: temporaryAccessReason.trim(),
      })
      setTemporaryAccessItems(await getTemporaryInternetAccess(temporaryAccessCustomer.id))
      setTemporaryAccessReason('')
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to grant temporary access.'))
    } finally {
      setTemporaryAccessSaving(false)
    }
  }

  const cancelCustomerTemporaryAccess = async (item: TemporaryInternetAccess) => {
    if (!temporaryAccessCustomer) return
    try {
      setTemporaryAccessSaving(true)
      setError('')
      await cancelTemporaryInternetAccess(temporaryAccessCustomer.id, item.id, 'Cancelled by authorized staff')
      setTemporaryAccessItems(await getTemporaryInternetAccess(temporaryAccessCustomer.id))
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to cancel temporary access.'))
    } finally {
      setTemporaryAccessSaving(false)
    }
  }

  const startEditReference = (
  reference: CustomerReference,
) => {
  setEditingReference(reference)
  setReferenceForm({
    name: reference.name,
    mobile: reference.mobile ?? '',
    relation: reference.relation ?? '',
    address: reference.address ?? '',
    note: reference.note ?? '',
  })
}

const cancelReferenceEdit = () => {
  setEditingReference(null)
  setReferenceForm(initialReferenceForm)
}

const saveReference = async () => {
  const name = referenceForm.name.trim()
  const mobile = referenceForm.mobile?.trim() ?? ''

  if (!name) {
    setError('Reference Name is required.')
    return
  }

  if (mobile && !isValidBangladeshMobile(mobile)) {
    setError(
      'Reference Mobile must be a valid 11-digit Bangladesh mobile number starting with 013-019.',
    )
    return
  }

  if (!editingCustomer) {
    setError(
      'Save the customer first before adding references.',
    )
    return
  }

  const payload: CustomerReferenceRequest = {
    name,
    mobile,
    relation: referenceForm.relation?.trim() ?? '',
    address: referenceForm.address?.trim() ?? '',
    note: referenceForm.note?.trim() ?? '',
  }

  try {
    setReferenceBusy(true)
    setError('')

    let savedReference: CustomerReference

    if (editingReference) {
      savedReference = await updateCustomerReference(
        editingCustomer.id,
        editingReference.id,
        payload,
      )

      setReferences((current) =>
        current.map((row) =>
          row.id === savedReference.id
            ? savedReference
            : row,
        ),
      )
    } else {
      savedReference = await createCustomerReference(
        editingCustomer.id,
        payload,
      )

      setReferences((current) => [
        ...current,
        savedReference,
      ])
    }

    setEditingReference(null)
    setReferenceForm(initialReferenceForm)
  } catch (referenceError: unknown) {
    setError(
      getAPIErrorMessage(
        referenceError,
        'Failed to save customer reference.',
      ),
    )
  } finally {
    setReferenceBusy(false)
  }
}

const removeReference = async (
  reference: CustomerReference,
) => {
  if (!editingCustomer) return

  const confirmed = window.confirm(
    `Delete reference "${reference.name}"?`,
  )

  if (!confirmed) return

  try {
    setReferenceBusy(true)
    setError('')

    await deleteCustomerReference(
      editingCustomer.id,
      reference.id,
    )

    setReferences((current) =>
      current.filter((row) => row.id !== reference.id),
    )

    if (editingReference?.id === reference.id) {
      setEditingReference(null)
      setReferenceForm(initialReferenceForm)
    }
  } catch (referenceError: unknown) {
    setError(
      getAPIErrorMessage(
        referenceError,
        'Failed to delete customer reference.',
      ),
    )
  } finally {
    setReferenceBusy(false)
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

if (!serviceForm.router_id || !serviceForm.pppoe_username.trim() ||
    (!editingCustomer && serviceForm.pppoe_password.length < 8) || !serviceForm.package_id ||
    !serviceForm.activation_date) {
  setCustomerTab(1)
  setError('Service Information is required: router, PPPoE username, password (minimum 8 characters), package, and activation date.')
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

const hasServiceInformation = Boolean(
  serviceForm.router_id || serviceForm.pppoe_username ||
  serviceForm.pppoe_password || serviceForm.package_id,
)

if (hasServiceInformation) {
  if (!serviceForm.router_id || !serviceForm.pppoe_username.trim() ||
      (!editingCustomer && serviceForm.pppoe_password.length < 8) || !serviceForm.package_id) {
    setCustomerTab(1)
    setError('Router, PPPoE username, password (minimum 8 characters), and package are required together.')
    return
  }

  try {
    await saveCustomerInternetCredential(savedCustomer.id, {
      router_id: serviceForm.router_id,
      pppoe_username: serviceForm.pppoe_username.trim(),
      pppoe_password: serviceForm.pppoe_password,
      mac_address: serviceForm.mac_address.trim(),
      static_ip_address: serviceForm.static_ip_address.trim(),
      sync_interval_minutes: serviceForm.sync_interval_minutes,
    })

    if (customerSubscription) {
      await updateSubscription(customerSubscription.id, {
        package_id: serviceForm.package_id,
        billing_day: Number(form.billing_day) || 1,
        remarks: serviceForm.remarks.trim(),
      })
      const currentExpiry = customerSubscription.expiry_date.slice(0, 10)
      if (isSuperadmin && serviceForm.expiry_date && serviceForm.expiry_date !== currentExpiry) {
        await adjustSubscriptionDate(customerSubscription.id, {
          new_expiry_date: serviceForm.expiry_date,
          reason: 'Customer Internet expiry updated from Customer Edit',
        })
      }
    } else {
      await createSubscription({
        customer_id: savedCustomer.id,
        package_id: serviceForm.package_id,
        activation_date: serviceForm.activation_date,
        billing_day: Number(form.billing_day) || 1,
        remarks: serviceForm.remarks.trim(),
      })
    }
  } catch (serviceError: unknown) {
    setError(getAPIErrorMessage(serviceError, 'Customer saved, but service provisioning could not be completed. Correct Service Information and retry Save Changes.'))
    await loadCustomers()
    return
  }
}

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
const wasEditingCustomer = Boolean(editingCustomer)
const previousAgentID = editingCustomer?.agent_id ?? null
const agentChanged = wasEditingCustomer && previousAgentID !== (form.agent_id ?? null)
setForm(initialForm)
setTechnicalForm(initialTechnicalForm)
setServiceForm(initialServiceForm())
setCustomerSubscription(null)
setEditingCustomer(null)
setOpen(false)

if (agentChanged) {
  setSuccessMessage('Customer agent/reseller changed successfully.')
} else {
  setSuccessMessage(wasEditingCustomer ? 'Customer updated successfully.' : 'Customer created successfully.')
}

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

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openCreateDialog}
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

      <Snackbar
        open={Boolean(successMessage)}
        autoHideDuration={5000}
        onClose={(_, reason) => {
          if (reason !== 'clickaway') setSuccessMessage('')
        }}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="success" variant="filled" onClose={() => setSuccessMessage('')}>
          {successMessage}
        </Alert>
      </Snackbar>

      {/* Customer Card */}
      <Card>
        <CardContent>
          {/* Search / Refresh */}
          {bulkExtendMode && (
            <Alert severity="info" sx={{ mb: 2 }} action={<Button color="inherit" size="small" disabled={selectedCustomerIDs.size === 0} onClick={() => setBulkExtendOpen(true)}>Extend Selected ({selectedCustomerIDs.size})</Button>}>
              Select up to 100 customers. Active accounts extend from current expiry; expired accounts extend from today.
            </Alert>
          )}
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
                const nextStatus = event.target.value as 'ACTIVE' | 'INACTIVE' | 'ARCHIVED' | ''
                setStatusFilter(nextStatus)
                setPage(0)
                setSearchParams(nextStatus ? { status: nextStatus } : { view: 'all' })
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
                    {bulkExtendMode && <TableCell padding="checkbox"><Checkbox checked={customers.length > 0 && customers.every((item) => selectedCustomerIDs.has(item.id))} onChange={(event) => setSelectedCustomerIDs(event.target.checked ? new Set(customers.map((item) => item.id)) : new Set())} /></TableCell>}
                    <TableCell>Code</TableCell>
                    <TableCell>Customer</TableCell>
                    <TableCell>Mobile</TableCell>
                    <TableCell>Email</TableCell>
                    <TableCell>Agent / Reseller</TableCell>
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
                      {bulkExtendMode && <TableCell padding="checkbox"><Checkbox checked={selectedCustomerIDs.has(customer.id)} onChange={() => setSelectedCustomerIDs((current) => { const next = new Set(current); if (next.has(customer.id)) next.delete(customer.id); else if (next.size < 100) next.add(customer.id); return next })} /></TableCell>}
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
                        {(() => {
                          const agent = agents.find((item) => item.id === customer.agent_id)
                          return agent ? `${agent.code} — ${agent.name}` : 'Unassigned'
                        })()}
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
                        <Tooltip title="PPPoE / portal credential">
                          <IconButton onClick={() => void openCredentialDialog(customer)}>
                            <VpnKeyIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Temporary Access / Promise to Pay">
                          <IconButton color="info" onClick={() => void openTemporaryAccessDialog(customer)} disabled={customer.status === 'ARCHIVED'}>
                            <AccessTimeIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="View customer">
                          <IconButton onClick={() => void openDetailDialog(customer)}>
                            <VisibilityIcon />
                          </IconButton>
                        </Tooltip>
                        {isAgent && <Tooltip title="Request customer change">
                          <IconButton onClick={() => navigate(`/customer-change-requests?customer_id=${customer.id}`)}>
                            <EditIcon />
                          </IconButton>
                        </Tooltip>}
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
              <Tab label="Reference Information" />
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
            <DDMMYYYYDateField
              label="Date of Birth"
              value={customerDateToISO(form.date_of_birth)}
              onChange={(value) =>
                handleChange(
                  'date_of_birth',
                  isoToCustomerDate(value),
                )
              }
            />
          </Grid>

          {/* Joining Date */}
          <Grid size={{ xs: 12, md: 4 }}>
            <DDMMYYYYDateField
              label="Joining Date"
              value={customerDateToISO(form.joining_date)}
              onChange={(value) =>
                handleChange(
                  'joining_date',
                  isoToCustomerDate(value),
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
            <DDMMYYYYDateField
              label="NID Birth Date"
              value={customerDateToISO(form.nid_birth_date)}
              onChange={(value) =>
                handleChange(
                  'nid_birth_date',
                  isoToCustomerDate(value),
                )
              }
            />
          </Grid>

          {/* NID Issue Date */}
          <Grid size={{ xs: 12, md: 4 }}>
            <DDMMYYYYDateField
              label="NID Issue Date"
              value={customerDateToISO(form.nid_issue_date)}
              onChange={(value) =>
                handleChange(
                  'nid_issue_date',
                  isoToCustomerDate(value),
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
                    void handleUpazilaChange(event.target.value)
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
                  select
                  fullWidth
                  label="Post Office / Dakghor"
                  value={form.post_office ?? ''}
                  disabled={!form.district || locationLoading}
                  onChange={(event) =>
                    handlePostOfficeChange(event.target.value)
                  }
                >
                  {form.post_office &&
                    !postOffices.some(
                      (item) => item.name === form.post_office,
                    ) && (
                      <MenuItem value={form.post_office}>
                        {form.post_office}
                      </MenuItem>
                    )}
                  {postOffices.map((item) => (
                    <MenuItem key={item.id} value={item.name}>
                      {item.name}
                      {item.postal_code
                        ? ` — ${item.postal_code}`
                        : ''}
                      {upazilas.find(
                        (upazila) => upazila.id === item.upazila_id,
                      )?.name
                        ? ` — ${
                            upazilas.find(
                              (upazila) =>
                                upazila.id === item.upazila_id,
                            )?.name
                          }`
                        : ''}
                    </MenuItem>
                  ))}
                </TextField>
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

              <Grid size={{ xs: 12 }}>
                <Divider sx={{ my: 1 }} />
                <Typography variant="subtitle1" sx={{ mb: 1 }}>
                  GPS Location
                </Typography>
              </Grid>

              <Grid size={{ xs: 12, md: 5 }}>
                <TextField
                  fullWidth
                  type="number"
                  label="Latitude"
                  value={form.latitude ?? ''}
                  slotProps={{
                    htmlInput: {
                      step: 'any',
                      min: -90,
                      max: 90,
                    },
                  }}
                  onChange={(event) => {
                    const value = event.target.value
                    setForm((current) => ({
                      ...current,
                      latitude:
                        value === '' ? null : Number(value),
                    }))
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 5 }}>
                <TextField
                  fullWidth
                  type="number"
                  label="Longitude"
                  value={form.longitude ?? ''}
                  slotProps={{
                    htmlInput: {
                      step: 'any',
                      min: -180,
                      max: 180,
                    },
                  }}
                  onChange={(event) => {
                    const value = event.target.value
                    setForm((current) => ({
                      ...current,
                      longitude:
                        value === '' ? null : Number(value),
                    }))
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 2 }}>
                <Button
                  fullWidth
                  variant="outlined"
                  onClick={useCurrentLocation}
                  disabled={geoLoading}
                  sx={{ minHeight: 56 }}
                >
                  {geoLoading
                    ? 'Locating...'
                    : 'Use Current Location'}
                </Button>
              </Grid>

              <Grid size={{ xs: 12 }}>
                <GeoLocationPicker
                  latitude={form.latitude}
                  longitude={form.longitude}
                  onChange={(latitude, longitude) => {
                    setForm((current) => ({
                      ...current,
                      latitude,
                      longitude,
                    }))
                  }}
                />
              </Grid>
            </Grid>

            {customerTab === 1 && (
              <Box sx={{ py: 2 }}>
                <Typography variant="h6" sx={{ mb: 2 }}>
                  Service Information
                </Typography>

                <Alert severity="info" sx={{ mb: 2 }}>
                  PPPoE username and password are also the Customer Portal credential. Saving a complete service creates the linked subscription and provisions MikroTik.
                </Alert>

                <Grid container spacing={{ xs: 2, md: 2 }}>
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  select
                  fullWidth
                  label="POP"
                  value={form.pop_id ?? ''}
                  onChange={(event) => {
                    const popID = event.target.value ? Number(event.target.value) : undefined
                    const eligibleAgents = popID
                      ? agents.filter(
                          (row) =>
                            (row.pop_id === popID || row.pop_ids.includes(popID)) &&
                            row.status === 'ACTIVE',
                        )
                      : []

                    setForm((current) => ({
                      ...current,
                      pop_id: popID,
                      agent_id:
                        eligibleAgents.length === 1
                          ? eligibleAgents[0].id
                          : undefined,
                    }))
                  }}
                >
                  <MenuItem value="">Unassigned</MenuItem>
                  {isAgent && form.pop_id && <MenuItem value={form.pop_id}>Assigned POP</MenuItem>}
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
                  {isAgent && form.agent_id && <MenuItem value={form.agent_id}>My assigned agent account</MenuItem>}
                  {agents
                    .filter(
                      (row) =>
                        (row.pop_id === form.pop_id ||
                          row.pop_ids.includes(form.pop_id ?? 0)) &&
                        (row.status === 'ACTIVE' ||
                          row.id === form.agent_id),
                    )
                    .map((row) => (
                      <MenuItem key={row.id} value={row.id}>
                        {row.code} — {row.name}
                      </MenuItem>
                    ))}
                </TextField>
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  select fullWidth required label="MikroTik Router"
                  value={serviceForm.router_id || ''}
                  onChange={(event) => setServiceForm((current) => ({ ...current, router_id: Number(event.target.value) }))}
                >
                  <MenuItem value="">Select router</MenuItem>
                  {routers.filter((row) => {
                    const selectedAgent = agents.find((agent) => agent.id === form.agent_id)
                    return ((isAgent || selectedAgent) || !form.pop_id || row.pop_id === form.pop_id) &&
                      (!selectedAgent || selectedAgent.router_ids.includes(row.id)) &&
                      (row.status === 'ACTIVE' || row.id === serviceForm.router_id)
                  }).map((row) => (
                    <MenuItem key={row.id} value={row.id}>{row.code} — {row.name}</MenuItem>
                  ))}
                </TextField>
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth required label="PPPoE Username / Portal Login Alias"
                  disabled={isAgent && Boolean(customerSubscription)}
                  value={serviceForm.pppoe_username}
                  onChange={(event) => setServiceForm((current) => ({ ...current, pppoe_username: event.target.value }))}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth required type="text" label="PPPoE & Portal Password"
                  value={serviceForm.pppoe_password}
                  helperText={editingCustomer ? 'Existing legacy password may be kept unchanged; a new password must contain at least 8 characters.' : 'Visible to authorized staff; minimum 8 characters.'}
                  onChange={(event) => setServiceForm((current) => ({ ...current, pppoe_password: event.target.value }))}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  select fullWidth required label="Internet Package"
                  value={serviceForm.package_id || ''}
                  onChange={(event) => setServiceForm((current) => ({ ...current, package_id: Number(event.target.value) }))}
                >
                  <MenuItem value="">Select package</MenuItem>
                  {packages.filter((row) => row.status === 'ACTIVE' || row.id === serviceForm.package_id).map((row) => (
                    <MenuItem key={row.id} value={row.id}>{row.name} ({row.mikrotik_profile})</MenuItem>
                  ))}
                </TextField>
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField fullWidth label="MAC Address" value={serviceForm.mac_address}
                  onChange={(event) => setServiceForm((current) => ({ ...current, mac_address: event.target.value }))} />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField fullWidth label="Static IP Address" value={serviceForm.static_ip_address}
                  onChange={(event) => setServiceForm((current) => ({ ...current, static_ip_address: event.target.value }))} />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <DDMMYYYYDateField
                  required
                  label="Activation Date"
                  disabled={Boolean(customerSubscription)}
                  value={serviceForm.activation_date}
                  onChange={(value) => setServiceForm((current) => ({ ...current, activation_date: value }))}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <DDMMYYYYDateField
                  label="Internet Expiry Date"
                  disabled={!isSuperadmin || !customerSubscription}
                  value={serviceForm.expiry_date}
                  helperText={isSuperadmin ? 'Changing this date immediately reconciles MikroTik.' : 'Only Super Admin can adjust Internet expiry.'}
                  onChange={(value) => setServiceForm((current) => ({ ...current, expiry_date: value }))}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField fullWidth label="Service Remarks" value={serviceForm.remarks}
                  onChange={(event) => setServiceForm((current) => ({ ...current, remarks: event.target.value }))} />
              </Grid>

              <Grid size={{ xs: 12, md: 6 }}>
                <TextField select fullWidth disabled label="Automatic MikroTik Sync"
                  value={30}>
                  <MenuItem value={30}>Every 30 minutes</MenuItem>
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
            label="MikroTik Interface / Port"
            value={technicalForm.mikrotik_port ?? ''}
            helperText="Example: ether5, sfp-sfpplus1, or the PPPoE interface name."
            onChange={(event) =>
              handleTechnicalChange(
                'mikrotik_port',
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
      Reference Information
    </Typography>

    {!editingCustomer && (
      <Alert severity="info" sx={{ mb: 2 }}>
        Save the customer first, then add one or more references.
      </Alert>
    )}

    <Grid container spacing={2}>
      <Grid size={{ xs: 12, md: 6 }}>
        <TextField
          fullWidth
          required
          label="Reference Name"
          value={referenceForm.name}
          disabled={!editingCustomer || referenceBusy}
          onChange={(event) =>
            handleReferenceChange(
              'name',
              event.target.value,
            )
          }
        />
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <TextField
          fullWidth
          label="Reference Mobile"
          value={referenceForm.mobile ?? ''}
          disabled={!editingCustomer || referenceBusy}
          error={
            Boolean(referenceForm.mobile) &&
            !isValidBangladeshMobile(
              referenceForm.mobile ?? '',
            )
          }
          helperText={
            referenceForm.mobile &&
            !isValidBangladeshMobile(
              referenceForm.mobile,
            )
              ? 'Enter a valid Bangladesh mobile number: 013-019, exactly 11 digits.'
              : 'Optional Bangladesh mobile number'
          }
          onChange={(event) =>
            handleReferenceChange(
              'mobile',
              event.target.value
                .replace(/\D/g, '')
                .slice(0, 11),
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

      <Grid size={{ xs: 12, md: 6 }}>
        <TextField
          fullWidth
          label="Relation"
          value={referenceForm.relation ?? ''}
          disabled={!editingCustomer || referenceBusy}
          onChange={(event) =>
            handleReferenceChange(
              'relation',
              event.target.value,
            )
          }
        />
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <TextField
          fullWidth
          label="Address"
          value={referenceForm.address ?? ''}
          disabled={!editingCustomer || referenceBusy}
          onChange={(event) =>
            handleReferenceChange(
              'address',
              event.target.value,
            )
          }
        />
      </Grid>

      <Grid size={{ xs: 12 }}>
        <TextField
          fullWidth
          multiline
          minRows={2}
          label="Reference Note"
          value={referenceForm.note ?? ''}
          disabled={!editingCustomer || referenceBusy}
          onChange={(event) =>
            handleReferenceChange(
              'note',
              event.target.value,
            )
          }
        />
      </Grid>

      <Grid size={{ xs: 12 }}>
        <Box
          sx={{
            display: 'flex',
            gap: 1,
            flexWrap: 'wrap',
          }}
        >
          <Button
            variant="contained"
            disabled={
              !editingCustomer ||
              referenceBusy ||
              !referenceForm.name.trim() ||
              (Boolean(referenceForm.mobile) &&
                !isValidBangladeshMobile(
                  referenceForm.mobile ?? '',
                ))
            }
            onClick={() => void saveReference()}
          >
            {referenceBusy
              ? 'Saving...'
              : editingReference
                ? 'Update Reference'
                : 'Add Reference'}
          </Button>

          {editingReference && (
            <Button
              disabled={referenceBusy}
              onClick={cancelReferenceEdit}
            >
              Cancel Edit
            </Button>
          )}
        </Box>
      </Grid>
    </Grid>

    <Divider sx={{ my: 3 }} />

    {referencesLoading ? (
      <Box sx={{ py: 3, textAlign: 'center' }}>
        <CircularProgress size={28} />
      </Box>
    ) : references.length === 0 ? (
      <Typography color="text.secondary">
        No references added.
      </Typography>
    ) : (
      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Mobile</TableCell>
              <TableCell>Relation</TableCell>
              <TableCell>Address</TableCell>
              <TableCell>Note</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>

          <TableBody>
            {references.map((reference) => (
              <TableRow key={reference.id}>
                <TableCell>
                  {reference.name}
                </TableCell>
                <TableCell>
                  {reference.mobile || '—'}
                </TableCell>
                <TableCell>
                  {reference.relation || '—'}
                </TableCell>
                <TableCell>
                  {reference.address || '—'}
                </TableCell>
                <TableCell>
                  {reference.note || '—'}
                </TableCell>
                <TableCell align="right">
                  <Tooltip title="Edit Reference">
                    <span>
                      <IconButton
                        size="small"
                        disabled={referenceBusy}
                        onClick={() =>
                          startEditReference(reference)
                        }
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>

                  <Tooltip title="Delete Reference">
                    <span>
                      <IconButton
                        size="small"
                        color="error"
                        disabled={referenceBusy}
                        onClick={() =>
                          void removeReference(reference)
                        }
                      >
                        <ArchiveIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    )}
  </Box>
)}

{customerTab === 4 && (
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
        open={Boolean(temporaryAccessCustomer)}
        onClose={() => !temporaryAccessSaving && setTemporaryAccessCustomer(null)}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>
          Temporary Access / Promise to Pay · {temporaryAccessCustomer?.customer_code}
        </DialogTitle>
        <DialogContent dividers>
          <Alert severity="info" sx={{ mb: 2 }}>
            Granted days are enabled immediately and deducted in full from the next regular recharge. Super Admin manual expiry adjustment remains separate.
          </Alert>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 4 }}>
              <TextField select fullWidth required label="Temporary Days" value={temporaryAccessDays}
                onChange={(event) => setTemporaryAccessDays(Number(event.target.value))}>
                {[1, 2, 3, 4, 5, 6, 7].map((days) => <MenuItem key={days} value={days}>{days} day{days > 1 ? 's' : ''}</MenuItem>)}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, sm: 4 }}>
              <TextField select fullWidth required label="Request Source" value={temporaryAccessSource}
                onChange={(event) => setTemporaryAccessSource(event.target.value as 'CUSTOMER' | 'RESELLER')}>
                <MenuItem value="CUSTOMER">Customer</MenuItem>
                <MenuItem value="RESELLER">Reseller</MenuItem>
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, sm: 4 }}>
              <TextField fullWidth type="number" label="Promised Amount" value={temporaryAccessAmount}
                slotProps={{ htmlInput: { min: 0 } }} onChange={(event) => setTemporaryAccessAmount(Number(event.target.value))} />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField fullWidth required multiline minRows={2} label="Reason / Promise Note" value={temporaryAccessReason}
                onChange={(event) => setTemporaryAccessReason(event.target.value)} />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <Button variant="contained" startIcon={<AccessTimeIcon />} onClick={() => void grantCustomerTemporaryAccess()}
                disabled={temporaryAccessSaving || !temporaryAccessReason.trim()}>
                {temporaryAccessSaving ? 'Processing...' : `Grant ${temporaryAccessDays} Day Temporary Access`}
              </Button>
            </Grid>
          </Grid>
          <Divider sx={{ my: 3 }} />
          <Typography variant="h6" sx={{ mb: 1 }}>History &amp; Pending Deduction</Typography>
          {temporaryAccessItems.length === 0 ? <Typography color="text.secondary">No temporary access history.</Typography> : (
            <TableContainer><Table size="small"><TableHead><TableRow>
              <TableCell>Duration</TableCell><TableCell>Start</TableCell><TableCell>End</TableCell><TableCell>Source</TableCell><TableCell>Amount</TableCell><TableCell>Status</TableCell><TableCell align="right">Action</TableCell>
            </TableRow></TableHead><TableBody>
              {temporaryAccessItems.map((item) => <TableRow key={item.id}>
                <TableCell>{Math.round(item.granted_duration_seconds / 86400)} day(s)</TableCell>
                <TableCell>{new Date(item.starts_at).toLocaleString('en-BD')}</TableCell>
                <TableCell>{new Date(item.ends_at).toLocaleString('en-BD')}</TableCell>
                <TableCell>{item.request_source}</TableCell><TableCell>৳{item.promised_amount.toFixed(2)}</TableCell><TableCell>{item.status}</TableCell>
                <TableCell align="right">{item.status === 'ACTIVE' && <Button color="error" size="small" onClick={() => void cancelCustomerTemporaryAccess(item)} disabled={temporaryAccessSaving}>Cancel</Button>}</TableCell>
              </TableRow>)}
            </TableBody></Table></TableContainer>
          )}
        </DialogContent>
        <DialogActions><Button onClick={() => setTemporaryAccessCustomer(null)} disabled={temporaryAccessSaving}>Close</Button></DialogActions>
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
        open={Boolean(credentialCustomer)}
        onClose={() => !credentialSaving && setCredentialCustomer(null)}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>
          PPPoE &amp; Customer Portal Credential · {credentialCustomer?.customer_code}
        </DialogTitle>
        <DialogContent dividers>
          <Alert severity="info" sx={{ mb: 2 }}>
            This customer uses the same PPPoE username and password to sign in to the Customer Portal. CID also remains a valid login ID.
          </Alert>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 4 }}>
              <TextField fullWidth required type="number" label="Router ID" value={credentialRouterID || ''} disabled={isAgent && Boolean(internetCredential)} onChange={(event) => setCredentialRouterID(Number(event.target.value))} />
            </Grid>
            <Grid size={{ xs: 12, sm: 8 }}>
              <TextField fullWidth required label="PPPoE Username / Login Alias" value={credentialUsername} disabled={isAgent && Boolean(internetCredential)} onChange={(event) => setCredentialUsername(event.target.value)} />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField fullWidth required label="PPPoE & Portal Password" value={credentialPassword} onChange={(event) => setCredentialPassword(event.target.value)} helperText="Visible to authorized staff. Minimum 8 characters." />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCredentialCustomer(null)} disabled={credentialSaving}>Cancel</Button>
          <Button variant="contained" onClick={() => void saveInternetCredential()} disabled={credentialSaving || !credentialRouterID || !credentialUsername.trim() || credentialPassword.length < 8}>
            {credentialSaving ? 'Saving...' : internetCredential ? 'Update Credential' : 'Create Credential'}
          </Button>
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

            <Grid size={{ xs: 12, sm: 6 }}>
              <Typography
                variant="caption"
                color="text.secondary"
              >
                Latitude
              </Typography>
              <Typography>
                {viewingCustomer?.latitude ?? '—'}
              </Typography>
            </Grid>

            <Grid size={{ xs: 12, sm: 6 }}>
              <Typography
                variant="caption"
                color="text.secondary"
              >
                Longitude
              </Typography>
              <Typography>
                {viewingCustomer?.longitude ?? '—'}
              </Typography>
            </Grid>

            {viewingCustomer?.latitude != null &&
              viewingCustomer?.longitude != null && (
                <Grid size={{ xs: 12 }}>
                  <Button
                    variant="outlined"
                    component="a"
                    href={`https://www.google.com/maps/search/?api=1&query=${viewingCustomer.latitude},${viewingCustomer.longitude}`}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Open in Google Maps
                  </Button>
                </Grid>
              )}
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

      <Dialog open={bulkExtendOpen} onClose={() => !bulkExtendSaving && setBulkExtendOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Bulk Date Extend</DialogTitle>
        <DialogContent dividers>
          <Alert severity="warning" sx={{ mb: 2 }}>This immediately updates {selectedCustomerIDs.size} customer account(s), writes an audit record, and reconciles MikroTik.</Alert>
          <TextField select fullWidth label="Extend by days" value={bulkExtendDays} onChange={(event) => setBulkExtendDays(Number(event.target.value))} sx={{ mb: 2 }}>
            {[1, 2, 3, 4, 5, 6, 7, 15, 30].map((days) => <MenuItem key={days} value={days}>{days} day{days === 1 ? '' : 's'}</MenuItem>)}
          </TextField>
          <TextField fullWidth required multiline minRows={3} label="Reason" value={bulkExtendReason} onChange={(event) => setBulkExtendReason(event.target.value)} />
        </DialogContent>
        <DialogActions><Button onClick={() => setBulkExtendOpen(false)} disabled={bulkExtendSaving}>Cancel</Button><Button variant="contained" color="warning" onClick={() => void submitBulkExtend()} disabled={bulkExtendSaving || !bulkExtendReason.trim()}>{bulkExtendSaving ? 'Extending...' : 'Confirm Extension'}</Button></DialogActions>
      </Dialog>
    </Box>
  )
}

export default Customers
