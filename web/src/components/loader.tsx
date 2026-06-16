import { cn } from '@/lib';
import { Loader } from 'lucide-react';
import { PageHeader } from './page-header';

const LoadingIcon = Loader;

export function Loading(props: React.ComponentProps<typeof LoadingIcon>) {
  return <LoadingIcon {...props} className={cn('animate-[spin_2s_linear_infinite] text-gray-700', props.className)} />;
}

export function LoadingPage() {
  return <div className="flex flex-1 flex-col">
    <div className="@container/main flex flex-1 flex-col gap-2">
      <div className="flex flex-col gap-4 py-4 px-4">
        <PageHeader title="Loading" />
        <Loading />
      </div>
    </div>
  </div>;
}