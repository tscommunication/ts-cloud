import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'
import { CssBaseline } from '@mui/material'
import { ThemeProvider, type PaletteMode } from '@mui/material/styles'
import { createAppTheme, type ThemeColor } from './theme'

const ThemeSettingsContext = createContext<{
  mode: PaletteMode; color: ThemeColor; setMode: (mode: PaletteMode) => void; setColor: (color: ThemeColor) => void
} | null>(null)

export function ThemeSettingsProvider({ children }: { children: ReactNode }) {
  const [mode, updateMode] = useState<PaletteMode>(() => localStorage.getItem('ts-cloud-theme-mode') === 'dark' ? 'dark' : 'light')
  const [color, updateColor] = useState<ThemeColor>(() => {
    const saved = localStorage.getItem('ts-cloud-theme-color')
    return saved === 'indigo' || saved === 'green' || saved === 'orange' ? saved : 'blue'
  })
  const value = useMemo(() => ({
    mode, color,
    setMode: (next: PaletteMode) => { localStorage.setItem('ts-cloud-theme-mode', next); updateMode(next) },
    setColor: (next: ThemeColor) => { localStorage.setItem('ts-cloud-theme-color', next); updateColor(next) },
  }), [mode, color])
  const theme = useMemo(() => createAppTheme(mode, color), [mode, color])
  return <ThemeSettingsContext.Provider value={value}><ThemeProvider theme={theme}><CssBaseline />{children}</ThemeProvider></ThemeSettingsContext.Provider>
}

export function useThemeSettings() {
  const context = useContext(ThemeSettingsContext)
  if (!context) throw new Error('useThemeSettings must be used inside ThemeSettingsProvider')
  return context
}
