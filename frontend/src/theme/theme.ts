import { createTheme } from '@mui/material/styles'

const theme = createTheme({
  palette: {
    mode: 'light',

    primary: {
      main: '#1565C0',
      light: '#42A5F5',
      dark: '#0D47A1',
      contrastText: '#FFFFFF',
    },

    secondary: {
      main: '#00897B',
      light: '#4DB6AC',
      dark: '#00695C',
      contrastText: '#FFFFFF',
    },

    background: {
      default: '#F5F7FA',
      paper: '#FFFFFF',
    },

    text: {
      primary: '#172033',
      secondary: '#667085',
    },

    divider: '#E4E7EC',
  },

  typography: {
    fontFamily: [
      'Inter',
      'Roboto',
      'Arial',
      'sans-serif',
    ].join(','),

    h4: {
      fontWeight: 700,
    },

    h5: {
      fontWeight: 700,
    },

    h6: {
      fontWeight: 600,
    },

    button: {
      textTransform: 'none',
      fontWeight: 600,
    },
  },

  shape: {
    borderRadius: 10,
  },

  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          border: '1px solid #E4E7EC',
          boxShadow: '0 1px 3px rgba(16, 24, 40, 0.08)',
        },
      },
    },

    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: '#FFFFFF',
          color: '#172033',
          boxShadow: '0 1px 3px rgba(16, 24, 40, 0.08)',
        },
      },
    },

    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: '#FFFFFF',
          borderRight: '1px solid #E4E7EC',
        },
      },
    },

    MuiListItemButton: {
      styleOverrides: {
        root: {
          margin: '4px 10px',
          borderRadius: 8,

          '&.Mui-selected': {
            backgroundColor: '#E3F2FD',
            color: '#1565C0',

            '& .MuiListItemIcon-root': {
              color: '#1565C0',
            },
          },

          '&.Mui-selected:hover': {
            backgroundColor: '#BBDEFB',
          },
        },
      },
    },

    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          minHeight: 40,
        },
      },
    },

    MuiTextField: {
      defaultProps: {
        size: 'small',
      },
    },
  },
})

export default theme
