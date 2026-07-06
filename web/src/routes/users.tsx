import { createFileRoute, Link } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { EasyTooltip } from '@/components/easy-tooltip';
import { userService, type User } from '@/services/user';
import { userKeyService } from '@/services/api-key';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { CopyButton } from '@/components/easy-button';

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
            <Link to="/user-keys" search={{ uid: [row.original.uid] }}>
              {row.original.userKeyCount}
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
      optionColumn={(column, domRender) => ({ ...column, cell: (res) => domRender(res.row.original) })}
      renderViewDetail={(entity) => <UserDetail entity={entity} />}
      formInitialValue={(formType, entity) => (formType == 'add' ? {
        uid: 0,
        username: '',
        password: '',
        name: '',
        phone: '',
        department: '',
        role: 'user',
        status: 'active',
      } : {
        ...entity!,
        password: '',
      })}
      renderViewForm={(form, _entity) => (
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

  const { data: apiKeys = [], isLoading: keysLoading } = useQuery({
    queryKey: ['user-api-keys', uid],
    queryFn: async () => {
      const result = await userKeyService.search({ filters: [{ field: 'uid', value: uid }] });
      return result.dataSet ?? [];
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
        <CardHeader>
          <CardTitle>API Keys</CardTitle>
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
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3} className="text-muted-foreground text-center">
                      暂无 API Key
                    </TableCell>
                  </TableRow>
                ) : (
                  apiKeys.map((key) => (
                    <TableRow key={key.keyId}>
                      <TableCell>{key.title}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <span className="font-mono text-xs break-all">{key.key}</span>
                          <CopyButton text={key.key} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={key.isActive ? 'default' : 'destructive'}>
                          {key.isActive ? '启用' : '禁用'}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
