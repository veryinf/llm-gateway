import { Check, EyeOff, Funnel } from 'lucide-react';
import { type Column } from '@tanstack/react-table';
import { cn } from '@/lib/utils';
import { EasyButton } from '../easy-button';
import { Popover, PopoverContent, PopoverTrigger } from '../ui/popover';
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList, CommandSeparator } from '../ui/command';

type DataTableColumnHeaderProps<TData, TValue> = React.HTMLAttributes<HTMLDivElement> & {
  column: Column<TData, TValue>;
  title: string;
};

export function DataTableColumnHeader<TData, TValue>({ column, title, className }: DataTableColumnHeaderProps<TData, TValue>) {
  const selectedValues = new Set(column?.getFilterValue() as (string | number)[]);
  const isFilter = selectedValues.size > 0;
  return (
    <div className={cn('flex items-center', className)}>
      <span>{title}</span>
      {column.columnDef.enableColumnFilter && column.columnDef.meta?.emuns && (
        <Popover>
          <PopoverTrigger asChild>
            <EasyButton variant="ghost" size="icon-xs" className="data-[state=open]:bg-accent h-8" tooltip="筛选此列">
              <Funnel className={`${isFilter ? 'text-primary' : 'text-muted-foreground'} size-3.5`} />
            </EasyButton>
          </PopoverTrigger>
          <PopoverContent className="w-44 p-0">
            <Command>
              <CommandInput placeholder="查找..." />
              <CommandList>
                <CommandEmpty>没有找到匹配项目.</CommandEmpty>
                <CommandGroup>
                  {column.columnDef.meta!.emuns!.map((option) => {
                    const isSelected = selectedValues.has(option.value);
                    return (
                      <CommandItem
                        key={option.value}
                        onSelect={() => {
                          if (isSelected) {
                            selectedValues.delete(option.value);
                          } else {
                            selectedValues.add(option.value);
                          }
                          const filterValues = Array.from(selectedValues);
                          column?.setFilterValue(filterValues.length ? filterValues : undefined);
                        }}
                        title={String(option.text)}
                      >
                        <span className="truncate">{option.text ?? option.label}</span>
                        <Check className={cn('ml-auto size-4 shrink-0', isSelected ? 'opacity-100' : 'opacity-0')} />
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
                {isFilter && (
                  <>
                    <CommandSeparator />
                    <CommandGroup>
                      <CommandItem onSelect={() => column?.setFilterValue(undefined)} className="justify-center text-center">
                        清除筛选
                      </CommandItem>
                    </CommandGroup>
                  </>
                )}
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      )}
      {column.getCanHide() && false && (
        <EasyButton variant="ghost" size="icon-xs" className="data-[state=open]:bg-accent h-8" tooltip="隐藏此列" onClick={() => column.toggleVisibility(false)}>
          <EyeOff className="text-muted-foreground size-3.5" />
        </EasyButton>
      )}
    </div>
  );
}
