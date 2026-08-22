import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Typography,
} from '@mui/material'

import { getStoredUser } from '../../api/auth'
import { getAPIErrorMessage } from '../../api/errors'
import {
  approveProvisionRequest,
  createProvisionRequest,
  getProvisionPackages,
  getProvisionRequests,
  getProvisionRouters,
  rejectProvisionRequest,
  type CustomerProvisionRequest,
  type CreateCustomerProvisionRequestInput,
  type ProvisionPackage,
  type ProvisionRequestStatus,
  type ProvisionRouter,
} from '../../api/customerProvisionRequests'
import {
  getDistricts,
  getDivisions,
  getPostOffices,
  getUpazilas,
  type District,
  type Division,
  type PostOffice,
  type Upazila,
} from '../../api/locations'

type StatusFilter = '' | ProvisionRequestStatus

const bangladeshMobileRegex = /^01[3-9][0-9]{8}$/
const customerNIDRegex = /^[0-9]{10,17}$/

function localDateString() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const initialForm: CreateCustomerProvisionRequestInput = {
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
  latitude: null,
  longitude: null,

  package_id: 0,
  router_id: undefined,

  pppoe_username: '',
  pppoe_password: '',

  billing_day: 1,
  activation_date: localDateString(),

  remarks: '',
}

