import { useCallback, useEffect, useMemo, useState } from 'react'
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
import LinkOffIcon from '@mui/icons-material/LinkOff'
import EditIcon from '@mui/icons-material/Edit'
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'
import PauseCircleIcon from '@mui/icons-material/PauseCircle'
import PlayCircleIcon from '@mui/icons-material/PlayCircle'
import AutorenewIcon from '@mui/icons-material/Autorenew'

import {
  getCustomers,
  type Customer,
} from '../../api/customers'
import {
  getPackages,
  type Package,
} from '../../api/packages'
import {
  createSubscription,
  activateSubscription,
  disconnectSubscription,
  getSubscriptions,
  renewSubscription,
  suspendSubscription,
  updateSubscription,
  type CreateSubscriptionRequest,
  type Subscription,
  type UpdateSubscriptionRequest,
} from '../../api/subscriptions'
import { getAPIErrorMessage } from '../../api/errors'
import { getStoredUser } from '../../api/auth'

const getToday = () =>
  new Date().toISOString().slice(0, 10)

const createInitialForm = (): CreateSubscriptionRequest => ({
  customer_id: 0,
  package_id: 0,
  activation_date: getToday(),
  billing_day: 1,
  router_id: 0,
  pppoe_username: '',
  pppoe_password: '',
  remarks: '',
})

