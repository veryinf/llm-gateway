import React from 'react';
import { cn } from '@/lib';

export type DescriptionsItem = {
  label: string;
  value: React.ReactNode;
  span?: number;
};

type DescriptionsProps = {
  title?: React.ReactNode;
  extra?: React.ReactNode;
  column?: number;
  items?: DescriptionsItem[];
  labelClassName?: string;
};

export function Descriptions(props: DescriptionsProps) {
  const { title, extra, column = 2, items = [] } = props;
  const gridCols = Array(column).fill('auto 1fr').join(' ');

  return (
    <div className="w-full">
      {(title || extra) && (
        <div className="flex justify-between items-center pb-4">
          {title && <div className="font-medium">{title}</div>}
          {extra && <div>{extra}</div>}
        </div>
      )}
      <div className="overflow-hidden rounded-md border text-sm">
        <div className="grid" style={{ gridTemplateColumns: gridCols }}>
          {items.map((item, index) => (
            <React.Fragment key={index}>
              <div className={cn('bg-muted px-2 py-2.5 border-r border-b text-right', props.labelClassName)}>{item.label}</div>
              <div className="min-w-0 px-2 py-2.5 border-r border-b last:border-r-0 whitespace-normal wrap-break-word">{item.value}</div>
            </React.Fragment>
          ))}
        </div>
      </div>
    </div>
  );
}
