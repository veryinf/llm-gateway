import { createFileRoute, Link } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { useModal } from '@/components/modal';
import { useConfirm } from '@/components/confirm';
import { EasyTooltip } from '@/components/easy-tooltip';
import { userService, type User } from '@/services/user';
import { apiKeyService, type CreateAPIKeyParams } from '@/services/api-key';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { toast } from 'sonner';

export const Route = createFileRoute('/users')({
  component: UsersPage,
});

const roleOptions = [
  { label: '管理员', value: 'admin' },
  { label: '普通用户', value: 'user' },
  { label: '只读', value: 'viewer' },
];

const pageInformation: PageInformation = {
  name: 'users',
  entityName: '用户',
  page: { title: '用户管理', description: '管理系统用户账号和权限' },
  breadcrumbs: [{ title: '管理' }, { title: '用户管理' }],
};

const columns: ColumnDef<User, any>[] = [
  {
    accessorKey: 'username',
    header: '用户名',
    meta: { label: '用户名', viewDetail: true },
  },
  {
    accessorKey: 'name',
    header: '姓名',
    meta: { label: '姓名', className: 'w-20' },
  },
  {
    accessorKey: 'phone',
    header: '手机号',
    meta: { label: '手机号', className: 'w-28' },
  },
  {
    accessorKey: 'department',
    header: '部门',
    meta: { label: '部门', className: 'w-24' },
  },
  {
    accessorKey: 'role',
    header: '角色',
    meta: { label: '角色', className: 'w-[90px]' },
    cell: ({ row }) => {
      const role = row.original.role;
      const variant = role === 'admin' ? 'default' : role === 'viewer' ? 'secondary' : 'outline';
      return <Badge variant={variant}>{roleOptions.find((r) => r.value === role)?.label ?? role}</Badge>;
    },
  },
  {
    accessorKey: 'status',
    header: '状态',
    meta: { label: '状态', className: 'w-20' },
    cell: ({ row }) => {
      const isActive = row.original.status === 'active';
      return (
        <Badge variant={isActive ? 'default' : 'destructive'}>
          {isActive ? '启用' : '禁用'}
        </Badge>
      );
    },
  },
];

function UsersPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const allColumns: ColumnDef<User, any>[] = [
    ...columns,
    {
      id: 'api_keys',
      header: 'API Keys',
      meta: { label: 'API Keys', className: 'w-24' },
      cell: ({ row }) => (
        <EasyTooltip tooltip="点击查看详情">
          <Button variant="ghost" size="sm" asChild>
            <Link to="/app-keys" search={{ user_id: row.original.uid }}>
              {row.original.apiKeyCount}
            </Link>
          </Button>
        </EasyTooltip>
      ),
    },
  ];

  return (
    <Page<User>
      infomation={pageInformation}
      columns={allColumns}
      service={userService}
      options={{ showSelectColumn: false }}
      renderViewDetail={(entity) => <UserDetail entity={entity} />}
      formInitialValue={(_type, entity) => ({
        uid: entity?.uid ?? 0,
        username: entity?.username ?? '',
        password: '',
        name: entity?.name ?? '',
        phone: entity?.phone ?? '',
        department: entity?.department ?? '',
        role: entity?.role ?? 'user',
        status: entity?.status ?? 'active',
        apiKeyCount: entity?.apiKeyCount ?? 0,
      })}
      renderViewAdd={(form) => (
        <div className="grid grid-cols-12 gap-4">
          <FormFieldInput className="col-span-4" form={form} name="username" title="用户名" required placeholder="请输入用户名" />
          <FormFieldInput className="col-span-4" form={form} name="password" title="密码" required placeholder="请输入密码" type="password" />
          <FormFieldSelect
            className="col-span-4"
            form={form}
            name="status"
            title="状态"
            options={[
              { label: '启用', value: 'active' },
              { label: '禁用', value: 'disabled' },
            ]}
          />
          <FormFieldSelect className="col-span-4" form={form} name="role" title="角色" options={roleOptions} />
          <FormFieldInput className="col-span-4" form={form} name="name" title="姓名" placeholder="请输入姓名" />
          <FormFieldInput className="col-span-4" form={form} name="department" title="部门" placeholder="请输入部门" />
          <FormFieldInput className="col-span-4" form={form} name="phone" title="手机号" placeholder="请输入手机号" />
        </div>
      )}
      renderViewUpdate={(form, _entity) => (
        <div className="grid grid-cols-12 gap-4">
          <FormFieldInput className="col-span-4" form={form} name="username" title="用户名" required />
          <FormFieldInput className="col-span-4" form={form} name="password" title="密码" placeholder="留空不修改" type="password" />
          <FormFieldSelect
            className="col-span-4"
            form={form}
            name="status"
            title="状态"
            options={[
              { label: '启用', value: 'active' },
              { label: '禁用', value: 'disabled' },
            ]}
          />
          <FormFieldSelect className="col-span-4" form={form} name="role" title="角色" options={roleOptions} />
          <FormFieldInput className="col-span-4" form={form} name="name" title="姓名" placeholder="请输入姓名" />
          <FormFieldInput className="col-span-4" form={form} name="department" title="部门" placeholder="请输入部门" />
          <FormFieldInput className="col-span-4" form={form} name="phone" title="手机号" placeholder="请输入手机号" />
        </div>
      )}
    />
  );
}

