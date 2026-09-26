const DEFAULT_INTERFACE_KEY_PREFIX = 'ts-cloud-default-router-interface:'

export function getDefaultRouterInterface(routerID: number | string): string {
  if (!routerID) return ''
  return localStorage.getItem(`${DEFAULT_INTERFACE_KEY_PREFIX}${routerID}`) ?? ''
}

export function saveDefaultRouterInterface(routerID: number | string, interfaceName: string): void {
  if (!routerID || !interfaceName.trim()) return
  localStorage.setItem(`${DEFAULT_INTERFACE_KEY_PREFIX}${routerID}`, interfaceName.trim())
}
