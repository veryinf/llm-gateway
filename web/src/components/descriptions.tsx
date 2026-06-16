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
  const gridCols = `repeat(${column * 2}, minmax(min-content, 1fr))`;

  // 预计算布局：每对占 span*2 列，累积判断行边界（含溢出换行）
  const layoutInfo: { span: number; gridColStart: number; gridColEnd: number; isEndOfRow: boolean; isInLastRow: boolean }[] = [];
  let curCol = 1;
  let prevIdx = -1;

  for (const item of items) {
    const span = Math.min(item.span ?? 1, column);
    const needed = span * 2;

    if (curCol + needed > column * 2 + 1) {
      if (prevIdx >= 0) layoutInfo[prevIdx].isEndOfRow = true;
      curCol = 1;
    }

    const gridColStart = curCol;
    const gridColEnd = curCol + needed;
    layoutInfo.push({ span, gridColStart, gridColEnd, isEndOfRow: false, isInLastRow: false });
    prevIdx = layoutInfo.length - 1;
    curCol = gridColEnd;
  }
  if (prevIdx >= 0) layoutInfo[prevIdx].isEndOfRow = true;

  // 反向遍历标记最后一行所有 item
  for (let i = layoutInfo.length - 1; i >= 0; i--) {
    layoutInfo[i].isInLastRow = true;
    if (layoutInfo[i].isEndOfRow) break;
  }

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
            const { gridColStart, gridColEnd, isEndOfRow, isInLastRow } = layoutInfo[index];
            return (
              <div
                key={index}
                className={cn(
                  'flex min-w-0',
                  !isEndOfRow && 'border-r',
                  !isInLastRow && 'border-b',
                )}
                style={{ gridColumn: `${gridColStart} / ${gridColEnd}` }}
              >
                <div
                  className={cn(
                    'bg-muted px-2 py-2.5 border-r text-right shrink-0',
                    props.labelClassName,
                  )}
                >
                  {item.label}
                </div>
                <div className="min-w-0 px-2 py-2.5 whitespace-normal wrap-break-word flex-1">
                  {item.value}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
