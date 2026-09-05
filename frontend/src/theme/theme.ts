import { alpha, createTheme, type PaletteMode } from '@mui/material/styles'

export type ThemeName = 'classic' | 'ocean' | 'emerald' | 'royal' | 'sunset' | 'midnight'
type ThemePreset = { label: string; mode: PaletteMode; primary: string; secondary: string; background: string; paper: string; text: string; secondaryText: string; divider: string }

export const themePresets: Record<ThemeName, ThemePreset> = {
  classic: { label: 'TS Cloud Classic', mode: 'light', primary: '#1565C0', secondary: '#00897B', background: '#F5F7FA', paper: '#FFFFFF', text: '#172033', secondaryText: '#667085', divider: '#E4E7EC' },
  ocean: { label: 'TS Cloud Ocean', mode: 'light', primary: '#0077B6', secondary: '#00B4D8', background: '#F1FAFE', paper: '#FFFFFF', text: '#102A43', secondaryText: '#526B84', divider: '#D6EAF3' },
  emerald: { label: 'TS Cloud Emerald', mode: 'light', primary: '#087F5B', secondary: '#16A34A', background: '#F3FBF7', paper: '#FFFFFF', text: '#153A2D', secondaryText: '#587066', divider: '#D8EBE1' },
  royal: { label: 'TS Cloud Royal', mode: 'light', primary: '#5B3CC4', secondary: '#7C3AED', background: '#F7F5FF', paper: '#FFFFFF', text: '#281C4C', secondaryText: '#675D82', divider: '#E4DFFD' },
  sunset: { label: 'TS Cloud Sunset', mode: 'light', primary: '#D97706', secondary: '#EA580C', background: '#FFF8F0', paper: '#FFFFFF', text: '#492A0A', secondaryText: '#7C644B', divider: '#F4DFC7' },
  midnight: { label: 'TS Cloud Midnight', mode: 'dark', primary: '#4DA3FF', secondary: '#2DD4BF', background: '#071525', paper: '#0E2238', text: '#EDF6FF', secondaryText: '#A5BDD4', divider: '#27425E' },
}
export const themeNames = Object.keys(themePresets) as ThemeName[]
export const isThemeName = (value: string | null): value is ThemeName => value !== null && value in themePresets

export function createAppTheme(name: ThemeName) {
  const selected = themePresets[name]
  const dark = selected.mode === 'dark'
  return createTheme({
    palette: {
      mode: selected.mode,
      primary: { main: selected.primary }, secondary: { main: selected.secondary },
      background: { default: selected.background, paper: selected.paper },
      text: { primary: selected.text, secondary: selected.secondaryText },
      divider: selected.divider,
    },
    typography: {
      fontFamily: ['Inter', 'Roboto', 'Arial', 'sans-serif'].join(','),
      h4: { fontWeight: 700 }, h5: { fontWeight: 700 }, h6: { fontWeight: 600 },
      button: { textTransform: 'none', fontWeight: 600 },
    },
    shape: { borderRadius: 10 },
    components: {
      MuiCard: { styleOverrides: { root: { border: `1px solid ${selected.divider}`, boxShadow: dark ? '0 1px 3px rgba(0,0,0,.42)' : '0 1px 3px rgba(16,24,40,.08)' } } },
      MuiAppBar: { styleOverrides: { root: { backgroundColor: selected.paper, color: selected.text, boxShadow: dark ? '0 1px 3px rgba(0,0,0,.42)' : '0 1px 3px rgba(16,24,40,.08)' } } },
      MuiDrawer: { styleOverrides: { paper: { backgroundColor: selected.paper, borderRight: `1px solid ${selected.divider}` } } },
      MuiListItemButton: { styleOverrides: { root: ({ theme }) => ({ margin: '4px 10px', borderRadius: 8, '&.Mui-selected': { backgroundColor: alpha(theme.palette.primary.main, dark ? .24 : .12), color: theme.palette.primary.main, '& .MuiListItemIcon-root': { color: theme.palette.primary.main } }, '&.Mui-selected:hover': { backgroundColor: alpha(theme.palette.primary.main, dark ? .32 : .2) } }) } },
      MuiButton: { styleOverrides: { root: { borderRadius: 8, minHeight: 40 } } },
      MuiTextField: { defaultProps: { size: 'small' } },
    },
  })
}

export default createAppTheme('classic')
