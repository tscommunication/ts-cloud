import {
  useCallback,
  useEffect,
  useState,
  type FormEvent,
} from 'react'

import { useLocation } from 'react-router-dom'

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  ListItemText,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TablePagination,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'

import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import ToggleOffIcon from '@mui/icons-material/ToggleOff'
import ToggleOnIcon from '@mui/icons-material/ToggleOn'
import SwapHorizIcon from '@mui/icons-material/SwapHoriz'
import DeleteIcon from '@mui/icons-material/Delete'

import {
  createAgent,
  createPOP,
  deleteAgent,
  deletePOP,
  getAgents,
  getArchivedAgents,
  getArchivedPOPs,
  getPOPs,
  migrateAgent,
  migratePOP,
  restoreAgent,
  restorePOP,
  setAgentStatus,
  setPOPStatus,
  updateAgent,
  updatePOP,
  type Agent,
  type AgentInput,
  type POP,
  type POPInput,
} from '../../api/distribution'

import { getAPIErrorMessage } from '../../api/errors'
import { getStoredUser } from '../../api/auth'

const emptyPOP: POPInput = {
  code: '',
  name: '',
  manager_name: '',
  mobile: '',
  address: '',
}

const emptyAgent: AgentInput = {
  code: '',
  name: '',
  pop_id: 0,
  pop_ids: [],
  mobile: '',
  address: '',
  commission_percent: 0,
}

