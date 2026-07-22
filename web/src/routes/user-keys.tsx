import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { userKeyService, type UserKey } from '@/services/api-key';
import { useAllUsers } from '@/services/user';
import { CopyButton } from '@/components/easy-button';

export const Route = createFileRoute('/user-keys')({
  validateSearch: (search: Record<string, unknown>) => ({
    uid: Array.isArray(search.uid) ? search.uid : undefined,
  }),
  component: AppKeysPage,
});

const pageInformation: PageInformation = {
  name: 'app-keys',
  entityName: 'API Key',
  page: { title: 'API Key 管理', description: '管理所有用户的 API Key' },
  breadcrumbs: [{ title: '管理' }, { title: 'API Key 管理' }],
};

function AppKeysPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const { allUserOptions, isLoading } = useAllUsers();


  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const columns: ColumnDef<UserKey, any>[] = [
    {
      accessorKey: 'title',
      header: '名称',
      meta: { label: '名称', viewDetail: true, className: 'w-40' },
    },
    {
      accessorKey: 'uid',
      header: '用户',
      enableColumnFilter: true,
      meta: { label: '用户', className: 'w-40', emuns: allUserOptions },
    },
    {
      accessorKey: 'key',
      header: 'Key',
      meta: { label: 'Key', className: '' },
      cell: ({ row }) => (
        <div className="flex items-center gap-1">
          <span className="font-mono text-xs break-all">{row.original.key}</span>
          <CopyButton text={row.original.key} />
        </div>
      ),
    },
    {
      accessorKey: 'isActive',
      header: '状态',
      meta: { label: '状态', className: 'w-[70px]' },
      cell: ({ row }) => (
        <Badge variant={row.original.isActive ? 'default' : 'destructive'}>
          {row.original.isActive ? '启用' : '禁用'}
        </Badge>
      ),
    },
  ];

  return (
    <Page<UserKey>
      infomation={pageInformation}
      ready={!isLoading}
      columns={columns}
      service={userKeyService}
      options={{ showSelectColumn: false }}
      optionColumn={(column, domRender) => ({ ...column, cell: (res) => domRender(res.row.original) })}
      formInitialValue={(formType, entity) => (formType == 'add' ? {
        keyId: 0,
        uid: '' as any,
        key: '',
        title: '',
        isActive: true,
      } : {
        ...entity!,
        uid: (entity?.uid ? String(entity.uid) : '') as any,
      })}
      renderViewDetail={(entity) =>
        <Descriptions
          title="API Key 信息"
          labelClassName='w-30'
          column={2}
          items={[
            { label: '名称', value: entity.title || '-' },
            { label: '所属用户', value: allUserOptions.find(u => u.value === entity.uid)?.label },
            { label: 'Key', value: <div className='flex items-center gap-1'><span className='font-mono text-xs break-all'>{entity.key}</span><CopyButton text={entity.key} /></div>, span: 2 },
            {
              label: '状态',
              value: (
                <Badge variant={entity.isActive ? 'default' : 'destructive'}>
                  {entity.isActive ? '启用' : '禁用'}
                </Badge>
              ),
            },
          ]}
        />
      }
      renderViewForm={(form, _entity, _formType) => (
        <div className="grid grid-cols-12 gap-4">
          <FormFieldSelect className="col-span-6" form={form} name="uid" title="用户" required options={allUserOptions} placeholder="请选择用户" />
          <FormFieldInput className="col-span-6" form={form} name="title" title="名称" required placeholder="例如: production-key" />
          <FormFieldSwitch className="col-span-12" form={form} name="isActive" title="启用" switchLabel="启用此 API Key" />
        </div>
      )}
    />
  );
}