function CustomerProvisionRequests() {
  const storedUser = getStoredUser()
  const isAgent = storedUser?.role === 'agent'

  const [rows, setRows] =
    useState<CustomerProvisionRequest[]>([])
  const [status, setStatus] =
    useState<StatusFilter>('PENDING')
  const [loading, setLoading] = useState(true)
  const [actionID, setActionID] =
    useState<number | null>(null)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [geoLoading, setGeoLoading] = useState(false)

  const [requestOpen, setRequestOpen] = useState(false)
  const [requestTab, setRequestTab] = useState(0)
  const [saving, setSaving] = useState(false)
  const [catalogLoading, setCatalogLoading] =
    useState(false)

  const [form, setForm] =
    useState<CreateCustomerProvisionRequestInput>(
      initialForm,
    )

  const [packages, setPackages] =
    useState<ProvisionPackage[]>([])
  const [routers, setRouters] =
    useState<ProvisionRouter[]>([])

  const [divisions, setDivisions] =
    useState<Division[]>([])
  const [districts, setDistricts] =
    useState<District[]>([])
  const [upazilas, setUpazilas] =
    useState<Upazila[]>([])
  const [postOffices, setPostOffices] =
    useState<PostOffice[]>([])
  const [locationLoading, setLocationLoading] =
    useState(false)

  const loadRows = useCallback(async () => {
    try {
      setLoading(true)
      setError('')

      const response =
        await getProvisionRequests(status)

      setRows(response.requests)
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to load connection requests.',
        ),
      )
    } finally {
      setLoading(false)
    }
  }, [status])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadRows()
    }, 0)

    return () => window.clearTimeout(timer)
  }, [loadRows])

  const openNewRequest = async () => {
    setError('')
    setMessage('')
    setRequestTab(0)
    setForm({
      ...initialForm,
      activation_date: localDateString(),
    })
    setDistricts([])
    setUpazilas([])
    setPostOffices([])
    setRequestOpen(true)

    try {
      setCatalogLoading(true)

      const [
        packageRows,
        routerRows,
        divisionRows,
      ] = await Promise.all([
        getProvisionPackages(),
        getProvisionRouters(),
        getDivisions(),
      ])

      setPackages(packageRows)
      setRouters(routerRows)
      setDivisions(divisionRows)
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to load connection request options.',
        ),
      )
    } finally {
      setCatalogLoading(false)
    }
  }

  const closeNewRequest = () => {
    if (saving) return
    setRequestOpen(false)
  }

  const handleDivisionChange = async (
    divisionName: string,
  ) => {
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
      setDistricts(
        await getDistricts(selected.id),
      )
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to load district options.',
        ),
      )
    } finally {
      setLocationLoading(false)
    }
  }

  const handleDistrictChange = async (
    districtName: string,
  ) => {
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
      setUpazilas(
        await getUpazilas(selected.id),
      )
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to load upazila options.',
        ),
      )
    } finally {
      setLocationLoading(false)
    }
  }

  const handleUpazilaChange = async (
    upazilaName: string,
  ) => {
    setForm((current) => ({
      ...current,
      upazila: upazilaName,
      post_office: '',
      postal_code: '',
    }))

    setPostOffices([])

    const selected = upazilas.find(
      (item) => item.name === upazilaName,
    )

    if (!selected) return

    try {
      setLocationLoading(true)

      const rows = await getPostOffices(
        selected.id,
      )

      setPostOffices(rows)
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to load post office options.',
        ),
      )
    } finally {
      setLocationLoading(false)
    }
  }

  const useCurrentLocation = () => {
    if (!navigator.geolocation) {
      setError('Geolocation is not supported by this browser.')
      return
    }

    setGeoLoading(true)
    setError('')
    setMessage('')

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

  const handlePostOfficeChange = (
    postOfficeName: string,
  ) => {
    const selected = postOffices.find(
      (item) => item.name === postOfficeName,
    )

    setForm((current) => ({
      ...current,
      post_office: postOfficeName,
      postal_code: selected?.postal_code ?? '',
    }))
  }

  const submitRequest = async () => {
    const fullName = form.full_name.trim()
    const mobile = form.mobile.trim()
    const nid = form.nid.trim()
    const pppoeUsername =
      form.pppoe_username.trim()

    setError('')
    setMessage('')

    if (!fullName) {
      setRequestTab(0)
      setError('Full Name is required.')
      return
    }

    if (!bangladeshMobileRegex.test(mobile)) {
      setRequestTab(0)
      setError(
        'Mobile must be a valid Bangladesh mobile number.',
      )
      return
    }

    if (
      form.alt_mobile?.trim() &&
      !bangladeshMobileRegex.test(
        form.alt_mobile.trim(),
      )
    ) {
      setRequestTab(0)
      setError(
        'Alternative mobile must be a valid Bangladesh mobile number.',
      )
      return
    }

    if (!customerNIDRegex.test(nid)) {
      setRequestTab(0)
      setError(
        'NID must contain 10 to 17 digits.',
      )
      return
    }

    if (!form.package_id) {
      setRequestTab(1)
      setError('Package is required.')
      return
    }

    if (!pppoeUsername) {
      setRequestTab(1)
      setError('PPPoE Username is required.')
      return
    }

    if (
      form.billing_day < 1 ||
      form.billing_day > 31
    ) {
      setRequestTab(2)
      setError(
        'Billing Day must be between 1 and 31.',
      )
      return
    }

    try {
      setSaving(true)

      const created =
        await createProvisionRequest({
          ...form,
          full_name: fullName,
          mobile,
          father_name:
            form.father_name?.trim(),
          mother_name:
            form.mother_name?.trim(),
          alt_mobile:
            form.alt_mobile?.trim(),
          email: form.email?.trim(),
          nid,
          post_office:
            form.post_office?.trim(),
          postal_code:
            form.postal_code?.trim(),
          road_or_area:
            form.road_or_area?.trim(),
          village_or_holding:
            form.village_or_holding?.trim(),
          pppoe_username: pppoeUsername,
          pppoe_password:
            form.pppoe_password ?? '',
          remarks: form.remarks?.trim(),
          router_id:
            form.router_id || undefined,
        })

      setRequestOpen(false)

      setMessage(
        `${created.request_code} submitted successfully and is waiting for approval.`,
      )

      setStatus('PENDING')
      await loadRows()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to submit connection request.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const approve = async (id: number) => {
    try {
      setActionID(id)
      setError('')
      setMessage('')

      await approveProvisionRequest(id)

      setMessage(
        'Request approved and customer subscription created.',
      )

      await loadRows()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to approve request.',
        ),
      )
    } finally {
      setActionID(null)
    }
  }

  const reject = async (
    row: CustomerProvisionRequest,
  ) => {
    const reason = window.prompt(
      `Reject ${row.request_code}\n\nEnter rejection reason:`,
    )

    if (reason === null) return

    if (!reason.trim()) {
      setError('Rejection reason is required.')
      return
    }

    try {
      setActionID(row.id)
      setError('')
      setMessage('')

      await rejectProvisionRequest(
        row.id,
        reason.trim(),
      )

      setMessage('Request rejected.')
      await loadRows()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to reject request.',
        ),
      )
    } finally {
      setActionID(null)
    }
  }

  const statusColor = (
    value: ProvisionRequestStatus,
  ):
    | 'default'
    | 'warning'
    | 'success'
    | 'error'
    | 'info' => {
    switch (value) {
      case 'PENDING':
        return 'warning'
      case 'COMPLETED':
        return 'success'
      case 'APPROVED':
        return 'info'
      case 'REJECTED':
        return 'error'
      default:
        return 'default'
    }
  }

  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: {
            xs: 'column',
            md: 'row',
          },
          justifyContent: 'space-between',
          alignItems: {
            xs: 'stretch',
            md: 'center',
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
            Connection Requests
          </Typography>

          <Typography color="text.secondary">
            {isAgent
              ? 'Submit and track new customer / PPPoE connection requests.'
              : 'Review and approve customer connection requests submitted by agents.'}
          </Typography>
        </Box>

        {isAgent && (
          <Button
            variant="contained"
            onClick={() =>
              void openNewRequest()
            }
          >
            New Customer / PPPoE Request
          </Button>
        )}
      </Box>

      {error && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => setError('')}
        >
          {error}
        </Alert>
      )}

      {message && (
        <Alert
          severity="success"
          sx={{ mb: 2 }}
          onClose={() => setMessage('')}
        >
          {message}
        </Alert>
      )}

      <Paper sx={{ mb: 2, p: 2 }}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: {
              xs: 'column',
              sm: 'row',
            },
            gap: 2,
            alignItems: {
              xs: 'stretch',
              sm: 'center',
            },
          }}
        >
          <FormControl
            size="small"
            sx={{ minWidth: 220 }}
          >
            <InputLabel>Status</InputLabel>

            <Select
              value={status}
              label="Status"
              onChange={(event) =>
                setStatus(
                  event.target
                    .value as StatusFilter,
                )
              }
            >
              <MenuItem value="">
                All Statuses
              </MenuItem>
              <MenuItem value="PENDING">
                Pending
              </MenuItem>
              <MenuItem value="COMPLETED">
                Completed
              </MenuItem>
              <MenuItem value="REJECTED">
                Rejected
              </MenuItem>
              <MenuItem value="CANCELLED">
                Cancelled
              </MenuItem>
              <MenuItem value="APPROVED">
                Approved
              </MenuItem>
            </Select>
          </FormControl>

          <TextField
            size="small"
            value={`${rows.length} request${rows.length === 1 ? '' : 's'}`}
            slotProps={{
              input: {
                readOnly: true,
              },
            }}
          />
        </Box>
      </Paper>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Request</TableCell>
              <TableCell>Customer</TableCell>
              <TableCell>Mobile</TableCell>
              <TableCell>
                PPPoE Username
              </TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Requested</TableCell>
              {!isAgent && (
                <TableCell align="right">
                  Actions
                </TableCell>
              )}
            </TableRow>
          </TableHead>

          <TableBody>
            {loading && (
              <TableRow>
                <TableCell
                  colSpan={isAgent ? 6 : 7}
                  align="center"
                  sx={{ py: 5 }}
                >
                  <CircularProgress size={28} />
                </TableCell>
              </TableRow>
            )}

            {!loading &&
              rows.map((row) => (
                <TableRow
                  key={row.id}
                  hover
                >
                  <TableCell>
                    <Typography
                      variant="body2"
                      sx={{ fontWeight: 600 }}
                    >
                      {row.request_code}
                    </Typography>

                    <Typography
                      variant="caption"
                      color="text.secondary"
                    >
                      {row.source}
                    </Typography>
                  </TableCell>

                  <TableCell>
                    {row.full_name}
                  </TableCell>

                  <TableCell>
                    {row.mobile}
                  </TableCell>

                  <TableCell>
                    {row.pppoe_username}
                  </TableCell>

                  <TableCell>
                    <Chip
                      size="small"
                      label={row.status}
                      color={statusColor(
                        row.status,
                      )}
                    />
                  </TableCell>

                  <TableCell>
                    {new Date(
                      row.requested_at,
                    ).toLocaleString('en-BD')}
                  </TableCell>

                  {!isAgent && (
                    <TableCell align="right">
                      {row.status ===
                        'PENDING' && (
                        <Box
                          sx={{
                            display: 'flex',
                            gap: 1,
                            justifyContent:
                              'flex-end',
                          }}
                        >
                          <Button
                            size="small"
                            variant="contained"
                            disabled={
                              actionID === row.id
                            }
                            onClick={() =>
                              void approve(
                                row.id,
                              )
                            }
                          >
                            Approve
                          </Button>

                          <Button
                            size="small"
                            color="error"
                            disabled={
                              actionID === row.id
                            }
                            onClick={() =>
                              void reject(row)
                            }
                          >
                            Reject
                          </Button>
                        </Box>
                      )}
                    </TableCell>
                  )}
                </TableRow>
              ))}

            {!loading &&
              rows.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={
                      isAgent ? 6 : 7
                    }
                    align="center"
                    sx={{ py: 5 }}
                  >
                    No connection requests found.
                  </TableCell>
                </TableRow>
              )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog
        open={requestOpen}
        onClose={closeNewRequest}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>
          New Customer / PPPoE Request
        </DialogTitle>

        <DialogContent dividers>
          {catalogLoading ? (
            <Box
              sx={{
                py: 8,
                display: 'flex',
                justifyContent: 'center',
              }}
            >
              <CircularProgress />
            </Box>
          ) : (
            <>
              <Tabs
                value={requestTab}
                onChange={(
                  _event,
                  value: number,
                ) => setRequestTab(value)}
                variant="scrollable"
                scrollButtons="auto"
                sx={{ mb: 3 }}
              >
                <Tab label="Basic Information" />
                <Tab label="Service Information" />
                <Tab label="Billing Information" />
              </Tabs>

              {requestTab === 0 && (
                <Box
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                      xs: '1fr',
                      md: '1fr 1fr',
                    },
                    gap: 2,
                  }}
                >
                  <TextField
                    required
                    fullWidth
                    label="Full Name"
                    value={form.full_name}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        full_name:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    required
                    fullWidth
                    label="Mobile"
                    value={form.mobile}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        mobile:
                          event.target.value,
                      })
                    }
                    helperText="Example: 01712345678"
                  />

                  <TextField
                    fullWidth
                    label="Father Name"
                    value={form.father_name}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        father_name:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    label="Mother Name"
                    value={form.mother_name}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        mother_name:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    label="Alternative Mobile"
                    value={form.alt_mobile}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        alt_mobile:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    type="email"
                    label="Email"
                    value={form.email}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        email:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    required
                    fullWidth
                    label="NID"
                    value={form.nid}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        nid: event.target.value,
                      })
                    }
                    helperText="10 to 17 digits"
                  />

                  <TextField
                    fullWidth
                    label="Country"
                    value="Bangladesh"
                    disabled
                  />

                  <TextField
                    select
                    fullWidth
                    label="Division"
                    value={form.division}
                    disabled={locationLoading}
                    onChange={(event) =>
                      void handleDivisionChange(
                        event.target.value,
                      )
                    }
                  >
                    <MenuItem value="">
                      Select Division
                    </MenuItem>

                    {divisions.map(
                      (division) => (
                        <MenuItem
                          key={division.id}
                          value={division.name}
                        >
                          {division.name}
                        </MenuItem>
                      ),
                    )}
                  </TextField>

                  <TextField
                    select
                    fullWidth
                    label="District"
                    value={form.district}
                    disabled={
                      locationLoading ||
                      !form.division
                    }
                    onChange={(event) =>
                      void handleDistrictChange(
                        event.target.value,
                      )
                    }
                  >
                    <MenuItem value="">
                      Select District
                    </MenuItem>

                    {districts.map(
                      (district) => (
                        <MenuItem
                          key={district.id}
                          value={district.name}
                        >
                          {district.name}
                        </MenuItem>
                      ),
                    )}
                  </TextField>

                  <TextField
                    select
                    fullWidth
                    label="Thana / Upazila"
                    value={form.upazila}
                    disabled={
                      locationLoading ||
                      !form.district
                    }
                    onChange={(event) =>
                      void handleUpazilaChange(
                        event.target.value,
                      )
                    }
                  >
                    <MenuItem value="">
                      Select Thana / Upazila
                    </MenuItem>

                    {upazilas.map(
                      (upazila) => (
                        <MenuItem
                          key={upazila.id}
                          value={upazila.name}
                        >
                          {upazila.name}
                        </MenuItem>
                      ),
                    )}
                  </TextField>

                  <TextField
                    select
                    fullWidth
                    label="Post Office / Dakghor"
                    value={form.post_office}
                    disabled={
                      locationLoading ||
                      !form.upazila
                    }
                    onChange={(event) =>
                      handlePostOfficeChange(
                        event.target.value,
                      )
                    }
                  >
                    <MenuItem value="">
                      Select Post Office / Dakghor
                    </MenuItem>

                    {postOffices.map(
                      (postOffice) => (
                        <MenuItem
                          key={postOffice.id}
                          value={postOffice.name}
                        >
                          {postOffice.name}
                          {postOffice.postal_code
                            ? ` — ${postOffice.postal_code}`
                            : ''}
                        </MenuItem>
                      ),
                    )}
                  </TextField>

                  <TextField
                    fullWidth
                    label="Postal Code"
                    value={form.postal_code}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        postal_code:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    label="Road / Para / Mohalla"
                    value={form.road_or_area}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        road_or_area:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    label="Village / Holding"
                    value={
                      form.village_or_holding
                    }
                    onChange={(event) =>
                      setForm({
                        ...form,
                        village_or_holding:
                          event.target.value,
                      })
                    }
                  />

                  <Box
                    sx={{
                      gridColumn: '1 / -1',
                      borderTop: 1,
                      borderColor: 'divider',
                      pt: 2,
                      mt: 1,
                    }}
                  >
                    <Typography
                      variant="subtitle1"
                      sx={{ mb: 2 }}
                    >
                      GPS Location
                    </Typography>

                    <Box
                      sx={{
                        display: 'grid',
                        gridTemplateColumns: {
                          xs: '1fr',
                          md: '1fr 1fr auto',
                        },
                        gap: 2,
                        alignItems: 'start',
                      }}
                    >
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
                          const value =
                            event.target.value

                          setForm((current) => ({
                            ...current,
                            latitude:
                              value === ''
                                ? null
                                : Number(value),
                          }))
                        }}
                      />

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
                          const value =
                            event.target.value

                          setForm((current) => ({
                            ...current,
                            longitude:
                              value === ''
                                ? null
                                : Number(value),
                          }))
                        }}
                      />

                      <Button
                        variant="outlined"
                        onClick={useCurrentLocation}
                        disabled={geoLoading}
                        sx={{
                          minHeight: 56,
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {geoLoading
                          ? 'Locating...'
                          : 'Use Current Location'}
                      </Button>
                    </Box>
                  </Box>
                </Box>
              )}

              {requestTab === 1 && (
                <Box
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                      xs: '1fr',
                      md: '1fr 1fr',
                    },
                    gap: 2,
                  }}
                >
                  <TextField
                    required
                    select
                    fullWidth
                    label="Package"
                    value={
                      form.package_id || ''
                    }
                    onChange={(event) =>
                      setForm({
                        ...form,
                        package_id: Number(
                          event.target.value,
                        ),
                      })
                    }
                  >
                    <MenuItem value="">
                      Select Package
                    </MenuItem>

                    {packages.map((pkg) => (
                      <MenuItem
                        key={pkg.id}
                        value={pkg.id}
                      >
                        {pkg.package_code} —{' '}
                        {pkg.name} — ৳
                        {pkg.price}
                      </MenuItem>
                    ))}
                  </TextField>

                  <TextField
                    select
                    fullWidth
                    label="Router"
                    value={
                      form.router_id || ''
                    }
                    onChange={(event) =>
                      setForm({
                        ...form,
                        router_id:
                          event.target.value
                            ? Number(
                                event.target
                                  .value,
                              )
                            : undefined,
                      })
                    }
                  >
                    <MenuItem value="">
                      No Router / Assign Later
                    </MenuItem>

                    {routers.map(
                      (router) => (
                        <MenuItem
                          key={router.id}
                          value={router.id}
                        >
                          {router.code} —{' '}
                          {router.name}
                          {router.pop_name
                            ? ` — ${router.pop_name}`
                            : ''}
                        </MenuItem>
                      ),
                    )}
                  </TextField>

                  <TextField
                    required
                    fullWidth
                    label="PPPoE Username"
                    value={
                      form.pppoe_username
                    }
                    onChange={(event) =>
                      setForm({
                        ...form,
                        pppoe_username:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    type="password"
                    label="PPPoE Password"
                    value={
                      form.pppoe_password
                    }
                    onChange={(event) =>
                      setForm({
                        ...form,
                        pppoe_password:
                          event.target.value,
                      })
                    }
                  />
                </Box>
              )}

              {requestTab === 2 && (
                <Box
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                      xs: '1fr',
                      md: '1fr 1fr',
                    },
                    gap: 2,
                  }}
                >
                  <TextField
                    required
                    fullWidth
                    type="number"
                    label="Billing Day"
                    value={form.billing_day}
                    slotProps={{
                      htmlInput: {
                        min: 1,
                        max: 31,
                      },
                    }}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        billing_day: Number(
                          event.target.value,
                        ),
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    type="date"
                    label="Activation Date"
                    value={
                      form.activation_date
                    }
                    slotProps={{
                      inputLabel: {
                        shrink: true,
                      },
                    }}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        activation_date:
                          event.target.value,
                      })
                    }
                  />

                  <TextField
                    fullWidth
                    multiline
                    minRows={3}
                    label="Remarks"
                    value={form.remarks}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        remarks:
                          event.target.value,
                      })
                    }
                    sx={{
                      gridColumn: {
                        xs: 'auto',
                        md: '1 / -1',
                      },
                    }}
                  />
                </Box>
              )}
            </>
          )}
        </DialogContent>

        <DialogActions>
          <Button
            onClick={closeNewRequest}
            disabled={saving}
          >
            Cancel
          </Button>

          {requestTab > 0 && (
            <Button
              onClick={() =>
                setRequestTab(
                  (current) => current - 1,
                )
              }
              disabled={saving}
            >
              Back
            </Button>
          )}

          {requestTab < 2 ? (
            <Button
              variant="contained"
              onClick={() =>
                setRequestTab(
                  (current) => current + 1,
                )
              }
              disabled={
                catalogLoading || saving
              }
            >
              Next
            </Button>
          ) : (
            <Button
              variant="contained"
              onClick={() =>
                void submitRequest()
              }
              disabled={
                catalogLoading || saving
              }
            >
              {saving
                ? 'Submitting...'
                : 'Submit for Approval'}
            </Button>
          )}
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default CustomerProvisionRequests
