import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSwitch, FormFieldTextarea } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { toast } from 'sonner';
import { Trash2 } from 'lucide-react';
import { userModelService, type UserModel } from '@/services/user-model';
import { userModelRouterService } from '@/services/user-model-router';
import { providerModelService, type ProviderModel } from '@/services/provider-model';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

export const Route = createFileRoute('/user-models')({
  component: UserModelsPage,
});

const pageInformation: PageInformation = {
  name: 'user-models',
  entityName: '模型',
  page: { title: '用户端模型', description: '配置暴露给用户的模型列表' },
  breadcrumbs: [{ title: '下游' }, { title: '用户端模型' }],
};

const columns: ColumnDef<UserModel, any>[] = [
  {
    accessorKey: 'name',
    header: '模型名称',
    meta: { label: '模型名称', className: 'w-[180px]', viewDetail: true },
  },
  {
    accessorKey: 'displayName',
    header: '展示名',
    meta: { label: '展示名', className: 'w-[140px]' },
    cell: ({ row }) => row.original.displayName || '-',
  },
  {
    accessorKey: 'description',
    header: '描述',
    meta: { label: '描述', className: 'w-[200px]' },
    cell: ({ row }) => row.original.description || '-',
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

function UserModelsPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const formInitialValue = (_type: string, entity?: UserModel) => ({
    userModelId: entity?.userModelId ?? 0,
    name: entity?.name ?? '',
    displayName: entity?.displayName ?? '',
    description: entity?.description ?? '',
    isActive: entity?.isActive ?? true,
  });

  return (
    <Page<UserModel>
      infomation={pageInformation}
      columns={columns}
      service={userModelService}
      options={{ showSelectColumn: false, useRefetchDetail: true }}
      formInitialValue={formInitialValue}
      renderViewDetail={(entity) => <UserModelDetail entity={entity} />}
      renderViewForm={(form, _entity, _formType) => (
        <div className="grid grid-cols-12 gap-4">
          <FormFieldInput className="col-span-6" form={form} name="name" title="模型名称" required placeholder="例如: gpt-4, claude-3" />
          <FormFieldInput className="col-span-6" form={form} name="displayName" title="展示名" placeholder="用户友好的显示名称" />
          <FormFieldTextarea className="col-span-12" form={form} name="description" title="描述" placeholder="模型描述信息" rows={2} />
          <FormFieldSwitch className="col-span-12" form={form} name="isActive" title="启用" switchLabel="启用此用户端模型" />
        </div>
      )}
    />
  );
}

function UserModelDetail({ entity }: { entity: UserModel }) {
  const queryClient = useQueryClient();

  const { data: routers = [], isLoading: routersLoading } = useQuery({
    queryKey: ['user-model-routers', entity.userModelId],
    queryFn: async () => {
      const result = await userModelRouterService.search({
        filters: [{ field: 'userModelId', value: entity.userModelId }],
        pagination: { pageIndex: 1, pageSize: 100 },
      });
      return result.dataSet ?? [];
    },
  });

  const { data: providerModelsData } = useQuery({
    queryKey: ['provider-models-list-for-router'],
    queryFn: () => providerModelService.search({ pagination: { pageIndex: 1, pageSize: 1000 } }),
  });

  const providerModelMap = new Map<number, ProviderModel>();
  (providerModelsData?.dataSet ?? []).forEach((m) => providerModelMap.set(m.modelId, m));

  const deleteMutation = useMutation({
    mutationFn: (routerId: number) => userModelRouterService.delete(routerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-model-routers', entity.userModelId] });
      toast.success('删除成功');
    },
  });

  return (
    <div className="flex flex-col gap-4">
      <Descriptions
        title="模型信息"
        labelClassName="w-20"
        items={[
          { label: '模型名称', value: <span className="font-mono text-xs">{entity.name}</span> },
          { label: '展示名', value: entity.displayName || '-' },
          { label: '描述', value: entity.description || '-' },
          {
            label: '状态',
            value: <Badge variant={entity.isActive ? 'default' : 'destructive'}>{entity.isActive ? '启用' : '禁用'}</Badge>,
          },
        ]}
      />

      <Card>
        <CardHeader>
          <CardTitle>路由规则</CardTitle>
        </CardHeader>
        <CardContent>
          {routersLoading ? (
            <Loading size={20} />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>优先级</TableHead>
                  <TableHead>上游模型</TableHead>
                  <TableHead>服务商</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {routers.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-muted-foreground text-center">
                      暂无路由规则
                    </TableCell>
                  </TableRow>
                ) : (
                  routers.map((r) => {
                    const pm = providerModelMap.get(r.providerModelId);
                    return (
                      <TableRow key={r.routerId}>
                        <TableCell>{r.priority}</TableCell>
                        <TableCell className="font-mono text-xs">{pm?.displayName || pm?.name || `#${r.providerModelId}`}</TableCell>
                        <TableCell>{pm?.provider?.title ?? '-'}</TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => deleteMutation.mutate(r.routerId)}
                          >
                            <Trash2 className="size-4 text-destructive" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
