import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Box,
  Card,
  CardContent,
  Grid,
  Typography,
  Button,
  Alert,
} from '@mui/material'

import PeopleIcon from '@mui/icons-material/People'
import CloudIcon from '@mui/icons-material/Cloud'
import UploadIcon from '@mui/icons-material/Upload'
import DownloadIcon from '@mui/icons-material/Download'
import LoginIcon from '@mui/icons-material/Login'

import { getFTPDashboard } from '../../api/ftpDashboard'
import { getBillingRuns, getBillingSummary, runBilling } from '../../api/billing'
import { getStoredUser } from '../../api/auth'
import AgentDashboard from './AgentDashboard'

function formatBytes(bytes: number) {
  if (bytes === 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.floor(Math.log(bytes) / Math.log(1024))

  return `${(bytes / Math.pow(1024, index)).toFixed(2)} ${units[index]}`
}

function AdminDashboard() {
  const queryClient = useQueryClient()
  const { data, isLoading, isError } = useQuery({
    queryKey: ['ftp-dashboard'],
    queryFn: getFTPDashboard,
    refetchInterval: 5000,
  })
  const billing = useQuery({ queryKey: ['billing-summary'], queryFn: getBillingSummary })
  const runs = useQuery({ queryKey: ['billing-runs'], queryFn: getBillingRuns })
  const billingRun = useMutation({
    mutationFn: runBilling,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['billing-summary'] }),
        queryClient.invalidateQueries({ queryKey: ['billing-runs'] }),
      ])
    },
  })
  const isSuperadmin = getStoredUser()?.role === 'superadmin'

  const stats = [
    {
      title: 'FTP Users',
      value: data?.total_users ?? 0,
      icon: <PeopleIcon />,
    },
    {
      title: 'Online Users',
      value: data?.online_users ?? 0,
      icon: <CloudIcon />,
    },
    {
      title: "Today's Logins",
      value: data?.today_logins ?? 0,
      icon: <LoginIcon />,
    },
    {
      title: "Today's Uploads",
      value: data?.today_uploads ?? 0,
      icon: <UploadIcon />,
    },
    {
      title: "Today's Downloads",
      value: data?.today_downloads ?? 0,
      icon: <DownloadIcon />,
    },
    {
      title: 'Upload Traffic',
      value: formatBytes(data?.today_upload_bytes ?? 0),
      icon: <UploadIcon />,
    },
    {
      title: 'Download Traffic',
      value: formatBytes(data?.today_download_bytes ?? 0),
      icon: <DownloadIcon />,
    },
  ]

  return (
    <Box>
      <Typography
        variant="h4"
        sx={{
          fontWeight: 700,
          mb: 1,
        }}
      >
        Dashboard
      </Typography>

      {billingRun.isError && <Alert severity="error" sx={{ mb: 2 }}>Billing run failed.</Alert>}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h5" sx={{ fontWeight: 700 }}>Billing Overview</Typography>
        {isSuperadmin && <Button variant="contained" disabled={billingRun.isPending} onClick={() => billingRun.mutate()}>{billingRun.isPending ? 'Running...' : 'Run Due Billing'}</Button>}
      </Box>
      <Grid container spacing={2} sx={{ mb: 4 }}>
        {[
          ['Total Invoiced', billing.data?.total_invoiced ?? 0],
          ['Total Collected', billing.data?.total_collected ?? 0],
          ['Outstanding', billing.data?.total_outstanding ?? 0],
          ["Today's Collection", billing.data?.today_collected ?? 0],
        ].map(([label, value]) => (
          <Grid key={String(label)} size={{ xs: 12, sm: 6, lg: 3 }}><Card><CardContent><Typography color="text.secondary">{label}</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>BDT {Number(value).toLocaleString()}</Typography></CardContent></Card></Grid>
        ))}
        <Grid size={{ xs: 12 }}><Typography color="text.secondary">Overdue invoices: {billing.data?.overdue_invoices ?? 0} · Open invoices: {billing.data?.unpaid_invoices ?? 0} · Last billing run: {runs.data?.[0] ? `${runs.data[0].status} (${runs.data[0].created_count} created, ${runs.data[0].failed_count} failed)` : 'Not run yet'}</Typography></Grid>
      </Grid>

      <Typography
        variant="body1"
        color="text.secondary"
        sx={{
          mb: 4,
        }}
      >
        TS-Cloud service overview
      </Typography>

      {isLoading && (
        <Typography color="text.secondary" sx={{ mb: 3 }}>
          Loading dashboard...
        </Typography>
      )}

      {isError && (
        <Typography color="error" sx={{ mb: 3 }}>
          Unable to load FTP dashboard data.
        </Typography>
      )}

      <Grid container spacing={3}>
        {stats.map((stat) => (
          <Grid
            key={stat.title}
            size={{
              xs: 12,
              sm: 6,
              md: 4,
              lg: 3,
            }}
          >
            <Card>
              <CardContent>
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    mb: 2,
                  }}
                >
                  <Typography color="text.secondary">
                    {stat.title}
                  </Typography>

                  <Box
                    sx={{
                      color: 'primary.main',
                      display: 'flex',
                    }}
                  >
                    {stat.icon}
                  </Box>
                </Box>

                <Typography
                  variant="h4"
                  sx={{
                    fontWeight: 700,
                  }}
                >
                  {stat.value}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  )
}

function Dashboard() {
  return getStoredUser()?.role === 'agent' ? <AgentDashboard /> : <AdminDashboard />
}

export default Dashboard
