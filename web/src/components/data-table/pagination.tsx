import { ChevronLeftIcon, ChevronRightIcon, ChevronLeftIcon as DoubleArrowLeftIcon, ChevronRightIcon as DoubleArrowRightIcon } from 'lucide-react';
import { type Table } from '@tanstack/react-table';
import { cn, getPageNumbers } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

type DataTablePaginationProps<TData> = {
  table: Table<TData>;
  className?: string;
};

export function DataTablePagination<TData>({ table, className }: DataTablePaginationProps<TData>) {
  const pagination = table.getState().pagination;
  const currentPage = pagination.pageIndex + 1;
  const totalPages = table.getPageCount();
  const pageNumbers = getPageNumbers(currentPage, totalPages);

  return (
    <div className={cn('flex items-center justify-between overflow-clip px-2 gap-4', '@max-2xl/content:flex-col-reverse @max-2xl/content:gap-4', className)} style={{ overflowClipMargin: 1 }}>
      <div className="flex w-full items-center justify-end">
        <div className="flex w-70 items-center justify-center text-sm font-medium @2xl/content:hidden">
          显示 {pagination.pageIndex * pagination.pageSize + 1}-{Math.min(currentPage * pagination.pageSize, table.getRowCount())} 条，共 {table.getRowCount()} 条记录
        </div>
        <div className="flex w-[100px] items-center justify-center text-sm font-medium @max-3xl/content:hidden">
          页码 {currentPage}/{totalPages}
        </div>
        <div className="flex items-center gap-2 @max-2xl/content:flex-row-reverse">
          <Select
            value={`${table.getState().pagination.pageSize}`}
            onValueChange={(value) => {
              table.setPageSize(Number(value));
            }}
          >
            <SelectTrigger className="h-8 w-[70px]">
              <SelectValue placeholder={table.getState().pagination.pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              {[10, 20, 30, 40, 50].map((pageSize) => (
                <SelectItem key={pageSize} value={`${pageSize}`}>
                  {pageSize}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="hidden text-sm font-medium sm:block">条/页</p>
        </div>
      </div>

      <div className="flex items-center sm:space-x-6 lg:space-x-8">
        <div className="flex items-center space-x-2">
          <Button variant="outline" className="size-8 p-0 @max-md/content:hidden" onClick={() => table.firstPage()} disabled={!table.getCanPreviousPage()}>
            <DoubleArrowLeftIcon className="h-4 w-4" />
          </Button>
          <Button variant="outline" className="size-8 p-0" onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()}>
            <ChevronLeftIcon className="h-4 w-4" />
          </Button>

          {/* Page number buttons */}
          {pageNumbers.map((pageNumber, index) => (
            <div key={`${pageNumber}-${index}`} className="flex items-center">
              {pageNumber === '...' ? (
                <span className="text-muted-foreground px-1 text-sm">...</span>
              ) : (
                <Button variant={currentPage === pageNumber ? 'default' : 'outline'} className="h-8 min-w-8 px-2" onClick={() => table.setPageIndex((pageNumber as number) - 1)}>
                  {pageNumber}
                </Button>
              )}
            </div>
          ))}

          <Button variant="outline" className="size-8 p-0" onClick={() => table.nextPage()} disabled={!table.getCanNextPage()}>
            <ChevronRightIcon className="h-4 w-4" />
          </Button>
          <Button variant="outline" className="size-8 p-0 @max-md/content:hidden" onClick={() => table.lastPage()} disabled={!table.getCanNextPage()}>
            <DoubleArrowRightIcon className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
