import { XIcon } from 'lucide-react';
import { type Table } from '@tanstack/react-table';
import { Button } from '@/components/ui/button';
import { DataTableFacetedFilter } from './faceted-filter';
import { DataTableViewOptions } from './view-options';
import { InputGroup, InputGroupInput, InputGroupAddon, InputGroupButton } from '../ui/input-group';
import { useState } from 'react';
import type { OptionsItem } from '@/lib';

type DataTableToolbarProps<TData> = {
  table: Table<TData>;
  searchPlaceholder?: string;
  searchKey?: string;
  filters?: {
    columnId: string;
    title: string;
    options: OptionsItem[];
  }[];
};

export function DataTableToolbar<TData>({ table, searchPlaceholder = '输入关键字查询...', filters = [] }: DataTableToolbarProps<TData>) {
  const [kw, setKW] = useState('');
  const isFiltered = table.getState().columnFilters.length > 0 || table.getState().globalFilter;

  function handleSearch() {
    if (kw) {
      table.setGlobalFilter(kw);
    } else {
      table.setGlobalFilter('');
    }
  }

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 flex-col-reverse items-start gap-y-2 sm:flex-row sm:items-center sm:space-x-2">
        <InputGroup className="h-8 w-50">
          <InputGroupInput
            placeholder={searchPlaceholder}
            value={kw}
            onChange={(e) => {
              setKW(e.target.value);
            }}
          />
          <InputGroupAddon align="inline-end">
            <InputGroupButton variant="secondary" onClick={handleSearch}>
              搜索
            </InputGroupButton>
          </InputGroupAddon>
        </InputGroup>
        <div className="flex gap-x-2">
          {filters.map((filter) => {
            const column = table.getColumn(filter.columnId);
            if (!column) return null;
            return <DataTableFacetedFilter key={filter.columnId} column={column} title={filter.title} options={filter.options} />;
          })}
        </div>
        {isFiltered && (
          <Button
            variant="outline"
            onClick={() => {
              table.resetColumnFilters();
              table.setGlobalFilter('');
              setKW('');
            }}
            className="h-8 px-2 lg:px-3"
          >
            清除
            <XIcon className="h-4 w-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
    </div>
  );
}
