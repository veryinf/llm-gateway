import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useConfirm } from '@/components/confirm';
import { apiKeyService, type APIKey } from '@/services/api-key';
import { useUsers } from '@/hooks/use-users';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import type { API } from '@/typings';

export const Route = createFileRoute('/app-keys')({
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
  const { confirmHandler, Confirm } = useConfirm();
  const queryClient = useQueryClient();

  const { users } = useUsers();

  const userOptions = useMemo(
    () => users.map((u) => ({ label: u.name || u.username, value: String(u.uid) })),
    [users],
  );

  const appKeyService: API.Service<APIKey> = useMemo(
    () => ({
      primaryKey: (entity) => entity.id,
      title: (entity) => entity.name || entity.key,

      async search(params) {
        let list = await apiKeyService.listAll();
        if (params.filters) {
          for (const f of params.filters) {
            if (f.field === 'user_id' && Array.isArray(f.value) && f.value.length > 0) {
              list = list.filter((k) => (f.value as string[]).includes(String(k.user_id)));
            }
          }
        }
        return { errCode: 0, errMsg: 'ok', dataSet: list, total: list.length };
      },

      async fetch() {
        return { errCode: 0, errMsg: 'ok', data: undefined };
      },

      async add() {
        return { errCode: 0, errMsg: 'ok' };
      },

      async update() {
        return { errCode: 0, errMsg: 'ok' };
      },

      async delete(id) {
        await apiKeyService.deleteGlobal(id);
        return { errCode: 0, errMsg: 'ok' };
      },
    }),
    [],
  );

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const handleToggle = async (entity: APIKey) => {
    try {
      const result = await apiKeyService.toggleActive(entity.id);
      toast.success(result.is_active ? '已启用' : '已禁用');
      queryClient.invalidateQueries({ queryKey: ['full-page', 'app-keys'] });
    } catch {
      toast.error('操作失败');
    }
  };

  const columns: ColumnDef<APIKey, any>[] = [
    {
      accessorKey: 'user_id',
      header: '用户',
      enableColumnFilter: true,
      meta: { label: '用户', className: 'w-[100px]', emuns: userOptions },
      cell: ({ row }) => {
        const user = users.find((u) => u.uid === row.original.user_id);
        return user?.name || user?.username || row.original.user_id;
      },
    },
    {
      accessorKey: 'name',
      header: '名称',
      meta: { label: '名称', className: 'w-[120px]' },
    },
    {
      accessorKey: 'key',
      header: 'Key',
      meta: { label: 'Key', className: 'w-[280px]' },
      cell: ({ row }) => (
        <span className="font-mono text-xs break-all">{row.original.key}</span>
      ),
    },
    {
      accessorKey: 'quota_limit',
      header: '配额',
      meta: { label: '配额', className: 'w-[80px]' },
      cell: ({ row }) => {
        const v = row.original.quota_limit;
        return v > 0 ? v.toLocaleString() : '不限';
      },
    },
    {
      accessorKey: 'quota_used',
      header: '已使用',
      meta: { label: '已使用', className: 'w-[80px]' },
      cell: ({ row }) => row.original.quota_used.toLocaleString(),
    },
    {
      accessorKey: 'rate_limit_qpm',
      header: 'QPM',
      meta: { label: 'QPM', className: 'w-[70px]' },
      cell: ({ row }) => row.original.rate_limit_qpm || '不限',
    },
    {
      accessorKey: 'is_active',
      header: '状态',
      meta: { label: '状态', className: 'w-[70px]' },
      cell: ({ row }) => (
        <Badge variant={row.original.is_active ? 'default' : 'destructive'}>
          {row.original.is_active ? '启用' : '禁用'}
        </Badge>
      ),
    },
    {
      accessorKey: 'created_at',
      header: '创建时间',
      meta: { label: '创建时间', className: 'w-[100px]' },
      cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
    },
    {
      id: 'actions',
      header: '操作',
      enableSorting: false,
      enableHiding: false,
      meta: { className: 'w-24' },
      cell: ({ row }) => {
        const entity = row.original;
        return (
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleToggle(entity)}
            >
              {entity.is_active ? '禁用' : '启用'}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive"
              onClick={() => {
                confirmHandler.confirmInvoke(
                  '确认删除',
                  async () => {
                    await appKeyService.delete(entity.id);
                    toast.success('API Key 已删除');
                    queryClient.invalidateQueries({ queryKey: ['full-page', 'app-keys'] });
                    return true;
                  },
                  `确认要删除 API Key「${entity.name}」吗？`,
                  true,
                );
              }}
            >
              删除
            </Button>
          </div>
        );
      },
    },
  ];

  return (
    <>
      <Page<APIKey>
        infomation={pageInformation}
        columns={columns}
        service={appKeyService}
        options={{ showSelectColumn: false }}
      />
      <Confirm />
    </>
  );
}
