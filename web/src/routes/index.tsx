import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'

export const Route = createFileRoute('/')({
  component: IndexPage,
})

function IndexPage() {
  const navigate = useNavigate()
  const token = localStorage.getItem('token')

  useEffect(() => {
    navigate({ to: token ? '/dashboard' : '/login', replace: true })
  }, [])

  return null
}
