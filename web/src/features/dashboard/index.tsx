import React from 'react'
import { useApps } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import AppList from './components/AppList'
import { Loader2 } from 'lucide-react'

function Dashboard() {
  const { data: apps, isLoading, error } = useApps()
  const setApps = useAppStore((state) => state.setApps)

  React.useEffect(() => {
    if (apps) {
      setApps(apps)
    }
  }, [apps, setApps])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center text-destructive">
        Failed to load apps. Please try again.
      </div>
    )
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground mt-2">
          Manage your self-hosted applications
        </p>
      </div>
      <AppList />
    </div>
  )
}

export default Dashboard
