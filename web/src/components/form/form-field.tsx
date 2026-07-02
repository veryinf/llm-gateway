import { isFunction, omit } from 'lodash-es';
import { Field, FieldDescription, FieldError, FieldLabel, FieldTitle } from '../ui/field';
import { Input } from '../ui/input';
import { Textarea } from '../ui/textarea';
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from '../ui/select';
import { useField } from '@tanstack/react-form';
import React from 'react';
import type { EasyFieldApi, EasyFieldOptions, EasyFormApi } from './utils';
import type { GroupOptionsItem, OptionsItem } from '@/lib';
import { Combobox, ComboboxChip, ComboboxChips, ComboboxChipsInput, ComboboxContent, ComboboxEmpty, ComboboxItem, ComboboxList, ComboboxValue, useComboboxAnchor } from '../ui/combobox';
import { Checkbox } from '../ui/checkbox';
import type { CheckedState } from '@radix-ui/react-checkbox';
import { Switch } from '../ui/switch';
import { Label } from '../ui/label';
import { Badge } from '../ui/badge';
import { CirclePlus, CircleQuestionMark, RefreshCcw, X } from 'lucide-react';
import { EasyTooltip } from '../easy-tooltip';

export type FormFieldProps<T extends any> = {
  form: EasyFormApi<T>;
  name: string;
  title?: string;
  description?: string | React.ReactNode;
  tips?: string;
  placeholder?: string;
  required?: boolean;
  titleMore?: React.ReactNode;
  className?: string;
  validators?: EasyFieldOptions<T>['validators'];
};

export function FormField<T extends any = any>(props: FormFieldProps<T> & { children?: React.ReactNode | ((field: EasyFieldApi<T>, isInvalid?: boolean) => React.ReactNode); }) {
  let validators = props.validators;
  if (!validators && props.required) {
    validators = {
      onChange: ({ value }: any) => (!value ? `${props.title}是必填项` : undefined),
    };
  }

  const field = useField({ name: props.name, form: props.form, validators });
  const isInvalid = !field.state.meta.isValid;
  return (
    <Field className={props.className}>
      {props.title && (
        <FieldTitle>
          <div className="flex gap-1 w-full items-center">
            <span>{props.title}</span>
            {props.required && <span className="text-red-600">*</span>}
            {props.tips && (
              <EasyTooltip tooltip={props.tips}>
                <span className="hover:text-primary">
                  <CircleQuestionMark size={18} />
                </span>
              </EasyTooltip>
            )}
            {props.titleMore && <span className="ml-auto flex items-end gap-1.5">{props.titleMore}</span>}
          </div>
        </FieldTitle>
      )}
      {isFunction(props.children) ? props.children(field, isInvalid) : props.children}
      {props.description && <FieldDescription>{props.description}</FieldDescription>}
      {isInvalid && <FieldError errors={field.state.meta.errors.map((e) => ({ message: typeof e === 'string' ? e : (e as any)?.message ?? String(e) }))} />}
    </Field>
  );
}

export function FormFieldInput<T extends any = any>(props: FormFieldProps<T> & { type?: 'text' | 'password' | 'number'; }) {
  function handleChange(field: EasyFieldApi<T>, e: React.ChangeEvent<HTMLInputElement>) {
    if (props.type === 'number') {
      field.handleChange(parseFloat(e.target.value));
    } else {
      field.handleChange(e.target.value);
    }
  }

  return (
    <FormField {...omit(props, ['type'])}>
      {(field, isInvalid) => {
        return <Input id={field.name} type={props.type ?? 'text'} placeholder={props.placeholder} required={props.required} value={field.state.value || ''} onBlur={field.handleBlur} onChange={(e) => handleChange(field, e)} aria-invalid={isInvalid} />;
      }}
    </FormField>
  );
}

export function FormFieldTextarea<T extends any = any>(props: FormFieldProps<T> & { rows?: number; }) {
  return (
    <FormField {...omit(props, [])}>
      {(field, isInvalid) => {
        return <Textarea id={field.name} placeholder={props.placeholder} required={props.required} value={field.state.value || ''} onBlur={field.handleBlur} rows={props.rows ?? 3} onChange={(e) => field.handleChange(e.target.value)} aria-invalid={isInvalid} />;
      }}
    </FormField>
  );
}