function Subscriptions() {
  const [subscriptions, setSubscriptions] = useState<
    Subscription[]
  >([])
  const [customers, setCustomers] = useState<Customer[]>([])
  const [packages, setPackages] = useState<Package[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<
    'ACTIVE' | 'SUSPENDED' | 'EXPIRED' | 'DISCONNECTED' | ''
  >('')
  const [expiringDays, setExpiringDays] = useState(0)

  const [open, setOpen] = useState(false)
  const [editingSubscription, setEditingSubscription] =
    useState<Subscription | null>(null)

  const [createForm, setCreateForm] =
    useState<CreateSubscriptionRequest>(createInitialForm)

  const [updateForm, setUpdateForm] =
    useState<UpdateSubscriptionRequest>({
      billing_day: 1,
      router_id: 0,
      pppoe_username: '',
      pppoe_password: '',
      remarks: '',
    })

  const [disconnectOpen, setDisconnectOpen] = useState(false)
  const [disconnectingSubscription, setDisconnectingSubscription] =
    useState<Subscription | null>(null)
  const [renewingSubscription, setRenewingSubscription] =
    useState<Subscription | null>(null)
  const [renewalMonths, setRenewalMonths] = useState(1)
  const [lifecycleSaving, setLifecycleSaving] = useState(false)
  const isSuperadmin = getStoredUser()?.role === 'superadmin'

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      setError('')

      const [subscriptionData, customerData, packageData] =
        await Promise.all([
          getSubscriptions({
            status: statusFilter,
            expiring_within_days: expiringDays || undefined,
          }),
          getCustomers({ page_size: 100, status: 'ACTIVE' }),
          getPackages(),
        ])

      setSubscriptions(subscriptionData.subscriptions)
      setCustomers(customerData.customers)
      setPackages(packageData.packages)
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(error, 'Failed to load subscription data.'),
      )
    } finally {
      setLoading(false)
    }
  }, [expiringDays, statusFilter])

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadData()
  }, [loadData])

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

  const filteredSubscriptions = useMemo(() => {
    const query = search.trim().toLowerCase()

    if (!query) {
      return subscriptions
    }

    return subscriptions.filter((subscription) => {
      const customer = customerMap.get(
        subscription.customer_id,
      )
      const pkg = packageMap.get(subscription.package_id)

      return [
        subscription.subscription_code,
        subscription.pppoe_username,
        subscription.status,
        subscription.remarks,
        customer?.customer_code,
        customer?.full_name,
        customer?.mobile,
        pkg?.package_code,
        pkg?.name,
      ]
        .join(' ')
        .toLowerCase()
        .includes(query)
    })
  }, [
    subscriptions,
    search,
    customerMap,
    packageMap,
  ])

  const handleOpenCreate = () => {
    setEditingSubscription(null)
    setCreateForm(createInitialForm())
    setError('')
    setOpen(true)
  }

  const handleOpenEdit = (
    subscription: Subscription,
  ) => {
    setEditingSubscription(subscription)
    setError('')

    setUpdateForm({
      billing_day: subscription.billing_day,
      router_id: subscription.router_id,
      pppoe_username: subscription.pppoe_username,
      pppoe_password: '',
      remarks: subscription.remarks,
    })

    setOpen(true)
  }

  const handleCloseDialog = () => {
    if (saving) {
      return
    }

    setOpen(false)
    setEditingSubscription(null)
  }

  const handleCreateChange = (
    field: keyof CreateSubscriptionRequest,
    value: string | number,
  ) => {
    setCreateForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleUpdateChange = (
    field: keyof UpdateSubscriptionRequest,
    value: string | number,
  ) => {
    setUpdateForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault()

    try {
      setSaving(true)
      setError('')

      if (editingSubscription) {
        await updateSubscription(
          editingSubscription.id,
          updateForm,
        )
      } else {
        if (
          !createForm.customer_id ||
          !createForm.package_id ||
          !createForm.pppoe_username.trim() ||
          !createForm.pppoe_password.trim()
        ) {
          return
        }

        await createSubscription({
          ...createForm,
          pppoe_username:
            createForm.pppoe_username.trim(),
          pppoe_password: createForm.pppoe_password,
          remarks: createForm.remarks.trim(),
        })
      }

      setOpen(false)
      setEditingSubscription(null)
      setCreateForm(createInitialForm())

      await loadData()
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          `Failed to ${editingSubscription ? 'update' : 'create'} subscription.`,
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const handleOpenDisconnect = (
    subscription: Subscription,
  ) => {
    setDisconnectingSubscription(subscription)
    setDisconnectOpen(true)
  }

  const handleCloseDisconnect = () => {
    if (disconnecting) {
      return
    }

    setDisconnectOpen(false)
    setDisconnectingSubscription(null)
  }

  const handleDisconnect = async () => {
    if (!disconnectingSubscription) {
      return
    }

    try {
      setDisconnecting(true)
      setError('')

      await disconnectSubscription(disconnectingSubscription.id)

      setDisconnectOpen(false)
      setDisconnectingSubscription(null)

      await loadData()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to disconnect subscription.'))
    } finally {
      setDisconnecting(false)
    }
  }

  const changeLifecycle = async (
    subscription: Subscription,
    action: 'suspend' | 'activate',
  ) => {
    try {
      setLifecycleSaving(true)
      setError('')
      if (action === 'suspend') await suspendSubscription(subscription.id)
      else await activateSubscription(subscription.id)
      await loadData()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, `Failed to ${action} subscription.`))
    } finally {
      setLifecycleSaving(false)
    }
  }

  const handleRenew = async () => {
    if (!renewingSubscription) return
    try {
      setLifecycleSaving(true)
      setError('')
      await renewSubscription(renewingSubscription.id, renewalMonths)
      setRenewingSubscription(null)
      setRenewalMonths(1)
      await loadData()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to renew subscription.'))
    } finally {
      setLifecycleSaving(false)
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
            Subscriptions
          </Typography>

          <Typography
            variant="body1"
            color="text.secondary"
          >
            Manage customer internet subscriptions and PPPoE access.
          </Typography>
        </Box>

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleOpenCreate}
          disabled={loading}
        >
          Add Subscription
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
              placeholder="Search subscriptions..."
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

            <TextField
              select
              size="small"
              label="Status"
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value as typeof statusFilter)}
              sx={{ minWidth: 150 }}
            >
              <MenuItem value="">All statuses</MenuItem>
              <MenuItem value="ACTIVE">Active</MenuItem>
              <MenuItem value="SUSPENDED">Suspended</MenuItem>
              <MenuItem value="EXPIRED">Expired</MenuItem>
              <MenuItem value="DISCONNECTED">Disconnected</MenuItem>
            </TextField>

            <TextField
              select
              size="small"
              label="Expiry"
              value={expiringDays}
              onChange={(event) => setExpiringDays(Number(event.target.value))}
              sx={{ minWidth: 170 }}
            >
              <MenuItem value={0}>Any expiry</MenuItem>
              <MenuItem value={7}>Next 7 days</MenuItem>
              <MenuItem value={30}>Next 30 days</MenuItem>
            </TextField>

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
          ) : filteredSubscriptions.length === 0 ? (
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
                No subscriptions found
              </Typography>

              <Typography
                variant="body2"
                color="text.secondary"
              >
                {search
                  ? 'Try a different search term.'
                  : 'Add your first subscription to get started.'}
              </Typography>
            </Box>
          ) : (
            <TableContainer sx={{ overflowX: 'auto' }}>
              <Table sx={{ minWidth: 1100 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Code</TableCell>
                    <TableCell>Customer</TableCell>
                    <TableCell>Package</TableCell>
                    <TableCell>PPPoE User</TableCell>
                    <TableCell>Billing</TableCell>
                    <TableCell>Expiry</TableCell>
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
                  {filteredSubscriptions.map(
                    (subscription) => (
                      <TableRow
                        key={subscription.id}
                        hover
                      >
                        <TableCell>
                          <Typography
                            sx={{ fontWeight: 600 }}
                          >
                            {subscription.subscription_code}
                          </Typography>
                        </TableCell>

                        <TableCell>
                          <Typography>
                            {getCustomerName(
                              subscription.customer_id,
                            )}
                          </Typography>

                          <Typography
                            variant="body2"
                            color="text.secondary"
                          >
                            {
                              customerMap.get(
                                subscription.customer_id,
                              )?.customer_code
                            }
                          </Typography>
                        </TableCell>

                        <TableCell>
                          {getPackageName(
                            subscription.package_id,
                          )}
                        </TableCell>

                        <TableCell>
                          {subscription.pppoe_username || '-'}
                        </TableCell>

                        <TableCell>
                          Day {subscription.billing_day}
                        </TableCell>

                        <TableCell>
                          <Typography
                            color={
                              new Date(`${subscription.expiry_date}T00:00:00`) < new Date()
                                ? 'error.main'
                                : 'text.primary'
                            }
                          >
                            {subscription.expiry_date}
                          </Typography>
                        </TableCell>

                        <TableCell>
                          <Typography
                            component="span"
                            sx={{
                              fontWeight: 600,
                              color:
                                subscription.status ===
                                'ACTIVE'
                                  ? 'success.main'
                                  : 'text.secondary',
                            }}
                          >
                            {subscription.status}
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
                              handleOpenEdit(subscription)
                            }
                          >
                            <EditIcon />
                          </IconButton>

                          {subscription.status === 'ACTIVE' && (
                            <IconButton
                              color="warning"
                              title="Suspend"
                              disabled={lifecycleSaving}
                              onClick={() => void changeLifecycle(subscription, 'suspend')}
                            >
                              <PauseCircleIcon />
                            </IconButton>
                          )}

                          {subscription.status === 'SUSPENDED' && (
                            <IconButton
                              color="success"
                              title="Activate"
                              disabled={lifecycleSaving}
                              onClick={() => void changeLifecycle(subscription, 'activate')}
                            >
                              <PlayCircleIcon />
                            </IconButton>
                          )}

                          {subscription.status !== 'DISCONNECTED' && (
                            <IconButton
                              color="secondary"
                              title="Renew"
                              disabled={lifecycleSaving}
                              onClick={() => {
                                setRenewingSubscription(subscription)
                                setRenewalMonths(1)
                              }}
                            >
                              <AutorenewIcon />
                            </IconButton>
                          )}

                          {isSuperadmin && subscription.status !== 'DISCONNECTED' && (
                            <IconButton
                              color="error"
                              title="Disconnect"
                              onClick={() =>
                                handleOpenDisconnect(subscription)
                              }
                            >
                              <LinkOffIcon />
                            </IconButton>
                          )}
                        </TableCell>
                      </TableRow>
                    ),
                  )}
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
            {editingSubscription
              ? 'Edit Subscription'
              : 'Add Subscription'}
          </DialogTitle>

          <DialogContent dividers>
            {editingSubscription ? (
              <Grid
                container
                spacing={2}
                sx={{ pt: 1 }}
              >
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    type="number"
                    label="Billing Day"
                    value={updateForm.billing_day}
                    onChange={(event) =>
                      handleUpdateChange(
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
                    fullWidth
                    type="number"
                    label="Router ID"
                    value={updateForm.router_id}
                    onChange={(event) =>
                      handleUpdateChange(
                        'router_id',
                        Number(event.target.value),
                      )
                    }
                    slotProps={{
                      htmlInput: { min: 0 },
                    }}
                  />
                </Grid>

                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    label="PPPoE Username"
                    value={updateForm.pppoe_username}
                    onChange={(event) =>
                      handleUpdateChange(
                        'pppoe_username',
                        event.target.value,
                      )
                    }
                  />
                </Grid>

                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    type="password"
                    label="PPPoE Password"
                    value={updateForm.pppoe_password}
                    onChange={(event) =>
                      handleUpdateChange(
                        'pppoe_password',
                        event.target.value,
                      )
                    }
                  />
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    multiline
                    minRows={3}
                    label="Remarks"
                    value={updateForm.remarks}
                    onChange={(event) =>
                      handleUpdateChange(
                        'remarks',
                        event.target.value,
                      )
                    }
                  />
                </Grid>
              </Grid>
            ) : (
              <Grid
                container
                spacing={2}
                sx={{ pt: 1 }}
              >
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    select
                    label="Customer"
                    value={createForm.customer_id || ''}
                    onChange={(event) =>
                      handleCreateChange(
                        'customer_id',
                        Number(event.target.value),
                      )
                    }
                  >
                    {customers.map((customer) => (
                      <MenuItem
                        key={customer.id}
                        value={customer.id}
                      >
                        {customer.customer_code} -{' '}
                        {customer.full_name}
                      </MenuItem>
                    ))}
                  </TextField>
                </Grid>

                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    select
                    label="Package"
                    value={createForm.package_id || ''}
                    onChange={(event) =>
                      handleCreateChange(
                        'package_id',
                        Number(event.target.value),
                      )
                    }
                  >
                    {packages.map((pkg) => (
                      <MenuItem key={pkg.id} value={pkg.id}>
                        {pkg.package_code} - {pkg.name}
                      </MenuItem>
                    ))}
                  </TextField>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <TextField
                    fullWidth
                    required
                    type="date"
                    label="Activation Date"
                    value={createForm.activation_date}
                    onChange={(event) =>
                      handleCreateChange(
                        'activation_date',
                        event.target.value,
                      )
                    }
                    slotProps={{
                      inputLabel: { shrink: true },
                    }}
                  />
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <TextField
                    fullWidth
                    required
                    type="number"
                    label="Billing Day"
                    value={createForm.billing_day}
                    onChange={(event) =>
                      handleCreateChange(
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

                <Grid size={{ xs: 12, md: 4 }}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Router ID"
                    value={createForm.router_id}
                    onChange={(event) =>
                      handleCreateChange(
                        'router_id',
                        Number(event.target.value),
                      )
                    }
                    slotProps={{
                      htmlInput: { min: 0 },
                    }}
                  />
                </Grid>

                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    label="PPPoE Username"
                    value={createForm.pppoe_username}
                    onChange={(event) =>
                      handleCreateChange(
                        'pppoe_username',
                        event.target.value,
                      )
                    }
                  />
                </Grid>

                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    type="password"
                    label="PPPoE Password"
                    value={createForm.pppoe_password}
                    onChange={(event) =>
                      handleCreateChange(
                        'pppoe_password',
                        event.target.value,
                      )
                    }
                  />
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    multiline
                    minRows={3}
                    label="Remarks"
                    value={createForm.remarks}
                    onChange={(event) =>
                      handleCreateChange(
                        'remarks',
                        event.target.value,
                      )
                    }
                  />
                </Grid>
              </Grid>
            )}
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
              disabled={
                saving ||
                (!editingSubscription &&
                  (!createForm.customer_id ||
                    !createForm.package_id ||
                    !createForm.pppoe_username.trim() ||
                    !createForm.pppoe_password.trim()))
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
                ? 'Saving...'
                : editingSubscription
                  ? 'Update Subscription'
                  : 'Create Subscription'}
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog
        open={Boolean(renewingSubscription)}
        onClose={() => !lifecycleSaving && setRenewingSubscription(null)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>Renew subscription</DialogTitle>
        <DialogContent dividers>
          <Typography sx={{ mb: 2 }}>
            {renewingSubscription?.subscription_code} currently expires on{' '}
            {renewingSubscription?.expiry_date}.
          </Typography>
          <TextField
            fullWidth
            required
            type="number"
            label="Renewal months"
            value={renewalMonths}
            onChange={(event) => setRenewalMonths(Number(event.target.value))}
            slotProps={{ htmlInput: { min: 1, max: 12 } }}
          />
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setRenewingSubscription(null)}
            disabled={lifecycleSaving}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={() => void handleRenew()}
            disabled={lifecycleSaving || renewalMonths < 1 || renewalMonths > 12}
            startIcon={lifecycleSaving ? <CircularProgress size={18} /> : <AutorenewIcon />}
          >
            {lifecycleSaving ? 'Renewing...' : 'Renew'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={disconnectOpen}
        onClose={handleCloseDisconnect}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>Disconnect Subscription</DialogTitle>

        <DialogContent>
          <Typography>
            Disconnect{' '}
            <strong>
              {disconnectingSubscription?.subscription_code}
            </strong>
            ? Its invoices, payments, and history will be preserved. A
            disconnected subscription cannot be renewed or activated.
          </Typography>
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button
            onClick={handleCloseDisconnect}
            disabled={disconnecting}
          >
            Cancel
          </Button>

          <Button
            variant="contained"
            color="error"
            onClick={() => void handleDisconnect()}
            disabled={disconnecting}
            startIcon={
              disconnecting ? (
                <CircularProgress size={18} />
              ) : (
                <LinkOffIcon />
              )
            }
          >
            {disconnecting ? 'Disconnecting...' : 'Disconnect'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default Subscriptions
