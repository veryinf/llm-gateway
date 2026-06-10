import React from 'react';
import { sumBy } from 'lodash-es';
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

  const rows: DescriptionsItem[][] = [];
  let rowIndex = 0;
  items.forEach((item) => {
    rows[rowIndex] = rows[rowIndex] || [];
    if (sumBy(rows[rowIndex], (i) => i.span ?? 1) + (item.span ?? 1) > column) {
      rowIndex++;
    }
    rows[rowIndex] = rows[rowIndex] || [];
    rows[rowIndex].push(item);
  });

  return (
    <div className="w-full">
      {(title || extra) && (
        <>
          <div className="flex justify-between items-center pb-4">
            {title && <div className="font-medium">{title}</div>}
            {extra && <div>{extra}</div>}
          </div>
        </>
      )}
      <div className="overflow-hidden rounded-md border text-sm">
        {rows.map((row, rowIndex) => (
          <div key={rowIndex} className={`flex border-b last:border-b-0`}>
            {row.map((item, itemIndex) => (
              <React.Fragment key={itemIndex}>
                <div className={cn('bg-muted px-2 py-2.5 border-r text-right', props.labelClassName)}>{item.label}</div>
                <div className={`flex-1 min-w-0 px-2 py-2.5 border-r last:border-r-0 whitespace-normal wrap-break-word`}>{item.value}</div>
              </React.Fragment>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