export function FormFieldCheckboxGroup<T extends any = any>(props: FormFieldProps<T> & { options: OptionsItem[] | GroupOptionsItem[]; columns?: number; }) {
  let validators = props.validators;
  if (!validators && props.required) {
    validators = {
      onChange: ({ value }: any) => (!value || value?.length === 0 ? `${props.title}是必填项` : undefined),
    };
  }

  const field = useField({ name: props.name, form: props.form, validators, mode: 'array' });
  const isInvalid = !field.state.meta.isValid;
  const handleChange = (value: CheckedState, optionValue: string | number) => {
    const stateValue: (string | number)[] = field.state.value ?? [];
    if (value === true) {
      if (!stateValue.includes(optionValue)) {
        field.pushValue(optionValue);
      }
    } else {
      field.removeValue(stateValue.indexOf(optionValue));
      //field.handleChange(stateValue.filter((v: any) => v !== optionValue));
    }
  };
  return (
    <FormField {...omit(props)}>
      <div className={`grid grid-cols-${props.columns ?? 4} gap-4`}>
        {props.options.length > 0 && 'options' in props.options[0]
          ? (props.options as GroupOptionsItem[]).map((_group: GroupOptionsItem) => <>不支持</>)
          : (props.options as OptionsItem[]).map((option: OptionsItem, index) => {
            console.log(field.state.value?.includes(option.value), field.state.value);
            return (
              <div className="flex items-center gap-2 py-1" key={option.value}>
                <Checkbox id={`${field.name}-${option.value}-${index}`} checked={field.state.value?.includes(option.value) || false} onCheckedChange={(value) => handleChange(value, option.value)} aria-invalid={isInvalid} />
                <FieldLabel htmlFor={`${field.name}-${option.value}-${index}`} className="font-normal">
                  {option.label}
                </FieldLabel>
              </div>
            );
          })}
      </div>
    </FormField>
  );
}

