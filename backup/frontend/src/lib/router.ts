export type Route = 'browse' | 'downloads' | 'settings'

export function getRouteFromPath(): Route {
  const path = window.location.pathname.toLowerCase()
  if (path.startsWith('/downloads')) return 'downloads'
  if (path.startsWith('/settings')) return 'settings'
  return 'browse'
}

export function navigateTo(route: Route) {
  const targetPath = route === 'browse' ? '/' : `/${route}`
  if (window.location.pathname !== targetPath) {
    window.history.pushState({}, '', targetPath)
    window.dispatchEvent(new PopStateEvent('popstate'))
  }
}