export default function Organization() {
  const location = useLocation()
  const showPOPs = location.pathname.endsWith('/pops')
  const showAgents = location.pathname.endsWith('/agents')

  const [pops, setPOPs] = useState<POP[]>([])
  const [agents, setAgents] = useState<Agent[]>([])

  const [archivedPOPs, setArchivedPOPs] =
    useState<POP[]>([])
  const [archivedAgents, setArchivedAgents] =
    useState<Agent[]>([])

  const [showArchivedPOPs, setShowArchivedPOPs] =
    useState(false)
  const [showArchivedAgents, setShowArchivedAgents] =
    useState(false)

  const [popSearch, setPOPSearch] = useState('')
  const [popStatusFilter, setPOPStatusFilter] =
    useState<'ALL' | POP['status']>('ALL')
  const [popPage, setPOPPage] = useState(0)
  const [popRowsPerPage, setPOPRowsPerPage] = useState(10)

  const [agentSearch, setAgentSearch] = useState('')
  const [agentStatusFilter, setAgentStatusFilter] =
    useState<'ALL' | Agent['status']>('ALL')
  const [agentPage, setAgentPage] = useState(0)
  const [agentRowsPerPage, setAgentRowsPerPage] = useState(10)

  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [saving, setSaving] = useState(false)

  const [popDialog, setPOPDialog] = useState(false)
  const [agentDialog, setAgentDialog] = useState(false)

  const [editingPOP, setEditingPOP] =
    useState<POP | null>(null)

  const [editingAgent, setEditingAgent] =
    useState<Agent | null>(null)

  const [popForm, setPOPForm] =
    useState<POPInput>(emptyPOP)

  const [agentForm, setAgentForm] =
    useState<AgentInput>(emptyAgent)

  /*
   * Agent Migration
   */
  const [migrationSource, setMigrationSource] =
    useState<Agent | null>(null)

  const [migrationTargetID, setMigrationTargetID] =
    useState(0)

  const [deleteTarget, setDeleteTarget] =
    useState<Agent | null>(null)

  /*
   * POP Migration
   */
  const [popMigrationSource, setPOPMigrationSource] =
    useState<POP | null>(null)

  const [
    popMigrationTargetID,
    setPOPMigrationTargetID,
  ] = useState(0)

  const [popDeleteTarget, setPOPDeleteTarget] =
    useState<POP | null>(null)

  const isSuperadmin =
    getStoredUser()?.role === 'superadmin'

  const visiblePOPs = showArchivedPOPs
    ? archivedPOPs
    : pops

  const visibleAgents = showArchivedAgents
    ? archivedAgents
    : agents

  const normalizedPOPSearch = popSearch.trim().toLowerCase()

  const filteredPOPs = visiblePOPs.filter((pop) => {
    if (
      popStatusFilter !== 'ALL' &&
      pop.status !== popStatusFilter
    ) {
      return false
    }

    if (!normalizedPOPSearch) {
      return true
    }

    return [
      pop.code,
      pop.name,
      pop.address,
      pop.manager_name,
      pop.mobile,
    ].some((value) =>
      String(value ?? '')
        .toLowerCase()
        .includes(normalizedPOPSearch),
    )
  })

  const popPageCount = Math.max(
    1,
    Math.ceil(filteredPOPs.length / popRowsPerPage),
  )

  const displayPOPPage = Math.min(
    popPage,
    popPageCount - 1,
  )

  const paginatedPOPs = filteredPOPs.slice(
    displayPOPPage * popRowsPerPage,
    displayPOPPage * popRowsPerPage + popRowsPerPage,
  )

  const normalizedAgentSearch =
    agentSearch.trim().toLowerCase()

  const filteredAgents = visibleAgents.filter((agent) => {
    if (
      agentStatusFilter !== 'ALL' &&
      agent.status !== agentStatusFilter
    ) {
      return false
    }

    if (!normalizedAgentSearch) {
      return true
    }

    return [
      agent.code,
      agent.name,
      agent.pop_name,
      agent.pop_names?.join(' ') ?? '',
      agent.mobile,
    ].some((value) =>
      String(value ?? '')
        .toLowerCase()
        .includes(normalizedAgentSearch),
    )
  })

  const agentPageCount = Math.max(
    1,
    Math.ceil(filteredAgents.length / agentRowsPerPage),
  )

  const displayAgentPage = Math.min(
    agentPage,
    agentPageCount - 1,
  )

  const paginatedAgents = filteredAgents.slice(
    displayAgentPage * agentRowsPerPage,
    displayAgentPage * agentRowsPerPage +
      agentRowsPerPage,
  )

  const load = useCallback(async () => {
    try {
      setError('')

      const [
        popRows,
        agentRows,
        archivedPOPRows,
        archivedAgentRows,
      ] = await Promise.all([
        getPOPs(),
        getAgents(),
        isSuperadmin
          ? getArchivedPOPs()
          : Promise.resolve<POP[]>([]),
        isSuperadmin
          ? getArchivedAgents()
          : Promise.resolve<Agent[]>([]),
      ])

      setPOPs(popRows)
      setAgents(agentRows)
      setArchivedPOPs(archivedPOPRows)
      setArchivedAgents(archivedAgentRows)
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to load organization data.',
        ),
      )
    }
  }, [isSuperadmin])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load()
  }, [load])

  const openPOP = (row?: POP) => {
    setEditingPOP(row ?? null)

    setPOPForm(
      row
        ? {
            code: row.code,
            name: row.name,
            manager_name: row.manager_name,
            mobile: row.mobile,
            address: row.address,
          }
        : emptyPOP,
    )

    setPOPDialog(true)
  }

  const openAgent = (row?: Agent) => {
    const firstPOP =
      pops.find(
        (pop) => pop.status === 'ACTIVE',
      )?.id ?? 0

    setEditingAgent(row ?? null)

    setAgentForm(
      row
        ? {
            code: row.code,
            name: row.name,
            pop_id: row.pop_id,
            pop_ids: row.pop_ids?.length
              ? row.pop_ids
              : [row.pop_id],
            mobile: row.mobile,
            address: row.address,
            commission_percent:
              row.commission_percent,
          }
        : {
            ...emptyAgent,
            pop_id: firstPOP,
            pop_ids: firstPOP
              ? [firstPOP]
              : [],
          },
    )

    setAgentDialog(true)
  }

  const savePOP = async (
    event: FormEvent,
  ) => {
    event.preventDefault()

    try {
      setSaving(true)
      setError('')

      if (editingPOP) {
        await updatePOP(
          editingPOP.id,
          popForm,
        )
      } else {
        await createPOP(popForm)
      }

      setPOPDialog(false)
      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to save POP.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const saveAgent = async (
    event: FormEvent,
  ) => {
    event.preventDefault()

    try {
      setSaving(true)
      setError('')

      if (editingAgent) {
        await updateAgent(
          editingAgent.id,
          agentForm,
        )
      } else {
        await createAgent(agentForm)
      }

      setAgentDialog(false)
      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to save agent.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  /*
   * Agent Migration
   */
  const migrate = async () => {
    if (
      !migrationSource ||
      !migrationTargetID
    ) {
      return
    }

    try {
      setSaving(true)
      setError('')
      setSuccess('')

      const result = await migrateAgent(
        migrationSource.id,
        migrationTargetID,
      )

      setMigrationSource(null)
      setMigrationTargetID(0)

      setSuccess(
        `Agent migrated: ${result.customers_migrated} customers, ` +
          `${result.login_users_migrated} login users and ` +
          `${result.pops_migrated} POP links moved.`,
      )

      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to migrate agent.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const removeAgent = async () => {
    if (!deleteTarget) {
      return
    }

    try {
      setSaving(true)
      setError('')
      setSuccess('')

      await deleteAgent(deleteTarget.id)

      setDeleteTarget(null)

      setSuccess(
        'Agent archived successfully.',
      )

      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Agent has dependencies; migrate it before deletion.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const restoreArchivedAgent = async (
    row: Agent,
  ) => {
    try {
      setSaving(true)
      setError('')
      setSuccess('')

      await restoreAgent(row.id)

      setSuccess(
        `${row.code} — ${row.name} restored as INACTIVE.`,
      )

      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to restore Agent / Reseller.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  /*
   * POP Migration
   */
  const migrateSelectedPOP = async () => {
    if (
      !popMigrationSource ||
      !popMigrationTargetID
    ) {
      return
    }

    try {
      setSaving(true)
      setError('')
      setSuccess('')

      const result = await migratePOP(
        popMigrationSource.id,
        popMigrationTargetID,
      )

      setPOPMigrationSource(null)
      setPOPMigrationTargetID(0)

      setSuccess(
        `POP migrated successfully: ` +
          `${result.customers_migrated} customers, ` +
          `${result.primary_agents_migrated} primary agents, ` +
          `${result.agent_memberships_migrated} agent POP links and ` +
          `${result.routers_migrated} network routers moved.`,
      )

      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to migrate POP.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const removePOP = async () => {
    if (!popDeleteTarget) {
      return
    }

    try {
      setSaving(true)
      setError('')
      setSuccess('')

      await deletePOP(
        popDeleteTarget.id,
      )

      setPOPDeleteTarget(null)

      setSuccess(
        'POP archived successfully.',
      )

      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'POP has linked customers, agents or network routers; migrate it before deletion.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const restoreArchivedPOP = async (
    row: POP,
  ) => {
    try {
      setSaving(true)
      setError('')
      setSuccess('')

      await restorePOP(row.id)

      setSuccess(
        `${row.code} — ${row.name} restored as INACTIVE.`,
      )

      await load()
    } catch (err) {
      setError(
        getAPIErrorMessage(
          err,
          'Failed to restore POP.',
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          justifyContent:
            'space-between',
          alignItems: 'center',
          mb: 3,
          gap: 2,
        }}
      >
        <Box>
          <Typography
            variant="h4"
            sx={{ fontWeight: 700 }}
          >
            {showPOPs ? 'POPs' : 'Agents / Resellers'}
          </Typography>

          <Typography color="text.secondary">
            {showPOPs
              ? 'Manage POP locations, assignments, migration and status.'
              : 'Manage Agents / Resellers, POP assignments, migration and account status.'}
          </Typography>
        </Box>
      </Box>

      {error && (
        <Alert
          severity="error"
          onClose={() => setError('')}
          sx={{ mb: 2 }}
        >
          {error}
        </Alert>
      )}

      {success && (
        <Alert
          severity="success"
          onClose={() => setSuccess('')}
          sx={{ mb: 2 }}
        >
          {success}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* POPS */}
        <Grid
          size={{ xs: 12 }}
          sx={{
            display: showPOPs ? 'block' : 'none',
          }}
        >
          <Card>
            <CardContent>
              <Box
                sx={{
                  display: 'flex',
                  justifyContent:
                    'space-between',
                  mb: 2,
                }}
              >
                <Typography variant="h6">
                  {showArchivedPOPs
                    ? `Archived POPs (${archivedPOPs.length})`
                    : `POPs (${pops.length})`}
                </Typography>

                <Box
                  sx={{
                    display: 'flex',
                    gap: 1,
                    flexWrap: 'wrap',
                    justifyContent: 'flex-end',
                  }}
                >
                  {isSuperadmin && (
                    <>
                      <Button
                        size="small"
                        variant={
                          showArchivedPOPs
                            ? 'outlined'
                            : 'contained'
                        }
                        onClick={() => {
                          setShowArchivedPOPs(false)
                          setPOPPage(0)
                        }}
                      >
                        Current
                      </Button>

                      <Button
                        size="small"
                        variant={
                          showArchivedPOPs
                            ? 'contained'
                            : 'outlined'
                        }
                        onClick={() => {
                          setShowArchivedPOPs(true)
                          setPOPPage(0)
                        }}
                      >
                        Archived ({archivedPOPs.length})
                      </Button>
                    </>
                  )}

                  {isSuperadmin && !showArchivedPOPs && (
                    <Button
                      startIcon={<AddIcon />}
                      variant="contained"
                      onClick={() =>
                        openPOP()
                      }
                    >
                      Add POP
                    </Button>
                  )}
                </Box>
              </Box>

              <Box
                sx={{
                  display: 'flex',
                  flexWrap: 'wrap',
                  alignItems: 'center',
                  gap: 2,
                  mb: 2,
                }}
              >
                <TextField
                  size="small"
                  label="Search POPs"
                  placeholder="Code, name, manager, mobile, location..."
                  value={popSearch}
                  onChange={(e) => {
                    setPOPSearch(e.target.value)
                    setPOPPage(0)
                  }}
                  sx={{
                    minWidth: {
                      xs: '100%',
                      sm: 320,
                    },
                  }}
                />

                <TextField
                  select
                  size="small"
                  label="Status"
                  value={popStatusFilter}
                  onChange={(e) => {
                    setPOPStatusFilter(
                      e.target.value as
                        | 'ALL'
                        | POP['status'],
                    )
                    setPOPPage(0)
                  }}
                  sx={{ minWidth: 160 }}
                >
                  <MenuItem value="ALL">
                    All Statuses
                  </MenuItem>
                  <MenuItem value="ACTIVE">
                    Active
                  </MenuItem>
                  <MenuItem value="INACTIVE">
                    Inactive
                  </MenuItem>
                </TextField>

                <Box sx={{ flexGrow: 1 }} />

                <Typography
                  variant="body2"
                  color="text.secondary"
                >
                  Showing {filteredPOPs.length} of{' '}
                  {visiblePOPs.length}
                </Typography>
              </Box>

              <Box
                sx={{
                  overflowX: 'auto',
                }}
              >
                <Table
                  size="small"
                  sx={{ minWidth: 760 }}
                >
                  <TableHead>
                    <TableRow>
                      <TableCell
                        sx={{ width: 70 }}
                      >
                        #
                      </TableCell>

                      <TableCell>
                        Code / Name
                      </TableCell>

                      <TableCell>
                        Location
                      </TableCell>

                      <TableCell>
                        Manager / Contact
                      </TableCell>

                      <TableCell>
                        Status
                      </TableCell>

                      <TableCell align="right">
                        Actions
                      </TableCell>
                    </TableRow>
                  </TableHead>

                  <TableBody>
                    {paginatedPOPs.map((row, index) => (
                      <TableRow key={row.id}>
                        <TableCell>
                          {displayPOPPage *
                            popRowsPerPage +
                            index +
                            1}
                        </TableCell>

                        <TableCell>
                          <b>{row.code}</b>
                          <br />
                          {row.name}
                        </TableCell>

                        <TableCell>
                          {row.address || '—'}
                        </TableCell>

                        <TableCell>
                          {row.manager_name ||
                            '—'}
                          <br />
                          {row.mobile}
                        </TableCell>

                        <TableCell>
                          <Chip
                            size="small"
                            color={
                              showArchivedPOPs
                                ? 'warning'
                                : row.status ===
                                    'ACTIVE'
                                  ? 'success'
                                  : 'default'
                            }
                            label={
                              showArchivedPOPs
                                ? 'ARCHIVED'
                                : row.status
                            }
                          />

                          {showArchivedPOPs &&
                            row.deleted_at && (
                              <Typography
                                variant="caption"
                                color="text.secondary"
                                sx={{
                                  display: 'block',
                                  mt: 0.5,
                                }}
                              >
                                {new Date(
                                  row.deleted_at,
                                ).toLocaleString()}
                              </Typography>
                            )}
                        </TableCell>

                        <TableCell align="right">
                          {isSuperadmin &&
                            showArchivedPOPs && (
                              <Button
                                size="small"
                                variant="outlined"
                                disabled={saving}
                                onClick={() =>
                                  void restoreArchivedPOP(
                                    row,
                                  )
                                }
                              >
                                Restore
                              </Button>
                            )}

                          {isSuperadmin &&
                            !showArchivedPOPs && (
                            <>
                              <Tooltip title="Edit POP">
                                <IconButton
                                  onClick={() =>
                                    openPOP(row)
                                  }
                                >
                                  <EditIcon />
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Migrate POP">
                                <IconButton
                                  color="primary"
                                  onClick={() => {
                                    setPOPMigrationSource(
                                      row,
                                    )
                                    setPOPMigrationTargetID(
                                      0,
                                    )
                                    setError('')
                                    setSuccess('')
                                  }}
                                >
                                  <SwapHorizIcon />
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Toggle status">
                                <IconButton
                                  onClick={async () => {
                                    await setPOPStatus(
                                      row.id,
                                      row.status ===
                                        'ACTIVE'
                                        ? 'INACTIVE'
                                        : 'ACTIVE',
                                    )

                                    await load()
                                  }}
                                >
                                  {row.status ===
                                  'ACTIVE' ? (
                                    <ToggleOnIcon color="success" />
                                  ) : (
                                    <ToggleOffIcon />
                                  )}
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Archive POP">
                                <IconButton
                                  color="error"
                                  onClick={() =>
                                    setPOPDeleteTarget(
                                      row,
                                    )
                                  }
                                >
                                  <DeleteIcon />
                                </IconButton>
                              </Tooltip>
                            </>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}

                    {!filteredPOPs.length && (
                      <TableRow>
                        <TableCell
                          colSpan={6}
                          align="center"
                        >
                          No POP matches the current filters.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </Box>

              <TablePagination
                component="div"
                count={filteredPOPs.length}
                page={displayPOPPage}
                rowsPerPage={popRowsPerPage}
                rowsPerPageOptions={[10, 25, 50]}
                onPageChange={(_, nextPage) =>
                  setPOPPage(nextPage)
                }
                onRowsPerPageChange={(e) => {
                  setPOPRowsPerPage(
                    Number(e.target.value),
                  )
                  setPOPPage(0)
                }}
                labelRowsPerPage="Rows per page:"
              />
            </CardContent>
          </Card>
        </Grid>

        {/* AGENTS */}
        <Grid
          size={{ xs: 12 }}
          sx={{
            display: showAgents ? 'block' : 'none',
          }}
        >
          <Card>
            <CardContent>
              <Box
                sx={{
                  display: 'flex',
                  justifyContent:
                    'space-between',
                  mb: 2,
                }}
              >
                <Typography variant="h6">
                  {showArchivedAgents
                    ? `Archived Agents / Resellers (${archivedAgents.length})`
                    : `Agents / Resellers (${agents.length})`}
                </Typography>

                <Box
                  sx={{
                    display: 'flex',
                    gap: 1,
                    flexWrap: 'wrap',
                    justifyContent: 'flex-end',
                  }}
                >
                  {isSuperadmin && (
                    <>
                      <Button
                        size="small"
                        variant={
                          showArchivedAgents
                            ? 'outlined'
                            : 'contained'
                        }
                        onClick={() => {
                          setShowArchivedAgents(false)
                          setAgentPage(0)
                        }}
                      >
                        Current
                      </Button>

                      <Button
                        size="small"
                        variant={
                          showArchivedAgents
                            ? 'contained'
                            : 'outlined'
                        }
                        onClick={() => {
                          setShowArchivedAgents(true)
                          setAgentPage(0)
                        }}
                      >
                        Archived ({archivedAgents.length})
                      </Button>
                    </>
                  )}

                  {!showArchivedAgents && (
                    <Button
                      startIcon={<AddIcon />}
                      variant="contained"
                      disabled={
                        !pops.some(
                          (row) =>
                            row.status ===
                            'ACTIVE',
                        )
                      }
                      onClick={() =>
                        openAgent()
                      }
                    >
                      Add Agent
                    </Button>
                  )}
                </Box>
              </Box>

              <Box
                sx={{
                  display: 'flex',
                  flexWrap: 'wrap',
                  alignItems: 'center',
                  gap: 2,
                  mb: 2,
                }}
              >
                <TextField
                  size="small"
                  label="Search Agents"
                  placeholder="Code, name, POP, mobile..."
                  value={agentSearch}
                  onChange={(e) => {
                    setAgentSearch(e.target.value)
                    setAgentPage(0)
                  }}
                  sx={{
                    minWidth: {
                      xs: '100%',
                      sm: 320,
                    },
                  }}
                />

                <TextField
                  select
                  size="small"
                  label="Status"
                  value={agentStatusFilter}
                  onChange={(e) => {
                    setAgentStatusFilter(
                      e.target.value as
                        | 'ALL'
                        | Agent['status'],
                    )
                    setAgentPage(0)
                  }}
                  sx={{ minWidth: 160 }}
                >
                  <MenuItem value="ALL">
                    All Statuses
                  </MenuItem>
                  <MenuItem value="ACTIVE">
                    Active
                  </MenuItem>
                  <MenuItem value="INACTIVE">
                    Inactive
                  </MenuItem>
                </TextField>

                <Box sx={{ flexGrow: 1 }} />

                <Typography
                  variant="body2"
                  color="text.secondary"
                >
                  Showing {filteredAgents.length} of{' '}
                  {visibleAgents.length}
                </Typography>
              </Box>

              <Box
                sx={{
                  overflowX: 'auto',
                }}
              >
                <Table
                  size="small"
                  sx={{ minWidth: 820 }}
                >
                  <TableHead>
                    <TableRow>
                      <TableCell
                        sx={{ width: 70 }}
                      >
                        #
                      </TableCell>

                      <TableCell>
                        Code / Name
                      </TableCell>

                      <TableCell>
                        Primary POP
                      </TableCell>

                      <TableCell>
                        Contact
                      </TableCell>

                      <TableCell>
                        Opening Balance
                      </TableCell>

                      <TableCell>
                        Commission
                      </TableCell>

                      <TableCell>
                        Status
                      </TableCell>

                      <TableCell align="right">
                        Actions
                      </TableCell>
                    </TableRow>
                  </TableHead>

                  <TableBody>
                    {paginatedAgents.map((row, index) => (
                      <TableRow key={row.id}>
                        <TableCell>
                          {displayAgentPage *
                            agentRowsPerPage +
                            index +
                            1}
                        </TableCell>

                        <TableCell>
                          <b>{row.code}</b>
                          <br />
                          {row.name}
                        </TableCell>

                        <TableCell>
                          {row.pop_names?.length
                            ? row.pop_names.join(
                                ', ',
                              )
                            : row.pop_name}
                        </TableCell>

                        <TableCell>
                          {row.mobile || '—'}
                        </TableCell>

                        <TableCell>
                          BDT{' '}
                          {row.opening_balance.toLocaleString()}
                        </TableCell>

                        <TableCell>
                          {
                            row.commission_percent
                          }
                          %
                        </TableCell>

                        <TableCell>
                          <Chip
                            size="small"
                            color={
                              showArchivedAgents
                                ? 'warning'
                                : row.status ===
                                    'ACTIVE'
                                  ? 'success'
                                  : 'default'
                            }
                            label={
                              showArchivedAgents
                                ? 'ARCHIVED'
                                : row.status
                            }
                          />

                          {showArchivedAgents &&
                            row.deleted_at && (
                              <Typography
                                variant="caption"
                                color="text.secondary"
                                sx={{
                                  display: 'block',
                                  mt: 0.5,
                                }}
                              >
                                {new Date(
                                  row.deleted_at,
                                ).toLocaleString()}
                              </Typography>
                            )}
                        </TableCell>

                        <TableCell align="right">
                          {showArchivedAgents ? (
                            isSuperadmin && (
                              <Button
                                size="small"
                                variant="outlined"
                                disabled={saving}
                                onClick={() =>
                                  void restoreArchivedAgent(
                                    row,
                                  )
                                }
                              >
                                Restore
                              </Button>
                            )
                          ) : (
                            <>
                              <Tooltip title="Edit">
                            <IconButton
                              onClick={() =>
                                openAgent(row)
                              }
                            >
                              <EditIcon />
                            </IconButton>
                          </Tooltip>

                          {isSuperadmin && (
                            <>
                              <Tooltip title="Migrate to another agent">
                                <IconButton
                                  color="primary"
                                  onClick={() => {
                                    setMigrationSource(
                                      row,
                                    )
                                    setMigrationTargetID(
                                      0,
                                    )
                                    setError('')
                                    setSuccess('')
                                  }}
                                >
                                  <SwapHorizIcon />
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Toggle status">
                                <IconButton
                                  onClick={async () => {
                                    await setAgentStatus(
                                      row.id,
                                      row.status ===
                                        'ACTIVE'
                                        ? 'INACTIVE'
                                        : 'ACTIVE',
                                    )

                                    await load()
                                  }}
                                >
                                  {row.status ===
                                  'ACTIVE' ? (
                                    <ToggleOnIcon color="success" />
                                  ) : (
                                    <ToggleOffIcon />
                                  )}
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Archive Agent">
                                <IconButton
                                  color="error"
                                  onClick={() =>
                                    setDeleteTarget(
                                      row,
                                    )
                                  }
                                >
                                  <DeleteIcon />
                                </IconButton>
                              </Tooltip>
                            </>
                          )}
                            </>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}

                    {!filteredAgents.length && (
                      <TableRow>
                        <TableCell
                          colSpan={8}
                          align="center"
                        >
                          No Agent / Reseller matches the current filters.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </Box>

              <TablePagination
                component="div"
                count={filteredAgents.length}
                page={displayAgentPage}
                rowsPerPage={agentRowsPerPage}
                rowsPerPageOptions={[10, 25, 50]}
                onPageChange={(_, nextPage) =>
                  setAgentPage(nextPage)
                }
                onRowsPerPageChange={(e) => {
                  setAgentRowsPerPage(
                    Number(e.target.value),
                  )
                  setAgentPage(0)
                }}
                labelRowsPerPage="Rows per page:"
              />
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* POP CREATE / EDIT */}
      <Dialog
        open={popDialog}
        onClose={() =>
          setPOPDialog(false)
        }
        fullWidth
        maxWidth="sm"
      >
        <Box
          component="form"
          onSubmit={savePOP}
        >
          <DialogTitle>
            {editingPOP
              ? 'Edit POP'
              : 'Add POP'}
          </DialogTitle>

          <DialogContent>
            <Grid
              container
              spacing={2}
              sx={{ mt: 0 }}
            >
              <Grid size={6}>
                <TextField
                  fullWidth
                  required
                  label="Code"
                  disabled={!!editingPOP}
                  value={popForm.code}
                  onChange={(e) =>
                    setPOPForm({
                      ...popForm,
                      code: e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={6}>
                <TextField
                  fullWidth
                  required
                  label="Name"
                  value={popForm.name}
                  onChange={(e) =>
                    setPOPForm({
                      ...popForm,
                      name: e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={6}>
                <TextField
                  fullWidth
                  label="Manager"
                  value={
                    popForm.manager_name
                  }
                  onChange={(e) =>
                    setPOPForm({
                      ...popForm,
                      manager_name:
                        e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={6}>
                <TextField
                  fullWidth
                  label="Mobile"
                  value={popForm.mobile}
                  onChange={(e) =>
                    setPOPForm({
                      ...popForm,
                      mobile: e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={12}>
                <TextField
                  fullWidth
                  multiline
                  label="Address"
                  value={popForm.address}
                  onChange={(e) =>
                    setPOPForm({
                      ...popForm,
                      address: e.target.value,
                    })
                  }
                />
              </Grid>
            </Grid>
          </DialogContent>

          <DialogActions>
            <Button
              onClick={() =>
                setPOPDialog(false)
              }
            >
              Cancel
            </Button>

            <Button
              type="submit"
              variant="contained"
              disabled={saving}
            >
              Save
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      {/* AGENT CREATE / EDIT */}
      <Dialog
        open={agentDialog}
        onClose={() =>
          setAgentDialog(false)
        }
        fullWidth
        maxWidth="sm"
      >
        <Box
          component="form"
          onSubmit={saveAgent}
        >
          <DialogTitle>
            {editingAgent
              ? 'Edit Agent'
              : 'Add Agent'}
          </DialogTitle>

          <DialogContent>
            <Grid
              container
              spacing={2}
              sx={{ mt: 0 }}
            >
              <Grid size={6}>
                <TextField
                  fullWidth
                  required
                  label="Code"
                  disabled={
                    !!editingAgent
                  }
                  value={agentForm.code}
                  onChange={(e) =>
                    setAgentForm({
                      ...agentForm,
                      code: e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={6}>
                <TextField
                  fullWidth
                  required
                  label="Name"
                  value={agentForm.name}
                  onChange={(e) =>
                    setAgentForm({
                      ...agentForm,
                      name: e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={12}>
                <FormControl
                  fullWidth
                  required
                >
                  <InputLabel>
                    POP Locations
                  </InputLabel>

                  <Select
                    multiple
                    label="POP Locations"
                    value={
                      agentForm.pop_ids
                    }
                    onChange={(e) => {
                      const ids = (
                        e.target
                          .value as number[]
                      ).map(Number)

                      setAgentForm({
                        ...agentForm,
                        pop_ids: ids,
                        pop_id:
                          ids[0] ?? 0,
                      })
                    }}
                    renderValue={(
                      selected,
                    ) =>
                      selected
                        .map(
                          (id) =>
                            pops.find(
                              (pop) =>
                                pop.id ===
                                id,
                            )?.name ?? id,
                        )
                        .join(', ')
                    }
                  >
                    {pops
                      .filter(
                        (row) =>
                          row.status ===
                            'ACTIVE' ||
                          agentForm.pop_ids.includes(
                            row.id,
                          ),
                      )
                      .map((row) => (
                        <MenuItem
                          key={row.id}
                          value={row.id}
                        >
                          <Checkbox
                            checked={agentForm.pop_ids.includes(
                              row.id,
                            )}
                          />

                          <ListItemText
                            primary={`${row.code} — ${row.name}`}
                            secondary={
                              row.address
                            }
                          />
                        </MenuItem>
                      ))}
                  </Select>
                </FormControl>
              </Grid>

              <Grid size={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Commission %"
                  slotProps={{
                    htmlInput: {
                      min: 0,
                      max: 100,
                      step: 0.01,
                    },
                  }}
                  value={
                    agentForm.commission_percent
                  }
                  onChange={(e) =>
                    setAgentForm({
                      ...agentForm,
                      commission_percent:
                        Number(
                          e.target.value,
                        ),
                    })
                  }
                />
              </Grid>

              <Grid size={6}>
                <TextField
                  fullWidth
                  label="Mobile"
                  value={agentForm.mobile}
                  onChange={(e) =>
                    setAgentForm({
                      ...agentForm,
                      mobile:
                        e.target.value,
                    })
                  }
                />
              </Grid>

              <Grid size={12}>
                <TextField
                  fullWidth
                  multiline
                  label="Address"
                  value={agentForm.address}
                  onChange={(e) =>
                    setAgentForm({
                      ...agentForm,
                      address:
                        e.target.value,
                    })
                  }
                />
              </Grid>
            </Grid>
          </DialogContent>

          <DialogActions>
            <Button
              onClick={() =>
                setAgentDialog(false)
              }
            >
              Cancel
            </Button>

            <Button
              type="submit"
              variant="contained"
              disabled={
                saving ||
                !agentForm.pop_ids.length
              }
            >
              Save
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      {/* POP MIGRATION */}
      <Dialog
        open={Boolean(
          popMigrationSource,
        )}
        onClose={() => {
          if (!saving) {
            setPOPMigrationSource(null)
            setPOPMigrationTargetID(0)
          }
        }}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>
          Migrate POP
        </DialogTitle>

        <DialogContent>
          <Alert
            severity="warning"
            sx={{ mb: 2 }}
          >
            Customers, primary Agent POP
            assignments, Agent POP links and
            Network Router assignments will
            move to the target POP. The source
            POP will be retired after a
            successful migration.
          </Alert>

          <Typography sx={{ mb: 2 }}>
            Source:{' '}
            <b>
              {popMigrationSource?.code} —{' '}
              {popMigrationSource?.name}
            </b>
          </Typography>

          <TextField
            fullWidth
            select
            label="Target POP"
            value={
              popMigrationTargetID || ''
            }
            onChange={(e) =>
              setPOPMigrationTargetID(
                Number(e.target.value),
              )
            }
          >
            {pops
              .filter(
                (pop) =>
                  pop.id !==
                    popMigrationSource?.id &&
                  pop.status ===
                    'ACTIVE',
              )
              .map((pop) => (
                <MenuItem
                  key={pop.id}
                  value={pop.id}
                >
                  {pop.code} — {pop.name}
                </MenuItem>
              ))}
          </TextField>
        </DialogContent>

        <DialogActions>
          <Button
            onClick={() => {
              setPOPMigrationSource(null)
              setPOPMigrationTargetID(0)
            }}
            disabled={saving}
          >
            Cancel
          </Button>

          <Button
            color="warning"
            variant="contained"
            onClick={() =>
              void migrateSelectedPOP()
            }
            disabled={
              saving ||
              !popMigrationTargetID
            }
          >
            {saving
              ? 'Migrating...'
              : 'Confirm POP Migration'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* POP DELETE */}
      <Dialog
        open={Boolean(popDeleteTarget)}
        onClose={() => {
          if (!saving) {
            setPOPDeleteTarget(null)
          }
        }}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>
          Archive POP
        </DialogTitle>

        <DialogContent>
          <Typography>
            Archive{' '}
            <b>
              {popDeleteTarget?.code} —{' '}
              {popDeleteTarget?.name}
            </b>
            ?
          </Typography>

          <Alert
            severity="info"
            sx={{ mt: 2 }}
          >
            This POP will move to Archived and
            can be restored later as INACTIVE.
            Only an empty POP can be archived.
            If it still has Customers, Agents,
            Agent POP links or Network Routers,
            use POP Migration first.
          </Alert>
        </DialogContent>

        <DialogActions>
          <Button
            onClick={() =>
              setPOPDeleteTarget(null)
            }
            disabled={saving}
          >
            Cancel
          </Button>

          <Button
            color="error"
            variant="contained"
            onClick={() =>
              void removePOP()
            }
            disabled={saving}
          >
            {saving
              ? 'Archiving...'
              : 'Archive'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* AGENT MIGRATION */}
      <Dialog
        open={Boolean(migrationSource)}
        onClose={() =>
          !saving &&
          setMigrationSource(null)
        }
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>
          Migrate Agent
        </DialogTitle>

        <DialogContent>
          <Alert
            severity="warning"
            sx={{ mb: 2 }}
          >
            Customers, login ownership and POP
            links will move to the target Agent.
            Historical collections and
            settlements will remain under the
            source Agent for audit.
          </Alert>

          <Typography sx={{ mb: 2 }}>
            Source:{' '}
            <b>
              {migrationSource?.code} —{' '}
              {migrationSource?.name}
            </b>
          </Typography>

          <TextField
            fullWidth
            select
            label="Target Agent"
            value={
              migrationTargetID || ''
            }
            onChange={(e) =>
              setMigrationTargetID(
                Number(e.target.value),
              )
            }
          >
            {agents
              .filter(
                (agent) =>
                  agent.id !==
                    migrationSource?.id &&
                  agent.status ===
                    'ACTIVE',
              )
              .map((agent) => (
                <MenuItem
                  key={agent.id}
                  value={agent.id}
                >
                  {agent.code} —{' '}
                  {agent.name}
                </MenuItem>
              ))}
          </TextField>
        </DialogContent>

        <DialogActions>
          <Button
            onClick={() =>
              setMigrationSource(null)
            }
            disabled={saving}
          >
            Cancel
          </Button>

          <Button
            color="warning"
            variant="contained"
            onClick={() =>
              void migrate()
            }
            disabled={
              saving ||
              !migrationTargetID
            }
          >
            {saving
              ? 'Migrating...'
              : 'Confirm Migration'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* AGENT DELETE */}
      <Dialog
        open={Boolean(deleteTarget)}
        onClose={() =>
          !saving &&
          setDeleteTarget(null)
        }
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>
          Archive Agent
        </DialogTitle>

        <DialogContent>
          <Typography>
            Archive{' '}
            <b>
              {deleteTarget?.code} —{' '}
              {deleteTarget?.name}
            </b>
            ?
          </Typography>

          <Alert
            severity="info"
            sx={{ mt: 2 }}
          >
            This Agent will move to Archived and
            can be restored later as INACTIVE.
            Only an Agent with no linked
            customers, login users or financial
            history can be archived. Otherwise
            use Migration.
          </Alert>
        </DialogContent>

        <DialogActions>
          <Button
            onClick={() =>
              setDeleteTarget(null)
            }
            disabled={saving}
          >
            Cancel
          </Button>

          <Button
            color="error"
            variant="contained"
            onClick={() =>
              void removeAgent()
            }
            disabled={saving}
          >
            {saving
              ? 'Archiving...'
              : 'Archive'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