export function FormFieldSelect<T extends any = any>(props: FormFieldProps<T> & { options: OptionsItem[] | GroupOptionsItem[]; onCreate?: () => void; onRefresh?: () => void; }) {
  return (
    <FormField
      {...omit(props, ['options', 'onCreate', 'onRefresh', 'titleMore'])}
      titleMore={
        <>
          {props.onCreate && (
            <EasyTooltip tooltip="新增">
              <span className="hover:text-primary" onClick={props.onCreate}>
                <CirclePlus size={16} />
              </span>
            </EasyTooltip>
          )}
          {props.onRefresh && (
            <EasyTooltip tooltip="刷新选项数据">
              <span className="hover:text-primary" onClick={props.onRefresh}>
                <RefreshCcw size={16} />
              </span>
            </EasyTooltip>
          )}
          {props.titleMore}
        </>
      }
    >
      {(field, isInvalid) => {
        return (
          <Select value={field.state.value} onValueChange={(value) => value && field.handleChange(value)}>
            <SelectTrigger id={field.name} aria-invalid={isInvalid}>
              <SelectValue placeholder={props.placeholder} />
            </SelectTrigger>
            <SelectContent>
              {props.options.length > 0 && 'options' in props.options[0]
                ? (props.options as GroupOptionsItem[]).map((group: GroupOptionsItem) => (
                  <SelectGroup>
                    <SelectLabel>{group.label}</SelectLabel>
                    {group.options.map((option: OptionsItem) => (
                      <ComboboxItem key={option.value} value={option.value}>
                        {option.label}
                      </ComboboxItem>
                    ))}
                  </SelectGroup>
                ))
                : (props.options as OptionsItem[]).map((option: OptionsItem) => (
                  <SelectItem key={option.value} value={String(option.value)}>
                    {option.label}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        );
      }}
    </FormField>
  );
}

export function FormFieldCombobox<T extends any = any>(_props: FormFieldProps<T> & { options: OptionsItem[] | GroupOptionsItem[]; }) {
  const anchor = useComboboxAnchor();
  const frameworks = ['Next.js', 'SvelteKit', 'Nuxt.js', 'Remix', 'Astro'] as const;
  return (
    // <FormField {...omit(props, ['options'])}>
    <>
      {/* <Combobox value={field.state.value} onValueChange={(value) => field.handleChange(value)}>
          <ComboboxInput id={field.name} placeholder={props.placeholder} aria-invalid={isInvalid} />
          <ComboboxContent>{props.options.length > 0 && 'options'}</ComboboxContent>
        </Combobox> */}
      <Combobox multiple autoHighlight items={frameworks} defaultValue={[frameworks[0]]} onValueChange={(values) => console.log(values)}>
        <ComboboxChips ref={anchor} className="w-full col-span-3">
          <ComboboxValue>
            {(values) => {
              console.log(values);
              return (
                <>
                  {values.map((value: string) => (
                    <ComboboxChip key={value}>{value}</ComboboxChip>
                  ))}
                  <ComboboxChipsInput />
                </>
              );
            }}
          </ComboboxValue>
        </ComboboxChips>
        <ComboboxContent anchor={anchor}>
          <ComboboxEmpty>No items found.</ComboboxEmpty>
          <ComboboxList>
            {(item) => (
              <ComboboxItem key={item} value={item}>
                {item}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </>
    // </FormField>
  );
}

export function FormFieldSwitch<T extends any = any>(props: FormFieldProps<T> & { switchLabel?: string; }) {
  return (
    <FormField {...omit(props)}>
      {(field, isInvalid) => {
        return (
          <div className="flex items-center space-x-2 h-9">
            <Switch id={field.name} aria-invalid={isInvalid} checked={field.state.value} onCheckedChange={(e) => field.handleChange(e)} />
            <Label htmlFor={field.name}>{props.switchLabel ?? '开关'}</Label>
          </div>
        );
      }}
    </FormField>
  );
}

export function FormFieldCheckbox<T extends any = any>(props: FormFieldProps<T>) {
  return (
    <FormField {...omit(props)}>
      {(field, isInvalid) => {
        return (
          <div className="flex items-center space-x-2 h-9">
            <Checkbox id={field.name} aria-invalid={isInvalid} checked={field.state.value} onCheckedChange={(e) => field.handleChange(e)} />
            <Label htmlFor={field.name}>{props.title}</Label>
          </div>
        );
      }}
    </FormField>
  );
}

export function FormFieldTagsInput<T extends any = any>(props: FormFieldProps<T>) {
  const [inputValue, setInputValue] = React.useState('');
  const inputRef = React.useRef<HTMLInputElement>(null);

  return (
    <FormField {...props}>
      {(field) => {
        const tags: string[] = (field.state.value as string[] | undefined) ?? [];

        function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
          if (e.key === 'Enter') {
            e.preventDefault();
            const tag = inputValue.trim();
            if (tag && !tags.includes(tag)) {
              field.handleChange([...tags, tag] as any);
              setInputValue('');
            }
          } else if (e.key === 'Backspace' && inputValue === '' && tags.length > 0) {
            field.handleChange(tags.slice(0, -1) as any);
          }
        }

        function removeTag(index: number) {
          field.handleChange(tags.filter((_, i) => i !== index) as any);
        }

        return (
          <div
            className="flex flex-wrap gap-1.5 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2"
            onClick={() => inputRef.current?.focus()}
          >
            {tags.map((tag, index) => (
              <Badge key={index} variant="secondary" className="gap-1 pr-1">
                <span className="max-w-[200px] truncate">{tag}</span>
                <button type="button" onClick={(e) => { e.stopPropagation(); removeTag(index); }} className="ml-0.5 rounded-full outline-none ring-offset-background hover:bg-destructive/20">
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
            <Input
              ref={inputRef}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={tags.length === 0 ? props.placeholder : ''}
              className="h-6 flex-1 border-0 bg-transparent p-0 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
            />
          </div>
        );
      }}
    </FormField>
  );
}
