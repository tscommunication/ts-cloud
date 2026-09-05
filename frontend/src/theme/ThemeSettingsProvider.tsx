import { useMemo, useState, type ReactNode } from 'react'
import { CssBaseline } from '@mui/material'
import { ThemeProvider } from '@mui/material/styles'
import { createAppTheme, isThemeName, type ThemeName } from './theme'
import { ThemeSettingsContext } from './useThemeSettings'

export function ThemeSettingsProvider({ children }: { children: ReactNode }) {
  const [themeName, updateThemeName] = useState<ThemeName>(() => {
    const saved = localStorage.getItem('ts-cloud-theme')
    if (isThemeName(saved)) return saved
    if (localStorage.getItem('ts-cloud-theme-mode') === 'dark') return 'midnight'
    const legacy = localStorage.getItem('ts-cloud-theme-color')
    return legacy === 'indigo' ? 'royal' : legacy === 'green' ? 'emerald' : legacy === 'orange' ? 'sunset' : 'classic'
  })
  const value = useMemo(() => ({
    themeName,
    setThemeName: (next: ThemeName) => { localStorage.setItem('ts-cloud-theme', next); updateThemeName(next) },
  }), [themeName])
  const theme = useMemo(() => createAppTheme(themeName), [themeName])
  return <ThemeSettingsContext.Provider value={value}><ThemeProvider theme={theme}><CssBaseline />{children}</ThemeProvider></ThemeSettingsContext.Provider>
}