function UserDetail({ entity }: { entity: User; }) {
  const uid = entity.uid;
  const queryClient = useQueryClient();
  const { Modal, modalHandler } = useModal();
  const { Confirm, confirmHandler } = useConfirm();
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyQuota, setNewKeyQuota] = useState('');
  const [newKeyQpm, setNewKeyQpm] = useState('');
  const [createdRawKey, setCreatedRawKey] = useState<string | null>(null);

  const { data: apiKeys = [], isLoading: keysLoading } = useQuery({
    queryKey: ['user-api-keys', uid],
    queryFn: () => apiKeyService.listByUser(uid),
  });

  const deleteMutation = useMutation({
    mutationFn: (keyId: number) => apiKeyService.delete(uid, keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-api-keys', uid] });
      toast.success('API Key 已删除');
    },
  });

  const createMutation = useMutation({
    mutationFn: (params: CreateAPIKeyParams) => apiKeyService.create(uid, params),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['user-api-keys', uid] });
      setCreatedRawKey(data.raw_key);
      setNewKeyName('');
      setNewKeyQuota('');
      setNewKeyQpm('');
      modalHandler.close();
      toast.success('API Key 创建成功');
    },
    onError: () => {
      toast.error('创建失败');
    },
  });

  return (
    <div className="flex flex-col gap-4">
      {/* 用户信息 */}
      <Descriptions
        title="用户信息"
        labelClassName='w-20'
        items={[
          { label: '用户名', value: entity.username },
          { label: '姓名', value: entity.name || '-' },
          { label: '手机号', value: entity.phone || '-' },
          { label: '部门', value: entity.department || '-' },
          {
            label: '角色',
            value: (
              <Badge variant={entity.role === 'admin' ? 'default' : 'outline'}>
                {roleOptions.find((r) => r.value === entity.role)?.label ?? entity.role}
              </Badge>
            ),
          },
          {
            label: '状态',
            value: (
              <Badge variant={entity.status === 'active' ? 'default' : 'destructive'}>
                {entity.status === 'active' ? '启用' : '禁用'}
              </Badge>
            ),
          },
        ]}
      />

      {/* API Keys 卡片 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>API Keys</CardTitle>
          <Button
            size="sm"
            onClick={() => {
              setCreatedRawKey(null);
              modalHandler.open('创建 API Key');
            }}
          >
            新增 Key
          </Button>
        </CardHeader>
        <CardContent>
          {keysLoading ? (
            <Loading size={20} />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>配额</TableHead>
                  <TableHead>已使用</TableHead>
                  <TableHead>QPM</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="w-16">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="text-muted-foreground text-center">
                      暂无 API Key
                    </TableCell>
                  </TableRow>
                ) : (
                  apiKeys.map((key) => (
                    <TableRow key={key.id}>
                      <TableCell>{key.name}</TableCell>
                      <TableCell className="font-mono text-xs break-all">{key.key}</TableCell>
                      <TableCell>{key.quota_limit > 0 ? key.quota_limit.toLocaleString() : '不限'}</TableCell>
                      <TableCell>{key.quota_used.toLocaleString()}</TableCell>
                      <TableCell>{key.rate_limit_qpm || '不限'}</TableCell>
                      <TableCell>
                        <Badge variant={key.is_active ? 'default' : 'destructive'}>
                          {key.is_active ? '启用' : '禁用'}
                        </Badge>
                      </TableCell>
                      <TableCell>{new Date(key.created_at).toLocaleDateString()}</TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive"
                          onClick={() => {
                            confirmHandler.confirmInvoke(
                              '确认删除',
                              async () => {
                                await deleteMutation.mutateAsync(key.id);
                                return true;
                              },
                              `确认要删除 API Key「${key.name}」吗？`,
                              true,
                            );
                          }}
                        >
                          删除
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 创建成功提示 */}
      {createdRawKey && (
        <Card className="border-green-500">
          <CardHeader>
            <CardTitle className="text-green-600">API Key 创建成功</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-2 text-sm">请立即复制保存，此 Key 仅显示一次：</p>
            <code className="bg-muted block rounded p-3 text-sm break-all">{createdRawKey}</code>
            <Button
              variant="outline"
              size="sm"
              className="mt-2"
              onClick={() => {
                navigator.clipboard.writeText(createdRawKey);
                toast.success('已复制到剪贴板');
              }}
            >
              复制
            </Button>
          </CardContent>
        </Card>
      )}

      {/* 创建 Key 弹窗 */}
      <Modal>
        <div className="flex flex-col gap-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">名称</label>
            <input
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              placeholder="例如: production-key"
              className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">配额限制 (0=不限)</label>
            <input
              type="number"
              value={newKeyQuota}
              onChange={(e) => setNewKeyQuota(e.target.value)}
              placeholder="0"
              className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">QPM 限制 (0=不限)</label>
            <input
              type="number"
              value={newKeyQpm}
              onChange={(e) => setNewKeyQpm(e.target.value)}
              placeholder="60"
              className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
            />
          </div>
          <Button
            disabled={!newKeyName || createMutation.isPending}
            onClick={() => {
              createMutation.mutate({
                name: newKeyName,
                quota_limit: Number(newKeyQuota) || 0,
                rate_limit_qpm: Number(newKeyQpm) || 0,
              });
            }}
          >
            {createMutation.isPending ? '创建中...' : '创建'}
          </Button>
        </div>
      </Modal>
      <Confirm />
    </div>
  );
}
