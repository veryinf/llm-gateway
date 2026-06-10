import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useConfirm } from '@/components/confirm';
import { apiKeyService, type APIKey } from '@/services/api-key';
import { toast } from 'sonner';
import { useQueryClient } from '@tanstack/react-query';
import type { API } from '@/typings';

export const Route = createFileRoute('/app-keys')({
  component: AppKeysPage,
});

const appKeyService: API.Service<APIKey> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.name || entity.key_prefix,

  async search() {
    const list = await apiKeyService.listAll();
    return { list, total: list.length };
  },

  async fetch() {
    return { data: undefined };
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
};

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
      header: '用户 ID',
      meta: { label: '用户 ID' },
    },
    {
      accessorKey: 'name',
      header: '名称',
      meta: { label: '名称' },
    },
    {
      accessorKey: 'key_prefix',
      header: 'Key 前缀',
      meta: { label: 'Key 前缀' },
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.key_prefix}...</span>
      ),
    },
    {
      accessorKey: 'quota_limit',
      header: '配额',
      meta: { label: '配额' },
      cell: ({ row }) => {
        const v = row.original.quota_limit;
        return v > 0 ? v.toLocaleString() : '不限';
      },
    },
    {
      accessorKey: 'quota_used',
      header: '已使用',
      meta: { label: '已使用' },
      cell: ({ row }) => row.original.quota_used.toLocaleString(),
    },
    {
      accessorKey: 'rate_limit_qpm',
      header: 'QPM',
      meta: { label: 'QPM' },
      cell: ({ row }) => row.original.rate_limit_qpm || '不限',
    },
    {
      accessorKey: 'is_active',
      header: '状态',
      meta: { label: '状态' },
      cell: ({ row }) => (
        <Badge variant={row.original.is_active ? 'default' : 'destructive'}>
          {row.original.is_active ? '启用' : '禁用'}
        </Badge>
      ),
    },
    {
      accessorKey: 'created_at',
      header: '创建时间',
      meta: { label: '创建时间' },
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
