import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { AppBar, Box, Button, Divider, Drawer, IconButton, List, ListItemButton, ListItemIcon, ListItemText, Stack, Toolbar, Typography } from '@mui/material'
import MenuIcon from '@mui/icons-material/Menu'
import DashboardIcon from '@mui/icons-material/Dashboard'
import WifiIcon from '@mui/icons-material/Wifi'
import InsightsIcon from '@mui/icons-material/Insights'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import ReceiptLongIcon from '@mui/icons-material/ReceiptLong'
import PaymentsIcon from '@mui/icons-material/Payments'
import PersonIcon from '@mui/icons-material/Person'
import AppsIcon from '@mui/icons-material/Apps'
import LogoutIcon from '@mui/icons-material/Logout'

import { getStoredUser, logout } from '../../api/auth'

const drawerWidth = 250
const navigation = [
  { label: 'Dashboard', path: '/selfcare', icon: <DashboardIcon /> },
  { label: 'Live Traffic', path: '/selfcare/live-traffic', icon: <InsightsIcon /> },
  { label: 'My Connection', path: '/selfcare/connection', icon: <WifiIcon /> },
  { label: 'My Package', path: '/selfcare/packages', icon: <Inventory2Icon /> },
  { label: 'Billing & Invoices', path: '/selfcare/billing', icon: <ReceiptLongIcon /> },
  { label: 'Payment History', path: '/selfcare/payments', icon: <PaymentsIcon /> },
  { label: 'My Profile', path: '/selfcare/profile', icon: <PersonIcon /> },
  { label: 'My Services', path: '/selfcare/services', icon: <AppsIcon /> },
]

export default function SelfCareLayout() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const navigate = useNavigate()
  const user = getStoredUser()
  const sidebar = <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}><Toolbar><Typography variant="h6" sx={{ color: 'primary.main', fontWeight: 800 }}>TS-Cloud</Typography></Toolbar><Divider /><List sx={{ px: 1, py: 2 }}>{navigation.map((item) => <ListItemButton key={item.path} component={NavLink} to={item.path} end={item.path === '/selfcare'} onClick={() => setMobileOpen(false)} sx={{ borderRadius: 2, mb: 0.5, '&.active': { bgcolor: 'primary.light', color: 'primary.dark', '& .MuiListItemIcon-root': { color: 'primary.dark' } } }}><ListItemIcon>{item.icon}</ListItemIcon><ListItemText primary={item.label} /></ListItemButton>)}</List><Box sx={{ mt: 'auto', p: 2 }}><Button fullWidth color="inherit" startIcon={<LogoutIcon />} onClick={() => { logout(); navigate('/selfcare/login', { replace: true }) }}>Sign out</Button></Box></Box>
  return <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', display: 'flex' }}><AppBar position="fixed" color="inherit" elevation={1} sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}><Toolbar><IconButton edge="start" sx={{ display: { md: 'none' }, mr: 1 }} onClick={() => setMobileOpen(true)} aria-label="Open customer portal menu"><MenuIcon /></IconButton><Typography variant="h6" sx={{ fontWeight: 800, flexGrow: 1 }}>Customer Portal</Typography><Stack spacing={0} sx={{ alignItems: 'flex-end' }}><Typography sx={{ fontWeight: 700 }}>{user?.username || 'Customer'}</Typography><Typography variant="caption" color="text.secondary">My account</Typography></Stack></Toolbar></AppBar><Box component="nav" sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}><Drawer variant="temporary" open={mobileOpen} onClose={() => setMobileOpen(false)} ModalProps={{ keepMounted: true }} sx={{ display: { xs: 'block', md: 'none' }, '& .MuiDrawer-paper': { width: drawerWidth } }}>{sidebar}</Drawer><Drawer variant="permanent" sx={{ display: { xs: 'none', md: 'block' }, '& .MuiDrawer-paper': { width: drawerWidth, boxSizing: 'border-box' } }} open>{sidebar}</Drawer></Box><Box component="main" sx={{ flexGrow: 1, minWidth: 0, pt: 8, p: { xs: 2, md: 3 } }}><Outlet /></Box></Box>
}
