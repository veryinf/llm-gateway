import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Page, type PageInformation, type FormType } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTextarea, usePopupForm, type DefaultFormState, type EasyFormMeta } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { useConfirm } from '@/components/confirm';
import { Pencil, Plus, Trash2 } from 'lucide-react';
import { UI } from '@/lib';
import { userModelService, type UserModel } from '@/services/user-model';
import { userModelRouterService, type UserModelRouter } from '@/services/user-model-router';
import { providerModelService } from '@/services/provider-model';
import { useAllProviders } from '@/services/provider';
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
    meta: { label: '模型名称', className: "w-30", viewDetail: true },
  },
  {
    accessorKey: 'displayName',
    header: '展示名',
    meta: { label: '展示名', className: 'w-50' },
    cell: ({ row }) => row.original.displayName || '-',
  },
  {
    accessorKey: 'description',
    header: '描述',
    meta: { label: '描述' },
    cell: ({ row }) => row.original.description || '-',
  },
  {
    id: 'activeProviderModel',
    header: '服务商模型',
    meta: { label: '服务商模型', className: 'w-48' },
    cell: ({ row }) => {
      const pm = row.original.activeProviderModel;
      if (!pm) return <span className="text-muted-foreground">-</span>;
      return (
        <div className="flex flex-col gap-0.5">
          <span className="font-mono text-xs">{pm.displayName || pm.name}</span>
          {pm.displayName && pm.displayName !== pm.name && (
            <span className="text-xs text-muted-foreground">{pm.name}</span>
          )}
        </div>
      );
    },
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

  const formInitialValue = (formType: FormType, entity?: UserModel) => (formType == 'add' ? { isActive: true } : { ...entity! });

  return (
    <Page<UserModel>
      infomation={pageInformation}
      columns={columns}
      service={userModelService}
      options={{ showSelectColumn: false, useRefetchDetail: true }}
      optionColumn={(column, domRender) => ({ ...column, cell: (res) => domRender(res.row.original) })}
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

function UserModelDetail({ entity }: { entity: UserModel; }) {
  const queryClient = useQueryClient();
  const { Confirm, confirmHandler } = useConfirm();
  const { PopupForm, formHandler } = usePopupForm<UserModelRouter, DefaultFormState>();
  const { allProviders } = useAllProviders();

  const { data: routers = [], isLoading: routersLoading } = useQuery({
    queryKey: ['user-model-routers', entity.userModelId],
    queryFn: async () => {
      const res = await userModelRouterService.search({
        filters: [{ field: 'userModelId', value: entity.userModelId }],
        pagination: { index: 1, size: 100 },
      });
      return res.dataSet ?? [];
    },
  });

  const { data: allProviderModels = [] } = useQuery({
    queryKey: ['provider-models-list-for-router'],
    queryFn: async () => {
      const res = await providerModelService.search({ pagination: { index: 1, size: 10000 } });
      return res.dataSet ?? [];
    },

  });

  const handleDelete = (routerId: number) => {
    confirmHandler.confirmInvoke(
      '确认删除',
      async () => {
        const ok = await UI.tips(userModelRouterService.delete(routerId), '删除成功');
        if (ok) {
          queryClient.invalidateQueries({ queryKey: ['user-model-routers', entity.userModelId] });
        }
        return ok;
      },
      '确认要删除此路由规则吗？',
      true,
    );
  };


  const handleSubmit = async (values: UserModelRouter, meta: EasyFormMeta<UserModelRouter, DefaultFormState>) => {
    const params: UserModelRouter = { providerModelId: Number(values.providerModelId), priority: Number(values.priority), isActive: values.isActive } as any;
    if (meta.state?.action === 'add') {
      params.userModelId = entity.userModelId;
      const ok = await UI.tips(userModelRouterService.add(params), '新增路由成功');
      if (ok) {
        queryClient.invalidateQueries({ queryKey: ['user-model-routers', entity.userModelId] });
      }
      return ok;
    } else {
      const router = meta.original;
      if (!router) return false;
      const ok = await UI.tips(userModelRouterService.update(router.routerId, params), '保存成功',);
      if (ok) {
        queryClient.invalidateQueries({ queryKey: ['user-model-routers', entity.userModelId] });
      }
      return ok;
    }
  };

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
          <div className="flex items-center justify-between">
            <CardTitle>路由规则</CardTitle>
            <Button variant="outline" size="sm" onClick={() => formHandler.open('新增路由规则', undefined, { action: 'add' })}>
              <Plus className="size-4" /> 新增
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {routersLoading ? (
            <Loading size={20} />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-center w-20">优先级</TableHead>
                  <TableHead>服务商模型</TableHead>
                  <TableHead>服务商</TableHead>
                  <TableHead className="w-20">状态</TableHead>
                  <TableHead className="w-28" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {routers.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-muted-foreground text-center">
                      暂无路由规则
                    </TableCell>
                  </TableRow>
                ) : (
                  routers.map((r) => {
                    const pm = allProviderModels.find(m => m.modelId == r.providerModelId);
                    const provider = pm ? allProviders.find(p => p.providerId == pm.providerId) : undefined;
                    return (
                      <TableRow key={r.routerId}>
                        <TableCell className="text-center">{r.priority}</TableCell>
                        <TableCell className="font-mono text-xs">{pm?.displayName || pm?.name || `#${r.providerModelId}`}</TableCell>
                        <TableCell>{provider?.title ?? '-'}</TableCell>
                        <TableCell>
                          <Badge variant={r.isActive ? 'default' : 'destructive'}>
                            {r.isActive ? '启用' : '禁用'}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => formHandler.open('编辑路由规则', { ...r, providerModelId: String(r.providerModelId) } as any, { action: 'edit' })}
                            >
                              <Pencil className="size-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(r.routerId)}
                            >
                              <Trash2 className="size-4 text-destructive" />
                            </Button>
                          </div>
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

      <PopupForm onSubmit={handleSubmit} type='dialog'>
        {(form, _data, state) => (
          <div className="grid grid-cols-12 gap-4">
            <FormFieldSelect className="col-span-12" form={form} name="providerModelId" title="上游模型" required placeholder="选择 ProviderModel" options={allProviders.map(p => {
              return {
                label: p.title, options: allProviderModels.filter(m => m.providerId == p.providerId).map(m => {
                  return {
                    label: m.displayName ? `${m.displayName}(${m.name})` : m.name,
                    value: m.modelId
                  };
                })
              };
            })} />
            <FormFieldInput className="col-span-12" form={form} name="priority" title="优先级" type="number" required description="数值越小越靠前" />
            {state?.action === 'edit' && (
              <FormFieldSwitch
                className="col-span-12"
                form={form}
                name="isActive"
                title="启用"
                switchLabel="启用此路由规则"
              />
            )}
          </div>
        )}
      </PopupForm>
      <Confirm />
    </div>
  );
}
