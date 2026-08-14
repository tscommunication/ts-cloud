import { useState } from 'react'
import type { FormEvent } from 'react'

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  TextField,
  Typography,
} from '@mui/material'
import LoginIcon from '@mui/icons-material/Login'
import { useNavigate } from 'react-router-dom'

import { login } from '../../api/auth'
import { getAPIErrorMessage } from '../../api/errors'

export default function Login() {
  const navigate = useNavigate()

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    setError('')
    setLoading(true)

    try {
      const response = await login({
        username,
        password,
      })

      localStorage.setItem('access_token', response.access_token)
      localStorage.setItem('user', JSON.stringify(response.user))

      navigate('/dashboard', { replace: true })
    } catch (error: unknown) {
      setError(
        getAPIErrorMessage(
          error,
          'Login failed. Please check your username and password.',
        ),
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#f5f7fb',
        p: 2,
      }}
    >
      <Card
        elevation={4}
        sx={{
          width: '100%',
          maxWidth: 420,
          borderRadius: 3,
        }}
      >
        <CardContent sx={{ p: 4 }}>
          <Box sx={{ textAlign: 'center', mb: 4 }}>
            <Typography
              variant="h4"
              component="h1"
              sx={{ fontWeight: 700 }}
              color="primary"
              gutterBottom
            >
              TS-Cloud
            </Typography>

            <Typography variant="body1" color="text.secondary">
              Admin Panel Login
            </Typography>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 3 }}>
              {error}
            </Alert>
          )}

          <Box component="form" onSubmit={handleSubmit}>
            <TextField
              fullWidth
              label="Username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              margin="normal"
              autoComplete="username"
              autoFocus
              required
            />

            <TextField
              fullWidth
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              margin="normal"
              autoComplete="current-password"
              required
            />

            <Button
              fullWidth
              type="submit"
              variant="contained"
              size="large"
              startIcon={
                loading ? (
                  <CircularProgress size={20} color="inherit" />
                ) : (
                  <LoginIcon />
                )
              }
              disabled={loading}
              sx={{
                mt: 3,
                py: 1.4,
                borderRadius: 2,
              }}
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
