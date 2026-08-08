import { useQuery } from '@tanstack/react-query'
import {
  Box,
  Card,
  CardContent,
  Grid,
  Typography,
} from '@mui/material'

import PeopleIcon from '@mui/icons-material/People'
import CloudIcon from '@mui/icons-material/Cloud'
import UploadIcon from '@mui/icons-material/Upload'
import DownloadIcon from '@mui/icons-material/Download'
import LoginIcon from '@mui/icons-material/Login'

import { getFTPDashboard } from '../../api/ftpDashboard'

function formatBytes(bytes: number) {
  if (bytes === 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.floor(Math.log(bytes) / Math.log(1024))

  return `${(bytes / Math.pow(1024, index)).toFixed(2)} ${units[index]}`
}

function Dashboard() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['ftp-dashboard'],
    queryFn: getFTPDashboard,
    refetchInterval: 5000,
  })

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

export default Dashboard
