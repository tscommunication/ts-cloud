import { alpha, createTheme, type PaletteMode } from '@mui/material/styles'

export type ThemeColor = 'blue' | 'indigo' | 'green' | 'orange'
export const themeColors: Record<ThemeColor, { label: string; primary: string; secondary: string }> = {
  blue: { label: 'TS Blue', primary: '#1565C0', secondary: '#00897B' },
  indigo: { label: 'Indigo', primary: '#4F46E5', secondary: '#7C3AED' },
  green: { label: 'Emerald', primary: '#087F5B', secondary: '#1971C2' },
  orange: { label: 'Amber', primary: '#D97706', secondary: '#C2410C' },
}

export function createAppTheme(mode: PaletteMode, color: ThemeColor) {
  const selected = themeColors[color]
  const dark = mode === 'dark'
  return createTheme({
    palette: {
      mode,
      primary: { main: selected.primary }, secondary: { main: selected.secondary },
      background: dark ? { default: '#101828', paper: '#1D2939' } : { default: '#F5F7FA', paper: '#FFFFFF' },
      text: dark ? { primary: '#F2F4F7', secondary: '#98A2B3' } : { primary: '#172033', secondary: '#667085' },
      divider: dark ? '#344054' : '#E4E7EC',
    },
    typography: {
      fontFamily: ['Inter', 'Roboto', 'Arial', 'sans-serif'].join(','),
      h4: { fontWeight: 700 }, h5: { fontWeight: 700 }, h6: { fontWeight: 600 },
      button: { textTransform: 'none', fontWeight: 600 },
    },
    shape: { borderRadius: 10 },
    components: {
      MuiCard: { styleOverrides: { root: { border: `1px solid ${dark ? '#344054' : '#E4E7EC'}`, boxShadow: dark ? '0 1px 3px rgba(0,0,0,.35)' : '0 1px 3px rgba(16,24,40,.08)' } } },
      MuiAppBar: { styleOverrides: { root: { backgroundColor: dark ? '#1D2939' : '#FFFFFF', color: dark ? '#F2F4F7' : '#172033', boxShadow: dark ? '0 1px 3px rgba(0,0,0,.35)' : '0 1px 3px rgba(16,24,40,.08)' } } },
      MuiDrawer: { styleOverrides: { paper: { backgroundColor: dark ? '#1D2939' : '#FFFFFF', borderRight: `1px solid ${dark ? '#344054' : '#E4E7EC'}` } } },
      MuiListItemButton: { styleOverrides: { root: ({ theme }) => ({ margin: '4px 10px', borderRadius: 8, '&.Mui-selected': { backgroundColor: alpha(theme.palette.primary.main, dark ? .24 : .12), color: theme.palette.primary.main, '& .MuiListItemIcon-root': { color: theme.palette.primary.main } }, '&.Mui-selected:hover': { backgroundColor: alpha(theme.palette.primary.main, dark ? .32 : .2) } }) } },
      MuiButton: { styleOverrides: { root: { borderRadius: 8, minHeight: 40 } } },
      MuiTextField: { defaultProps: { size: 'small' } },
    },
  })
}

export default createAppTheme('light', 'blue')
