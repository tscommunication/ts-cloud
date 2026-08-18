import { useState, type ReactNode } from 'react'
import {
  AppBar,
  Box,
  Collapse,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
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
import RouterIcon from '@mui/icons-material/Router'
import WifiTetheringIcon from '@mui/icons-material/WifiTethering'
import UploadFileIcon from '@mui/icons-material/UploadFile'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'

import {
  Outlet,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import {
  getStoredUser,
  logout,
} from '../../api/auth'

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
    label: 'Customers',
    path: '/customers',
    icon: <PeopleIcon />,
    roles: ['superadmin', 'admin', 'agent'],
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
    ],
  },
  {
    label: 'MikroTik Routers',
    path: '/network/routers',
    icon: <RouterIcon />,
    roles: ['superadmin', 'admin'],
  },
  {
    label: 'Live PPPoE Users',
    path: '/network/pppoe-sessions',
    icon: <WifiTetheringIcon />,
    roles: ['superadmin', 'admin'],
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
  const navigate = useNavigate()
  const location = useLocation()

  const [mobileOpen, setMobileOpen] = useState(false)

  const [organizationOpen, setOrganizationOpen] =
    useState(
      location.pathname.startsWith('/organization'),
    )

  const storedUser = getStoredUser()
  const role = storedUser?.role

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

            return (
              <Box key={item.path}>
                <ListItemButton
                  onClick={() =>
                    setOrganizationOpen(
                      (open) => !open,
                    )
                  }
                  sx={{
                    color:
                      location.pathname.startsWith(
                        '/organization',
                      )
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

                  {organizationOpen ? (
                    <ExpandLessIcon />
                  ) : (
                    <ExpandMoreIcon />
                  )}
                </ListItemButton>

                <Collapse
                  in={organizationOpen}
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
                            location.pathname ===
                            child.path
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
