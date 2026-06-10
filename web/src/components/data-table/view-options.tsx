import { Check, Settings2 } from 'lucide-react';
import { cn } from '@/lib';
import { type Table } from '@tanstack/react-table';
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '../ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '../ui/popover';
import { EasyButton } from '../easy-button';

type DataTableViewOptionsProps<TData> = {
  table: Table<TData>;
};

export function DataTableViewOptions<TData>({ table }: DataTableViewOptionsProps<TData>) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <EasyButton variant="outline" size="sm" className="ms-auto hidden h-8 lg:flex" tooltip="显示/隐藏列" tooltipProps={{ side: 'left' }}>
          <Settings2 className="size-4" />
        </EasyButton>
      </PopoverTrigger>
      <PopoverContent className="w-44 p-0">
        <Command>
          <CommandInput placeholder="查找列..." />
          <CommandList>
            <CommandEmpty>没有找到匹配列.</CommandEmpty>
            <CommandGroup>
              {table
                .getAllColumns()
                .filter((column) => typeof column.accessorFn !== 'undefined' && column.getCanHide())
                .map((column) => (
                  <CommandItem key={column.id} onSelect={() => column.toggleVisibility(!column.getIsVisible())}>
                    <span className="truncate">{column.columnDef.meta?.label ?? column.id}</span>
                    <Check className={cn('ml-auto size-4 shrink-0', column.getIsVisible() ? 'opacity-100' : 'opacity-0')} />
                  </CommandItem>
                ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
