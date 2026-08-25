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
import DeleteIcon from '@mui/icons-material/Delete'
import EditIcon from '@mui/icons-material/Edit'
import RefreshIcon from '@mui/icons-material/Refresh'
import SearchIcon from '@mui/icons-material/Search'

import {
  createPackage,
  deletePackage,
  getPackages,
  updatePackage,
  type CreatePackageRequest,
  type Package,
} from '../../api/packages'
import { getAPIErrorMessage } from '../../api/errors'

const initialForm: CreatePackageRequest = {
  name: '',
  price: 0,
  download_speed: 0,
  upload_speed: 0,
  burst_download: 0,
  burst_upload: 0,
  validity_days: 30,
  mikrotik_profile: '',
  radius_profile: '',
  description: '',
}

function Packages() {
  const [packages, setPackages] = useState<Package[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [editingPackage, setEditingPackage] =
    useState<Package | null>(null)
  const [form, setForm] =
    useState<CreatePackageRequest>(initialForm)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deletingPackage, setDeletingPackage] =
    useState<Package | null>(null)

  const loadPackages = async () => {
    try {
      setLoading(true)
      setError('')

      const data = await getPackages()
      setPackages(data.packages)
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to load packages.'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // Initial API synchronization for this route.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadPackages()
  }, [])

  const filteredPackages = useMemo(() => {
    const query = search.trim().toLowerCase()

    if (!query) {
      return packages
    }

    return packages.filter((pkg) =>
      [
        pkg.name,
        pkg.mikrotik_profile,
        pkg.radius_profile,
        pkg.status,
        pkg.description,
      ]
        .join(' ')
        .toLowerCase()
        .includes(query),
    )
  }, [packages, search])

  const handleChange = (
    field: keyof CreatePackageRequest,
    value: string | number,
  ) => {
    setForm((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleOpenCreate = () => {
    setEditingPackage(null)
    setForm(initialForm)
    setError('')
    setOpen(true)
  }

  const handleOpenEdit = (pkg: Package) => {
    setEditingPackage(pkg)
    setError('')

    setForm({
      name: pkg.name,
      price: pkg.price,
      download_speed: pkg.download_speed,
      upload_speed: pkg.upload_speed,
      burst_download: pkg.burst_download,
      burst_upload: pkg.burst_upload,
      validity_days: pkg.validity_days,
      mikrotik_profile: pkg.mikrotik_profile,
      radius_profile: pkg.radius_profile,
      description: pkg.description,
    })

    setOpen(true)
  }

  const handleCloseDialog = () => {
    if (saving) {
      return
    }

    setOpen(false)
    setEditingPackage(null)
    setForm(initialForm)
  }

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault()

    if (!form.name.trim()) {
      return
    }

    try {
      setSaving(true)
      setError('')

      const payload: CreatePackageRequest = {
        ...form,
        name: form.name.trim(),
        mikrotik_profile: form.mikrotik_profile.trim(),
        radius_profile: form.radius_profile.trim(),
        description: form.description.trim(),
      }

      if (editingPackage) {
        await updatePackage(editingPackage.id, payload)
      } else {
        await createPackage(payload)
      }

      setOpen(false)
      setEditingPackage(null)
      setForm(initialForm)

      await loadPackages()
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          `Failed to ${editingPackage ? 'update' : 'create'} package.`,
        ),
      )
    } finally {
      setSaving(false)
    }
  }

  const handleOpenDelete = (pkg: Package) => {
    setDeletingPackage(pkg)
    setDeleteOpen(true)
  }

  const handleCloseDelete = () => {
    if (deleting) {
      return
    }

    setDeleteOpen(false)
    setDeletingPackage(null)
  }

  const handleDelete = async () => {
    if (!deletingPackage) {
      return
    }

    try {
      setDeleting(true)
      setError('')

      await deletePackage(deletingPackage.id)

      setDeleteOpen(false)
      setDeletingPackage(null)

      await loadPackages()
    } catch (error: unknown) {
      setError(getAPIErrorMessage(error, 'Failed to delete package.'))
    } finally {
      setDeleting(false)
    }
  }

  const formatSpeed = (speed: number) => {
    return speed ? `${speed} Mbps` : '-'
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
            Packages ({packages.length})
          </Typography>

          <Typography
            variant="body1"
            color="text.secondary"
          >
            Manage ISP internet packages and service profiles synced from the
            approved Packages List catalog.
          </Typography>
        </Box>

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleOpenCreate}
        >
          Add Package
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
              placeholder="Search packages..."
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
              onClick={() => void loadPackages()}
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
          ) : filteredPackages.length === 0 ? (
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
                No packages found
              </Typography>

              <Typography
                variant="body2"
                color="text.secondary"
              >
                {search
                  ? 'Try a different search term.'
                  : 'Add your first package to get started.'}
              </Typography>
            </Box>
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>SL</TableCell>
                    <TableCell>Package</TableCell>
                    <TableCell>Price</TableCell>
                    <TableCell>Profile</TableCell>
                    <TableCell>Speed</TableCell>
                    <TableCell>Commission</TableCell>
                    <TableCell>Validity</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell align="right">
                      Actions
                    </TableCell>
                  </TableRow>
                </TableHead>

                <TableBody>
                  {filteredPackages.map((pkg, index) => (
                    <TableRow key={pkg.id} hover>
                      <TableCell>{index + 1}</TableCell>
                      <TableCell>
                        <Typography
                          sx={{ fontWeight: 600 }}
                        >
                          {pkg.name}
                        </Typography>

                        {pkg.description && (
                          <Typography
                            variant="body2"
                            color="text.secondary"
                            sx={{
                              maxWidth: 260,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {pkg.description}
                          </Typography>
                        )}
                      </TableCell>

                      <TableCell>
                        BDT {pkg.price.toLocaleString()}
                      </TableCell>

                      <TableCell>
                        {pkg.mikrotik_profile || '-'}
                      </TableCell>

                      <TableCell>
                        <Typography variant="body2">
                          Download: {formatSpeed(pkg.download_speed)}
                        </Typography>

                        <Typography
                          variant="body2"
                          color="text.secondary"
                        >
                          Upload: {formatSpeed(pkg.upload_speed)}
                        </Typography>
                      </TableCell>

                      <TableCell>
                        BDT {pkg.commission.toLocaleString()}
                      </TableCell>

                      <TableCell>
                        {pkg.validity_days} days
                      </TableCell>

                      <TableCell>
                        <Typography
                          component="span"
                          sx={{
                            fontWeight: 600,
                            color:
                              pkg.status === 'ACTIVE'
                                ? 'success.main'
                                : 'text.secondary',
                          }}
                        >
                          {pkg.status}
                        </Typography>
                      </TableCell>

                      <TableCell align="right">
                        <IconButton
                          color="primary"
                          title="Edit"
                          onClick={() => handleOpenEdit(pkg)}
                        >
                          <EditIcon />
                        </IconButton>

                        <IconButton
                          color="error"
                          title="Delete"
                          onClick={() =>
                            handleOpenDelete(pkg)
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
            {editingPackage
              ? 'Edit Package'
              : 'Add Package'}
          </DialogTitle>

          <DialogContent dividers>
            <Grid
              container
              spacing={2}
              sx={{ pt: 1 }}
            >
              <Grid size={{ xs: 12, md: 8 }}>
                <TextField
                  fullWidth
                  required
                  label="Package Name"
                  value={form.name}
                  onChange={(event) =>
                    handleChange(
                      'name',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  label="Price"
                  value={form.price}
                  onChange={(event) =>
                    handleChange(
                      'price',
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
                  label="Download Speed (Mbps)"
                  value={form.download_speed}
                  onChange={(event) =>
                    handleChange(
                      'download_speed',
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
                  type="number"
                  label="Upload Speed (Mbps)"
                  value={form.upload_speed}
                  onChange={(event) =>
                    handleChange(
                      'upload_speed',
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
                  type="number"
                  label="Burst Download (Mbps)"
                  value={form.burst_download}
                  onChange={(event) =>
                    handleChange(
                      'burst_download',
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
                  type="number"
                  label="Burst Upload (Mbps)"
                  value={form.burst_upload}
                  onChange={(event) =>
                    handleChange(
                      'burst_upload',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: { min: 0 },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  required
                  type="number"
                  label="Validity Days"
                  value={form.validity_days}
                  onChange={(event) =>
                    handleChange(
                      'validity_days',
                      Number(event.target.value),
                    )
                  }
                  slotProps={{
                    htmlInput: { min: 1 },
                  }}
                />
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="MikroTik Profile"
                  value={form.mikrotik_profile}
                  onChange={(event) =>
                    handleChange(
                      'mikrotik_profile',
                      event.target.value,
                    )
                  }
                />
              </Grid>

              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="RADIUS Profile"
                  value={form.radius_profile}
                  onChange={(event) =>
                    handleChange(
                      'radius_profile',
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
                  label="Description"
                  value={form.description}
                  onChange={(event) =>
                    handleChange(
                      'description',
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
              disabled={saving || !form.name.trim()}
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
                : editingPackage
                  ? 'Update Package'
                  : 'Create Package'}
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
        <DialogTitle>Delete Package</DialogTitle>

        <DialogContent>
          <Typography>
            Delete <strong>{deletingPackage?.name}</strong>?
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

export default Packages
