import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import axios from 'axios'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  Grid,
  TextField,
  Typography,
} from '@mui/material'
import LogoutIcon from '@mui/icons-material/Logout'
import SaveIcon from '@mui/icons-material/Save'
import { useNavigate } from 'react-router-dom'

import { logout } from '../../api/auth'
import {
  getCurrentUser,
  getUser,
  updateUser,
  type User,
} from '../../api/users'

interface SettingsForm {
  name: string
  username: string
  email: string
  password: string
  confirmPassword: string
}

const emptyForm: SettingsForm = {
  name: '',
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
}

const errorMessage = (error: unknown, fallback: string) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error || fallback
  }
  return fallback
}

function Settings() {
  const navigate = useNavigate()
  const [user, setUser] = useState<User | null>(null)
  const [form, setForm] = useState<SettingsForm>(emptyForm)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const loadProfile = async () => {
    try {
      setLoading(true)
      setError('')
      const current = await getCurrentUser()
      const profile = await getUser(current.id)
      setUser(profile)
      setForm({
        name: profile.name,
        username: profile.username,
        email: profile.email,
        password: '',
        confirmPassword: '',
      })
    } catch (err: unknown) {
      setError(errorMessage(err, 'Failed to load account settings.'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadProfile()
  }, [])

  const change = (key: keyof SettingsForm, value: string) => {
    setForm((current) => ({ ...current, [key]: value }))
    setSuccess('')
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!user) return
    if (form.password && form.password.length < 6) {
      setError('New password must be at least 6 characters.')
      return
    }
    if (form.password !== form.confirmPassword) {
      setError('New password and confirmation do not match.')
      return
    }

    try {
      setSaving(true)
      setError('')
      setSuccess('')
      const updated = await updateUser(user.id, {
        name: form.name.trim(),
        username: form.username.trim(),
        email: form.email.trim(),
        ...(form.password ? { password: form.password } : {}),
      })
      setUser(updated)
      setForm((current) => ({
        ...current,
        name: updated.name,
        username: updated.username,
        email: updated.email,
        password: '',
        confirmPassword: '',
      }))
      localStorage.setItem(
        'user',
        JSON.stringify({
          id: updated.id,
          username: updated.username,
          role: updated.role,
        }),
      )
      setSuccess('Account settings updated successfully.')
    } catch (err: unknown) {
      setError(errorMessage(err, 'Failed to update account settings.'))
    } finally {
      setSaving(false)
    }
  }

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <Box sx={{ maxWidth: 900 }}>
      <Typography variant="h4" component="h1" sx={{ fontWeight: 700 }}>
        Settings
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Manage your administrator account and sign-in credentials.
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError('')}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert severity="success" sx={{ mb: 3 }} onClose={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      {loading ? (
        <Box sx={{ py: 8, textAlign: 'center' }}><CircularProgress /></Box>
      ) : (
        <Card>
          <CardContent sx={{ p: { xs: 2, sm: 4 } }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 2, mb: 3 }}>
              <Box>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>Account Profile</Typography>
                <Typography variant="body2" color="text.secondary">User ID #{user?.id}</Typography>
              </Box>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Chip label={user?.role || '-'} color="primary" variant="outlined" />
                <Chip label={user?.active ? 'ACTIVE' : 'DISABLED'} color={user?.active ? 'success' : 'default'} />
              </Box>
            </Box>

            <Box component="form" onSubmit={submit}>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12 }}>
                  <TextField fullWidth required label="Name" value={form.name} onChange={(event) => change('name', event.target.value)} />
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField fullWidth required label="Username" value={form.username} onChange={(event) => change('username', event.target.value)} autoComplete="username" />
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField fullWidth required type="email" label="Email" value={form.email} onChange={(event) => change('email', event.target.value)} autoComplete="email" />
                </Grid>
              </Grid>

              <Divider sx={{ my: 4 }} />
              <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>Change Password</Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>Leave both fields blank to keep your existing password.</Typography>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField fullWidth type="password" label="New Password" value={form.password} onChange={(event) => change('password', event.target.value)} autoComplete="new-password" helperText="Minimum 6 characters" />
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField fullWidth type="password" label="Confirm New Password" value={form.confirmPassword} onChange={(event) => change('confirmPassword', event.target.value)} autoComplete="new-password" />
                </Grid>
              </Grid>

              <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, justifyContent: 'space-between', gap: 2, mt: 4 }}>
                <Button color="error" variant="outlined" startIcon={<LogoutIcon />} onClick={handleLogout} disabled={saving}>Sign Out</Button>
                <Button type="submit" variant="contained" startIcon={saving ? <CircularProgress size={18} /> : <SaveIcon />} disabled={saving || !form.name.trim() || !form.username.trim() || !form.email.trim()}>{saving ? 'Saving...' : 'Save Changes'}</Button>
              </Box>
            </Box>
          </CardContent>
        </Card>
      )}
    </Box>
  )
}

export default Settings
