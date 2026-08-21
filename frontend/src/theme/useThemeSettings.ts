import { createContext, useContext } from 'react'
import type { PaletteMode } from '@mui/material/styles'
import type { ThemeColor } from './theme'

export const ThemeSettingsContext = createContext<{
  mode: PaletteMode
  color: ThemeColor
  setMode: (mode: PaletteMode) => void
  setColor: (color: ThemeColor) => void
} | null>(null)

export function useThemeSettings() {
  const context = useContext(ThemeSettingsContext)
  if (!context) {
    throw new Error('useThemeSettings must be used inside ThemeSettingsProvider')
  }
  return context
}
