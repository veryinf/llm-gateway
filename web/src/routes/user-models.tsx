import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTextarea } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { useModal } from '@/components/modal';
import { useConfirm } from '@/components/confirm';
import { Plus, Trash2 } from 'lucide-react';
import { UI } from '@/lib';
import { userModelService, type UserModel } from '@/services/user-model';
import { userModelRouterService } from '@/services/user-model-router';
import { providerModelService, type ProviderModel } from '@/services/provider-model';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

const addRouterSchema = z.object({
  providerModelId: z.string().min(1, '请选择上游模型'),
  priority: z.number({ message: '必填项' }).int('必须是整数').min(0, '不能为负数'),
});

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
    meta: { label: '模型名称', viewDetail: true },
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
  const { Modal, modalHandler } = useModal();
  const { Confirm, confirmHandler } = useConfirm();

  const { data: routers = [], isLoading: routersLoading } = useQuery({
    queryKey: ['user-model-routers', entity.userModelId],
    queryFn: async () => {
      const result = await userModelRouterService.search({
        filters: [{ field: 'userModelId', value: entity.userModelId }],
        pagination: { index: 1, size: 100 },
      });
      return result.dataSet ?? [];
    },
  });

  const { data: providerModelsData } = useQuery({
    queryKey: ['provider-models-list-for-router'],
    queryFn: () => providerModelService.search({ pagination: { index: 1, size: 1000 } }),
  });

  const providerModelMap = new Map<number, ProviderModel>();
  (providerModelsData?.dataSet ?? []).forEach((m) => providerModelMap.set(m.modelId, m));

  const providerModelOptions = useMemo(
    () => (providerModelsData?.dataSet ?? []).map((m) => ({
      label: m.displayName
        ? `${m.displayName} (${m.provider?.title ?? '-'})`
        : `${m.name} (${m.provider?.title ?? '-'})`,
      value: m.modelId,
    })),
    [providerModelsData],
  );

  const defaultPriority = useMemo(() => {
    if (routers.length === 0) return 0;
    return Math.max(...routers.map((r) => r.priority)) + 1;
  }, [routers]);

  const deleteMutation = useMutation({
    mutationFn: (routerId: number) => userModelRouterService.delete(routerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-model-routers', entity.userModelId] });
    },
  });

  const handleDelete = (routerId: number) => {
    confirmHandler.confirmInvoke(
      '确认删除',
      async () => {
        const ok = await UI.tips(deleteMutation.mutateAsync(routerId), '删除成功');
        if (ok) {
          return ok;
        }
        return false;
      },
      '确认要删除此路由规则吗？',
      true,
    );
  };

  const addForm = useForm({
    defaultValues: {
      providerModelId: '',
      priority: 0,
    },
    validators: {
      onChange: addRouterSchema,
    },
    onSubmit: async ({ value }) => {
      const ok = await UI.tips(
        userModelRouterService.add({
          routerId: 0,
          userModelId: entity.userModelId,
          providerModelId: Number(value.providerModelId),
          priority: value.priority,
        }),
        '新增路由成功',
      );
      if (ok) {
        modalHandler.close();
        queryClient.invalidateQueries({ queryKey: ['user-model-routers', entity.userModelId] });
      }
    },
  });

  const handleOpenAdd = () => {
    addForm.reset({
      providerModelId: '',
      priority: defaultPriority,
    });
    modalHandler.show({
      title: '新增路由规则',
      description: `为「${entity.displayName || entity.name}」配置新的上游路由`,
      actions: (
        <>
          <Button className="h-8 px-6" variant="secondary" onClick={() => modalHandler.close()}>
            取消
          </Button>
          <addForm.Subscribe>
            {(state) => (
              <Button className="h-8 px-6" onClick={() => addForm.handleSubmit()} disabled={state.isSubmitting}>
                {state.isSubmitting ? '提交中...' : '确认'}
              </Button>
            )}
          </addForm.Subscribe>
        </>
      ),
    });
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
            <Button variant="outline" size="sm" onClick={handleOpenAdd}>
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
                            onClick={() => handleDelete(r.routerId)}
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

      <Modal type="dialog">
        <div className="grid grid-cols-12 gap-4">
          <FormFieldSelect
            className="col-span-12"
            form={addForm}
            name="providerModelId"
            title="上游模型"
            required
            placeholder="选择 ProviderModel"
            options={providerModelOptions}
          />
          <FormFieldInput
            className="col-span-12"
            form={addForm}
            name="priority"
            title="优先级"
            type="number"
            required
            description="数值越小越靠前"
          />
        </div>
      </Modal>

      <Confirm />
    </div>
  );
}
