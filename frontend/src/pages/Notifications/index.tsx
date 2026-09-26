import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Box, Button, Card, CardContent, Chip, Divider, List, ListItem, ListItemText, Typography } from '@mui/material'

import { getNotificationHistory, type AppNotification } from '../../api/notifications'

function notificationTime(value: string) {
  const timestamp = new Date(value)
  if (Number.isNaN(timestamp.getTime())) return 'Time unavailable'
  return timestamp.toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

export default function Notifications() {
  const navigate = useNavigate()
  const [items, setItems] = useState<AppNotification[]>([])
  const [total, setTotal] = useState(0)

  useEffect(() => {
    void getNotificationHistory().then((data) => {
      setItems(data.notifications)
      setTotal(data.total_count)
    })
  }, [])

  return <Box>
    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, mb: 0.5 }}>
      <Typography variant="h5" sx={{ fontWeight: 700 }}>All notifications</Typography>
      <Button size="small" onClick={() => navigate('/dashboard')}>Close</Button>
    </Box>
    <Typography color="text.secondary" sx={{ mb: 2 }}>Last 24 hours · {total} total · cleared items remain here</Typography>
    <Card><CardContent sx={{ p: 0 }}>
      {items.length === 0 ? <Typography sx={{ p: 2 }} color="text.secondary">No notifications in the last 24 hours.</Typography> :
        <List disablePadding>{items.map((item, index) => <Box key={item.id}>
          {index > 0 && <Divider />}
          <ListItem alignItems="flex-start" sx={{ py: 1.5 }}>
            <ListItemText
              primary={<Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
                <Typography variant="body2" sx={{ fontWeight: 700, color: item.severity === 'CRITICAL' ? 'error.main' : 'text.primary' }}>{item.title}</Typography>
                {item.read && <Chip size="small" label="Cleared" variant="outlined" />}
              </Box>}
              secondary={<><Typography component="span" variant="body2" color="text.secondary">{item.message}</Typography><Typography component="span" variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>{notificationTime(item.created_at)}</Typography></>}
            />
          </ListItem>
        </Box>)}</List>}
    </CardContent></Card>
  </Box>
}
