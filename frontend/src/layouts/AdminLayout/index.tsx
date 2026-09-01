import { useEffect, useState, type MouseEvent, type ReactNode } from 'react'
import {
  AppBar,
  Badge,
  Box,
  Collapse,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material'

import MenuIcon from '@mui/icons-material/Menu'
import DashboardIcon from '@mui/icons-material/Dashboard'
import PeopleIcon from '@mui/icons-material/People'
import HowToRegIcon from '@mui/icons-material/HowToReg'
import InventoryIcon from '@mui/icons-material/Inventory'
import SubscriptionsIcon from '@mui/icons-material/Subscriptions'
import ReceiptIcon from '@mui/icons-material/Receipt'
import PaymentsIcon from '@mui/icons-material/Payments'
import CloudIcon from '@mui/icons-material/Cloud'
import SettingsIcon from '@mui/icons-material/Settings'
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts'
import AccountTreeIcon from '@mui/icons-material/AccountTree'
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet'
import LogoutIcon from '@mui/icons-material/Logout'
import NotificationsIcon from '@mui/icons-material/Notifications'
import RouterIcon from '@mui/icons-material/Router'
import UploadFileIcon from '@mui/icons-material/UploadFile'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import PaletteIcon from '@mui/icons-material/Palette'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import LightModeIcon from '@mui/icons-material/LightMode'

import {
  Outlet,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import {
  getStoredUser,
  logout,
} from '../../api/auth'
import {
  getNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type AppNotification,
} from '../../api/notifications'
import { useThemeSettings } from '../../theme/useThemeSettings'
import { themeColors, type ThemeColor } from '../../theme/theme'

const drawerWidth = 250

interface SubMenuItem {
  label: string
  path: string
  roles: string[]
}

interface MenuItem {
  label: string
  path: string
  icon: ReactNode
  roles: string[]
  children?: SubMenuItem[]
}

const menuItems: MenuItem[] = [
  {
    label: 'Dashboard',
    path: '/dashboard',
    icon: <DashboardIcon />,
    roles: ['superadmin', 'admin', 'agent'],
  },
  {
    label: 'Connection Requests',
    path: '/customer-provision-requests',
    icon: <HowToRegIcon />,
    roles: ['superadmin', 'admin', 'agent'],
  },
  {
    label: 'Customer Change Requests',
    path: '/customer-change-requests',
    icon: <HowToRegIcon />,
    roles: ['superadmin', 'admin', 'agent'],
  },
  {
    label: 'Customers',
    path: '/customers',
    icon: <PeopleIcon />,
    roles: ['superadmin', 'admin', 'agent'],
    children: [
      {
        label: 'Customers List',
        path: '/customers?view=all',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Add Customer',
        path: '/customers?action=add',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Active Customers',
        path: '/customers?status=ACTIVE',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Bulk Date Extend',
        path: '/customers?action=bulk-extend',
        roles: ['superadmin'],
      },
      {
        label: 'Deactivated Customers',
        path: '/customers?status=INACTIVE',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Disabled Customers',
        path: '/customers?view=DISABLED',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Expired Customers',
        path: '/customers?view=EXPIRED',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Pending Customers',
        path: '/customers?view=PENDING',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Recent Customers',
        path: '/customers?view=RECENT',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Online Customers',
        path: '/customers?view=ONLINE',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Offline Customers',
        path: '/customers?view=OFFLINE',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Deleted Customers',
        path: '/customers?status=ARCHIVED',
        roles: ['superadmin', 'admin', 'agent'],
      },
      {
        label: 'Live PPPoE Users',
        path: '/network/pppoe-sessions',
        roles: ['superadmin', 'admin', 'agent'],
      },
    ],
  },
  {
    label: 'POP & Agents',
    path: '/organization',
    icon: <AccountTreeIcon />,
    roles: ['superadmin', 'admin'],
    children: [
      {
        label: 'POPs',
        path: '/organization/pops',
        roles: ['superadmin', 'admin'],
      },
      {
        label: 'Agents / Resellers',
        path: '/organization/agents',
        roles: ['superadmin', 'admin'],
      },
      {
        label: 'Assign Package Permission',
        path: '/organization/agent-package-permissions',
        roles: ['superadmin', 'admin'],
      },
      {
        label: 'Code & Serial Management',
        path: '/organization/code-management',
        roles: ['superadmin'],
      },
    ],
  },
  {
    label: 'Network Integration',
    path: '/network',
    icon: <RouterIcon />,
    roles: ['superadmin', 'admin', 'agent'],
    children: [
      { label: 'OLT & Switch Monitoring', path: '/network/devices', roles: ['superadmin', 'admin', 'agent'] },
      { label: 'MikroTik Routers', path: '/network/routers', roles: ['superadmin', 'admin'] },
    ],
  },
  {
    label: 'Agent Collections',
    path: '/agent-collections',
    icon: <AccountBalanceWalletIcon />,
    roles: ['superadmin', 'admin', 'agent'],
  },
  {
    label: 'Packages',
    path: '/packages',
    icon: <InventoryIcon />,
    roles: ['superadmin', 'admin'],
  },
  {
    label: 'Subscriptions',
    path: '/subscriptions',
    icon: <SubscriptionsIcon />,
    roles: ['superadmin', 'admin'],
  },
  {
    label: 'Invoices',
    path: '/invoices',
    icon: <ReceiptIcon />,
    roles: ['superadmin', 'admin'],
  },
  {
    label: 'Payments',
    path: '/payments',
    icon: <PaymentsIcon />,
    roles: ['superadmin', 'admin', 'agent'],
  },
  {
    label: 'FTP',
    path: '/ftp',
    icon: <CloudIcon />,
    roles: ['superadmin', 'admin'],
  },
  {
    label: 'Service Entitlements',
    path: '/service-entitlements',
    icon: <CloudIcon />,
    roles: ['superadmin', 'admin'],
  },
  {
    label: 'Data Import & Export',
    path: '/customers/import',
    icon: <UploadFileIcon />,
    roles: ['superadmin'],
  },
  {
    label: 'Settings',
    path: '/settings',
    icon: <SettingsIcon />,
    roles: ['superadmin', 'admin', 'agent', 'user'],
  },
  {
    label: 'Users',
    path: '/users',
    icon: <ManageAccountsIcon />,
    roles: ['superadmin', 'admin'],
  },
]

function AdminLayout() {
	const { mode, color, setMode, setColor } = useThemeSettings()
  const navigate = useNavigate()
  const location = useLocation()

  const [mobileOpen, setMobileOpen] = useState(false)
  const [notificationAnchor, setNotificationAnchor] = useState<HTMLElement | null>(null)
  const [themeAnchor, setThemeAnchor] = useState<HTMLElement | null>(null)
  const [notifications, setNotifications] = useState<AppNotification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)

  const [openMenus, setOpenMenus] = useState<
    Record<string, boolean>
  >({
    '/customers':
      location.pathname.startsWith('/customers') ||
      location.pathname.startsWith('/network/pppoe-sessions'),
    '/organization': location.pathname.startsWith('/organization'),
  })

  const storedUser = getStoredUser()
  const role = storedUser?.role

  useEffect(() => {
    if (role !== 'superadmin' && role !== 'admin') return

    let cancelled = false

    const loadNotifications = async () => {
      try {
        const data = await getNotifications()
        if (cancelled) return
        setNotifications(Array.isArray(data.notifications) ? data.notifications : [])
        setUnreadCount(Number.isFinite(data.unread_count) ? data.unread_count : 0)
      } catch {
        // Header polling must never interrupt normal navigation.
      }
    }

    const initialLoad = window.setTimeout(() => {
      void loadNotifications()
    }, 0)

    const timer = window.setInterval(() => {
      void loadNotifications()
    }, 10000)

    return () => {
      cancelled = true
      window.clearTimeout(initialLoad)
      window.clearInterval(timer)
    }
  }, [role])

  const openNotifications = (event: MouseEvent<HTMLElement>) => setNotificationAnchor(event.currentTarget)
  const selectNotification = async (item: AppNotification) => {
    setNotificationAnchor(null)
    if (!item.read) {
      await markNotificationRead(item.id)
      setNotifications((current) => current.map((row) => row.id === item.id ? { ...row, read: true } : row))
      setUnreadCount((count) => Math.max(0, count - 1))
    }
    navigate(item.target_path)
  }

  const readAllNotifications = async () => {
    await markAllNotificationsRead()
    setNotifications((current) => current.map((row) => ({ ...row, read: true })))
    setUnreadCount(0)
  }

  const visibleMenuItems = menuItems.filter(
    (item) =>
      role
        ? item.roles.includes(role)
        : false,
  )

  const handleDrawerToggle = () => {
    setMobileOpen((open) => !open)
  }

  const handleSignOut = () => {
    logout()
    navigate('/login', {
      replace: true,
    })
  }

  const drawer = (
    <Box>
      <Toolbar>
        <Typography
          variant="h6"
          sx={{ fontWeight: 700 }}
        >
          TS-Cloud
        </Typography>
      </Toolbar>

      <Divider />

      <List>
        {visibleMenuItems.map((item) => {
          if (item.children?.length) {
            const visibleChildren =
              item.children.filter(
                (child) =>
                  role
                    ? child.roles.includes(role)
                    : false,
              )
            const expanded = Boolean(openMenus[item.path])
            const currentLocation = `${location.pathname}${location.search}`
            const active = visibleChildren.some(
              (child) =>
                currentLocation === child.path ||
                (child.path === '/customers?view=all' &&
                  location.pathname === '/customers' &&
                  !location.search),
            )

            return (
              <Box key={item.path}>
                <ListItemButton
                  onClick={() =>
                    setOpenMenus((current) => ({
                      ...current,
                      [item.path]: !current[item.path],
                    }))
                  }
                  sx={{
                    color: active
                      ? 'primary.main'
                      : 'inherit',
                  }}
                >
                  <ListItemIcon>
                    {item.icon}
                  </ListItemIcon>

                  <ListItemText
                    primary={item.label}
                  />

                  {expanded ? (
                    <ExpandLessIcon />
                  ) : (
                    <ExpandMoreIcon />
                  )}
                </ListItemButton>

                <Collapse
                  in={expanded}
                  timeout="auto"
                  unmountOnExit
                >
                  <List
                    component="div"
                    disablePadding
                  >
                    {visibleChildren.map(
                      (child) => (
                        <ListItemButton
                          key={child.path}
                          selected={
                            currentLocation === child.path ||
                            (child.path === '/customers?view=all' &&
                              location.pathname === '/customers' &&
                              !location.search)
                          }
                          onClick={() => {
                            navigate(child.path)
                            setMobileOpen(false)
                          }}
                          sx={{ pl: 7 }}
                        >
                          <ListItemText
                            primary={
                              child.label
                            }
                          />
                        </ListItemButton>
                      ),
                    )}
                  </List>
                </Collapse>
              </Box>
            )
          }

          return (
            <ListItemButton
              key={item.path}
              selected={
                location.pathname ===
                item.path
              }
              onClick={() => {
                navigate(item.path)
                setMobileOpen(false)
              }}
            >
              <ListItemIcon>
                {item.icon}
              </ListItemIcon>

              <ListItemText
                primary={
                  role === 'agent' &&
                  item.path === '/payments'
                    ? 'Collections & Receipts'
                    : item.label
                }
              />
            </ListItemButton>
          )
        })}
      </List>
    </Box>
  )

  return (
    <Box
      sx={{
        display: 'flex',
        minHeight: '100vh',
        backgroundColor:
          'background.default',
      }}
    >
      <AppBar
        position="fixed"
        sx={{
          width: {
            sm: `calc(100% - ${drawerWidth}px)`,
          },
          ml: {
            sm: `${drawerWidth}px`,
          },
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            edge="start"
            onClick={handleDrawerToggle}
            sx={{
              mr: 2,
              display: {
                sm: 'none',
              },
            }}
          >
            <MenuIcon />
          </IconButton>

          <Typography
            variant="h6"
            noWrap
            sx={{ fontWeight: 600 }}
          >
            {role === 'agent'
              ? 'TS-Cloud Agent Portal'
              : 'TS-Cloud Admin Panel'}
          </Typography>

          <Box sx={{ flexGrow: 1 }} />

          <Tooltip title="Color theme">
            <IconButton color="inherit" aria-label="Choose color theme" onClick={(event) => setThemeAnchor(event.currentTarget)}>
              <PaletteIcon />
            </IconButton>
          </Tooltip>
          <Menu anchorEl={themeAnchor} open={Boolean(themeAnchor)} onClose={() => setThemeAnchor(null)}>
            <MenuItem onClick={() => setMode(mode === 'dark' ? 'light' : 'dark')}>
              <ListItemIcon>{mode === 'dark' ? <LightModeIcon /> : <DarkModeIcon />}</ListItemIcon>
              <ListItemText primary={mode === 'dark' ? 'Light mode' : 'Dark mode'} />
            </MenuItem>
            <Divider />
            {(Object.keys(themeColors) as ThemeColor[]).map((themeColor) => (
              <MenuItem key={themeColor} selected={color === themeColor} onClick={() => setColor(themeColor)}>
                <ListItemIcon>
                  <Box sx={{ width: 20, height: 20, borderRadius: '50%', bgcolor: themeColors[themeColor].primary, border: '2px solid', borderColor: color === themeColor ? 'text.primary' : 'transparent' }} />
                </ListItemIcon>
                <ListItemText primary={themeColors[themeColor].label} />
              </MenuItem>
            ))}
          </Menu>

          {(role === 'superadmin' || role === 'admin' || role === 'agent') && (
            <>
              <Tooltip title="Notifications">
                <IconButton color="inherit" aria-label={`${unreadCount} unread notifications`} onClick={openNotifications}>
                  <Badge badgeContent={unreadCount} color="error" max={99}>
                    <NotificationsIcon />
                  </Badge>
                </IconButton>
              </Tooltip>
              <Menu
                anchorEl={notificationAnchor}
                open={Boolean(notificationAnchor)}
                onClose={() => setNotificationAnchor(null)}
                slotProps={{ paper: { sx: { width: { xs: 330, sm: 410 }, maxHeight: 480 } } }}
              >
                <Box sx={{ px: 2, py: 1, display: 'flex', alignItems: 'center' }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>Notifications</Typography>
                  <Box sx={{ flexGrow: 1 }} />
                  {unreadCount > 0 && (
                    <Typography component="button" variant="caption" onClick={() => void readAllNotifications()}
                      sx={{ border: 0, background: 'none', color: 'primary.main', cursor: 'pointer' }}>
                      Mark all read
                    </Typography>
                  )}
                </Box>
                <Divider />
                {(Array.isArray(notifications) ? notifications : []).length === 0 ? (
                  <MenuItem disabled>No active notifications</MenuItem>
                ) : (Array.isArray(notifications) ? notifications : []).map((item) => (
                  <MenuItem key={item.id} onClick={() => void selectNotification(item)}
                    sx={{ alignItems: 'flex-start', whiteSpace: 'normal', py: 1.25, bgcolor: item.read ? 'transparent' : 'action.hover' }}>
                    <Box>
                      <Typography variant="body2" sx={{ fontWeight: item.read ? 500 : 700, color: item.severity === 'CRITICAL' ? 'error.main' : 'text.primary' }}>
                        {item.title}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">{item.message}</Typography>
                    </Box>
                  </MenuItem>
                ))}
              </Menu>
            </>
          )}

          <Typography
            variant="body2"
            sx={{
              mr: 1.5,
              display: {
                xs: 'none',
                sm: 'block',
              },
            }}
          >
            {storedUser?.username}
          </Typography>

          <Tooltip title="Sign out">
            <IconButton
              color="inherit"
              aria-label="Sign out"
              onClick={handleSignOut}
            >
              <LogoutIcon />
            </IconButton>
          </Tooltip>
        </Toolbar>
      </AppBar>

      <Box
        component="nav"
        sx={{
          width: {
            sm: drawerWidth,
          },
          flexShrink: {
            sm: 0,
          },
        }}
      >
        <Drawer
          variant="temporary"
          open={mobileOpen}
          onClose={handleDrawerToggle}
          ModalProps={{
            keepMounted: true,
          }}
          sx={{
            display: {
              xs: 'block',
              sm: 'none',
            },
            '& .MuiDrawer-paper': {
              width: drawerWidth,
              boxSizing: 'border-box',
            },
          }}
        >
          {drawer}
        </Drawer>

        <Drawer
          variant="permanent"
          open
          sx={{
            display: {
              xs: 'none',
              sm: 'block',
            },
            '& .MuiDrawer-paper': {
              width: drawerWidth,
              boxSizing: 'border-box',
            },
          }}
        >
          {drawer}
        </Drawer>
      </Box>

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          width: {
            sm: `calc(100% - ${drawerWidth}px)`,
          },
        }}
      >
        <Toolbar />
        <Outlet />
      </Box>
    </Box>
  )
}

export default AdminLayout
