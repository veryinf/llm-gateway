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
          {items.map((item, index) => {
            const rowIndex = Math.floor(index / column);
            const colIndex = index % column;
            const totalRows = Math.ceil(items.length / column);
            const isLastRow = rowIndex === totalRows - 1;
            const isLastCol = colIndex === column - 1;

            return (
              <React.Fragment key={index}>
                <div
                  className={cn(
                    'bg-muted px-2 py-2.5 border-r text-right',
                    !isLastRow && 'border-b',
                    props.labelClassName,
                  )}
                >
                  {item.label}
                </div>
                <div
                  className={cn(
                    'min-w-0 px-2 py-2.5 whitespace-normal wrap-break-word',
                    !isLastCol && 'border-r',
                    !isLastRow && 'border-b',
                  )}
                >
                  {item.value}
                </div>
              </React.Fragment>
            );
          })}
        </div>
      </div>
    </div>
  );
}
