import { cn } from '@/lib/utils'
import { Loader } from 'lucide-react'

export function Loading(props: React.ComponentProps<typeof Loader>) {
  return <Loader {...props} className={cn('animate-[spin_2s_linear_infinite] text-gray-700', props.className)} />
}
