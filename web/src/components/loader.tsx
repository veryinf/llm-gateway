import { cn } from '@/lib';
import { Loader } from 'lucide-react';

const LoadingIcon = Loader;
export function Loading(props: React.ComponentProps<typeof LoadingIcon>) {
  return <LoadingIcon {...props} className={cn('animate-[spin_2s_linear_infinite] text-gray-700', props.className)} />;
}
