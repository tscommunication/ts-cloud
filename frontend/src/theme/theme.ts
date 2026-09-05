import { alpha, createTheme, type PaletteMode } from '@mui/material/styles'

export type ThemeName = 'classic' | 'ocean' | 'emerald' | 'royal' | 'sunset' | 'midnight'
type ThemePreset = { label: string; mode: PaletteMode; primary: string; secondary: string; sidebar: string; background: string; paper: string; text: string; secondaryText: string; hover: string; divider: string }

export const themePresets: Record<ThemeName, ThemePreset> = {
  classic: { label: 'TS Cloud Classic', mode: 'light', primary: '#2563EB', secondary: '#3B82F6', sidebar: '#0F172A', background: '#F8FAFC', paper: '#FFFFFF', text: '#0F172A', secondaryText: '#475569', hover: '#1D4ED8', divider: '#E2E8F0' },
  ocean: { label: 'TS Cloud Ocean', mode: 'light', primary: '#0891B2', secondary: '#06B6D4', sidebar: '#083344', background: '#ECFEFF', paper: '#FFFFFF', text: '#164E63', secondaryText: '#4B7280', hover: '#0E7490', divider: '#CFFAFE' },
  emerald: { label: 'TS Cloud Emerald', mode: 'light', primary: '#059669', secondary: '#10B981', sidebar: '#064E3B', background: '#ECFDF5', paper: '#FFFFFF', text: '#064E3B', secondaryText: '#4B6E62', hover: '#047857', divider: '#D1FAE5' },
  royal: { label: 'TS Cloud Royal', mode: 'light', primary: '#7C3AED', secondary: '#6366F1', sidebar: '#2E1065', background: '#F5F3FF', paper: '#FFFFFF', text: '#2E1065', secondaryText: '#665487', hover: '#6D28D9', divider: '#E9E3FF' },
  sunset: { label: 'TS Cloud Sunset', mode: 'light', primary: '#EA580C', secondary: '#F59E0B', sidebar: '#431407', background: '#FFF7ED', paper: '#FFFFFF', text: '#431407', secondaryText: '#7C5B4B', hover: '#C2410C', divider: '#FFEDD5' },
  midnight: { label: 'TS Cloud Midnight', mode: 'dark', primary: '#3B82F6', secondary: '#6366F1', sidebar: '#020617', background: '#0F172A', paper: '#1E293B', text: '#F8FAFC', secondaryText: '#CBD5E1', hover: '#2563EB', divider: '#334155' },
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
      MuiDrawer: { styleOverrides: { paper: { backgroundColor: selected.sidebar, color: dark ? selected.text : '#F8FAFC', borderRight: `1px solid ${selected.divider}` } } },
      MuiListItemButton: { styleOverrides: { root: ({ theme }) => ({ margin: '4px 10px', borderRadius: 8, '& .MuiListItemIcon-root': { color: 'inherit' }, '&.Mui-selected': { backgroundColor: alpha(theme.palette.primary.main, dark ? .32 : .18), color: dark ? selected.text : '#F8FAFC', '& .MuiListItemIcon-root': { color: 'inherit' } }, '&:hover, &.Mui-selected:hover': { backgroundColor: selected.hover, color: '#F8FAFC', '& .MuiListItemIcon-root': { color: '#F8FAFC' } } }) } },
      MuiButton: { styleOverrides: { root: { borderRadius: 8, minHeight: 40 } } },
      MuiTextField: { defaultProps: { size: 'small' } },
    },
  })
}

export default createAppTheme('classic')
