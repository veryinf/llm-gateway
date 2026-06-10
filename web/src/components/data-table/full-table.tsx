import { Trash2 } from 'lucide-react';
import { EasyButton } from '@/components/easy-button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { cn } from '@/lib/utils';
import { flexRender, getCoreRowModel, useReactTable, type CellContext, type ColumnDef } from '@tanstack/react-table';
import { DataTableToolbar } from './toolbar';
import { DataTablePagination } from './pagination';
import { DataTableBulkActions } from './bulk-actions';
import { useEffect, useMemo, useRef, type ComponentProps } from 'react';
import { Loading } from '../loader';
import { useTableUrlState } from './use-table-url-state';
import { DataTableColumnHeader } from './column-header';
import type { API } from '@/typings';

type FullTableProps<T> = {
  entityName?: string;
  columns: ColumnDef<T, {}>[];
  loading?: boolean;
  data: T[];
  total?: number;
  onRefresh?: (state: FullTableState) => void;
};

export type FullTableState = API.SearchParams;

export function FullTable<T>(props: FullTableProps<T>) {
  const requestRef = useRef<FullTableState | undefined>(undefined);
  const filterColumns = props.columns.filter((column) => column.enableColumnFilter && column.meta?.emuns) ?? [];

  const { globalFilter, onGlobalFilterChange, columnFilters, onColumnFiltersChange, pagination, onPaginationChange } = useTableUrlState({
    pagination: { defaultPage: 1, defaultPageSize: 10 },
    globalFilter: { enabled: true },
    columnFilters: filterColumns.map((column) => {
      const columnId = column.id ?? (column as any).accessorKey ?? '';
      return {
        columnId,
        searchKey: columnId,
        type: 'array',
      };
    }),
  });
  const requestState: FullTableState = {
    kw: globalFilter,
    filters: columnFilters,
    pagination,
  };

  useEffect(() => {
    if (props.onRefresh) {
      if (JSON.stringify(requestRef.current) != JSON.stringify(requestState)) {
        props.onRefresh!(requestState);
        requestRef.current = requestState;
      }
    }
  }, [requestState, props.onRefresh]);

  const columns = useMemo(() => {
    //增加列定义
    return props.columns.map((c) => {
      const column = { ...c };
      if (column.enableColumnFilter || column.enableSorting || column.enableHiding) {
        const title = typeof column.header === 'string' ? column.header : column.meta?.label || column.id || (column as any).accessorKey || '';
        column.header = (r) => {
          return <DataTableColumnHeader column={r.column} title={title} />;
        };
      }
      if (!column.cell && column.meta?.emuns) {
        column.cell = (props: CellContext<T, any>) => {
          const value = props.getValue();
          const option = column.meta!.emuns!.find((e) => e.value === value);
          return option?.label || value;
        };
      }
      return column;
    });
  }, [props.columns]);

  const table = useReactTable<T>({
    state: {
      globalFilter,
      columnFilters,
      pagination,
    },
    columns,
    data: props.data,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    rowCount: props.total || props.data.length,
    autoResetPageIndex: true,
    manualSorting: true,
    manualFiltering: true,
    onPaginationChange,
    onGlobalFilterChange,
    onColumnFiltersChange,
  });

  const toolbarFilters: ComponentProps<typeof DataTableToolbar>['filters'] = filterColumns.map((column) => {
    const columnId = column.id ?? (column as any).accessorKey ?? '';
    return {
      columnId,
      title: column.meta?.label || columnId,
      options: (column.meta?.emuns || []) as any,
    };
  });

  return (
    <div className={cn('max-sm:has-[div[role="toolbar"]]:mb-16', 'flex flex-1 flex-col gap-4')}>
      <DataTableToolbar table={table} filters={toolbarFilters} />
      <div className="overflow-hidden rounded-md border relative">
        {props.loading && (
          <div className="absolute inset-0 bg-white/50 backdrop-blur-sm z-10 flex items-center justify-center">
            <Loading />
          </div>
        )}
        {table.getRowModel().rows?.length ? (
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => {
                    return (
                      <TableHead
                        key={header.id}
                        colSpan={header.colSpan}
                        className={cn(header.column.columnDef.meta?.className, header.column.columnDef.meta?.thClassName, 'first:pl-4')}
                      >
                        {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                      </TableHead>
                    );
                  })}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id} data-state={row.getIsSelected() && 'selected'}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id} className={cn(cell.column.columnDef.meta?.className, cell.column.columnDef.meta?.tdClassName, 'first:pl-4')}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center">
                    没有数据.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        ) : (
          <div className="text-center p-8">没有数据.</div>
        )}
      </div>
      <DataTablePagination table={table} className="mt-auto" />
      <DataTableBulkActions table={table} entityName="条记录">
        <EasyButton variant="destructive" size="icon" onClick={() => alert(`确认删除这些${props.entityName}?`)} className="size-8" aria-label="删除选择的记录" title="删除选择的记录" tooltip="删除选择的记录">
          <Trash2 />
        </EasyButton>
      </DataTableBulkActions>
    </div>
  );
}
