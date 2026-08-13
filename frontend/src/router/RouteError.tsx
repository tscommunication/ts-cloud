import { Box, Button, Paper, Typography } from '@mui/material'
import { isRouteErrorResponse, useRouteError } from 'react-router-dom'

export default function RouteError() {
  const error = useRouteError()
  const isNotFound = isRouteErrorResponse(error) && error.status === 404

  return (
    <Box
      component="main"
      sx={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'center',
        bgcolor: 'background.default',
        p: 3,
      }}
    >
      <Paper sx={{ width: '100%', maxWidth: 520, p: 4, textAlign: 'center' }}>
        <Typography component="h1" variant="h4" gutterBottom>
          {isNotFound ? 'Page not found' : 'Something went wrong'}
        </Typography>
        <Typography color="text.secondary" sx={{ mb: 3 }}>
          {isNotFound
            ? 'The requested page does not exist.'
            : 'The page could not be loaded. Please refresh and try again.'}
        </Typography>
        <Button
          variant="contained"
          onClick={() => {
            if (isNotFound) window.location.assign('/')
            else window.location.reload()
          }}
        >
          {isNotFound ? 'Go to dashboard' : 'Reload page'}
        </Button>
      </Paper>
    </Box>
  )
}
