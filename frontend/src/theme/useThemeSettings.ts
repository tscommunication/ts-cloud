import { createContext, useContext } from 'react'
import type { ThemeName } from './theme'

export const ThemeSettingsContext = createContext<{
  themeName: ThemeName
  setThemeName: (theme: ThemeName) => void
} | null>(null)

export function useThemeSettings() {
  const context = useContext(ThemeSettingsContext)
  if (!context) {
    throw new Error('useThemeSettings must be used inside ThemeSettingsProvider')
  }
  return context
}
