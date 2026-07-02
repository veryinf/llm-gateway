import { useCallback, useEffect, useMemo, useRef, useState, type PropsWithChildren, type ReactNode } from 'react';
import { Outlet } from '@tanstack/react-router';
import { useModal } from './modal';
import { PageHeader, type PageHeaderProps } from './page-header';
import { Button } from './ui/button';
import { type ColumnDef } from '@tanstack/react-table';
import type { API } from '@/typings';
import { FullTable, type FullTableState } from './data-table/full-table';
import { Checkbox } from './ui/checkbox';
import { ReceiptText, SquarePen, Trash2 } from 'lucide-react';
import { useConfirm } from './confirm';
import { usePopupForm } from './form';
import { pick } from 'lodash-es';
import { UI } from '@/lib';
import type { EasyFormMeta, EasyFormApi } from './form/utils';
import { useQuery } from '@tanstack/react-query';
import { Loading } from './loader';
import { useBreadcrumb, type BreadcrumbItem } from '@/hooks/use-breadcrumb';
import { EasyButton } from './easy-button';

export type PageInformation = {
  /** 页面唯一标识符（英文） */
  name: string;
  /** 实体名称（中文），用于显示在界面上 */
  entityName: string;
  /** 页面信息配置 */
  page: PageHeaderProps;
  breadcrumbs?: BreadcrumbItem[];
};

export type FormType = 'add' | 'update';

type PageProps<TEntity> = PropsWithChildren<{
  /** 页面信息配置 */
  infomation: PageInformation;
  ready?: boolean;
  /** 表格列定义 */
  columns: ColumnDef<TEntity, {}>[];
  /** 数据服务，提供增删改查等操作 */
  service: API.Service<TEntity>;
  optionColumn?: (defaultDef: ColumnDef<TEntity, {}>, domRender: (entity: TEntity) => ReactNode) => ColumnDef<TEntity, {}>;
  /** 实体转换函数，用于转换从服务获取的实体数据 */
  entityTransfer?: (entity: TEntity) => TEntity;
  /** 添加表单初始值函数，用于初始化添加表单的值 */
  formInitialValue?: (formType: FormType, entity?: TEntity) => Promise<Partial<TEntity>> | Partial<TEntity>;
  /** 添加表单验证器，验证添加操作的数据是否有效 */
  formAddValidator?: (entity: TEntity) => Promise<boolean> | boolean;
  /** 更新表单验证器，验证更新操作的数据是否有效 */
  formUpdateValidator?: (entity: TEntity, original: TEntity) => Promise<boolean> | boolean;
  /** 点击详情的回调 */
  onViewDetail?: (entity: TEntity) => void | Promise<void>;
  /** 渲染详情视图的函数 */
  renderViewDetail?: (entity: TEntity) => ReactNode;
  /** 渲染通用表单视图的函数 */
  renderViewForm?: (form: EasyFormApi<TEntity>, entity: TEntity | undefined, formType: FormType) => ReactNode;
  /** 渲染添加表单视图的函数 */
  renderViewAdd?: (form: EasyFormApi<TEntity>) => ReactNode;
  /** 渲染更新表单视图的函数 */
  renderViewUpdate?: (form: EasyFormApi<TEntity>, entity: TEntity) => ReactNode;
  /** 可选配置项 */
  options?: {
    /** 是否显示选择列 */
    showSelectColumn?: boolean;
    /** 查看详情时是否重新获取数据 */
    useRefetchDetail?: boolean;
    /** 更新数据时是否重新获取数据 */
    useRefetchUpdate?: boolean;
  };
}>;

type FormState = {
  type: FormType;
};

export function Page<TEntity>(props: PageProps<TEntity>) {
  const { setBreadcrumbs } = useBreadcrumb();
  const { modalHandler, Modal, meta } = useModal<{ entity?: TEntity; }>();
  const { PopupForm, formHandler } = usePopupForm<TEntity, FormState>();
  const { confirmHandler, Confirm } = useConfirm();
  const tableState = useRef<FullTableState | undefined>(undefined);
  const { infomation, options = {} } = props;
  const { showSelectColumn = true, useRefetchDetail = false, useRefetchUpdate = false } = options;
  const {
    data: source,
    refetch,
    isLoading,
  } = useQuery({
    //
    queryKey: ['full-page', infomation.name],
    queryFn: () => {
      return props.service.search(tableState.current ?? {});
    },
    select: (x) => {
      const list = props.entityTransfer ? (x.dataSet ?? []).map((e: TEntity) => props.entityTransfer!(e)) : (x.dataSet ?? []);
      return { list, total: x.total };
    },
    enabled: !!props.service && !!tableState.current,
  });

  useEffect(() => {
    setBreadcrumbs(infomation.breadcrumbs ?? []);
  }, []);

  const handleDelete = useCallback(async (entity: TEntity) => {
    confirmHandler.confirmInvoke(
      '确认删除',
      async () => {
        const entityId = props.service.primaryKey(entity);
        const deleteTask = props.service.delete(entityId);
        const isDone = await UI.tips(deleteTask, `成功删除此${infomation.entityName}`);
        if (isDone) {
          refetch();
        }
        return isDone;
      },
      `确认要删除此${infomation.entityName}吗？`,
      true,
    );
  }, []);

  const handleSubmit = useCallback(async (value: TEntity, meta: EasyFormMeta<TEntity, FormState>) => {
    let isDone = false;
    if (meta.state?.type === 'update') {
      if (props.formUpdateValidator) {
        const isValid = await props.formUpdateValidator(value, meta.original!);
        if (!isValid) {
          return false;
        }
      }
      const entityId = props.service.primaryKey(meta.original!);
      const updateTask = props.service.update(entityId, value);
      isDone = await UI.tips(updateTask, `成功更新${infomation.entityName}`);
    } else {
      if (props.formAddValidator) {
        const isValid = await props.formAddValidator(value);
        if (!isValid) {
          return false;
        }
      }
      const addTask = props.service.add(value);
      isDone = await UI.tips(addTask, `成功添加${infomation.entityName}`);
    }
    if (isDone) {
      refetch();
    }
    return isDone;
  }, []);

  const columns: ColumnDef<TEntity, {}>[] = useMemo(() => {
    const c = [
      ...props.columns.map((column) => {
        if (typeof column.header === 'string' || !column.meta?.label) {
          if (!column.meta) {
            column.meta = {};
          }
          column.meta.label = column.header as string;
        }
        if ((props.renderViewDetail || props.onViewDetail) && !column.cell && column.meta.viewDetail) {
          column.cell = (cell) => {
            const entity = cell.row.original;
            return (
              <a
                href="#"
                className="text-primary hover:underline"
                onClick={(e) => {
                  e.preventDefault();
                  if (props.renderViewDetail) {
                    modalHandler.open(`查看${infomation.entityName}详情`, '', { entity });
                  } else {
                    props.onViewDetail!(entity);
                  }
                }}
              >
                {(entity as any)[(column as any).accessorKey]}
              </a>
            );
          };
        }
        return column;
      }),
    ];
    if (showSelectColumn) {
      c.unshift({
        id: 'select',
        header: ({ table }) => <Checkbox checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && 'indeterminate')} onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)} aria-label="Select all" className="translate-y-[2px]" />,
        cell: ({ row }) => <Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} aria-label="Select row" className="translate-y-0.5" />,
        enableSorting: false,
        enableHiding: false,
        meta: { thClassName: 'w-8' },
      });
    }
    if (props.optionColumn) {
      const defaultDef = {
        header: '操作',
        accessorKey: 'actions',
        enableSorting: false,
        enableHiding: false,
        meta: { className: 'w-14 text-center' },
      };
      const domRender = (entity: TEntity) => <>
        <div className="flex">
          <EasyButton tooltip="查看详情" variant="link" size="icon-sm"
            onClick={() => {
              if (props.renderViewDetail) {
                modalHandler.open(`查看${infomation.entityName}详情`, '', { entity });
              } else {
                props.onViewDetail!(entity);
              }
            }}
          >
            <ReceiptText size={16} />
          </EasyButton>
          <EasyButton tooltip="编辑" variant="link" size="icon-sm"
            onClick={async () => {
              let initValue = entity;
              if (props.formInitialValue) {
                initValue = (await props.formInitialValue('update', initValue)) as any;
              }
              formHandler.open(`编辑${infomation.entityName} - ${props.service.title(entity)}`, initValue, { type: 'update' });
            }}
          >
            <SquarePen size={16} />
          </EasyButton>
          <EasyButton tooltip="删除" variant="link" size="icon-sm" onClick={() => handleDelete(entity)}>
            <Trash2 size={16} className="text-destructive" />
          </EasyButton>
        </div>
      </>;

      c.push(props.optionColumn(defaultDef, domRender));
    }
    return c;
  }, [props.ready !== false]);

  function handleRefresh(state: FullTableState): void {
    tableState.current = state;
    refetch();
  }

  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 px-4">
          <PageHeader
            actions={
              <>
                <Button
                  onClick={async () => {
                    let initValue = {} as any;
                    if (props.formInitialValue) {
                      initValue = await props.formInitialValue('add');
                      if (!initValue) {
                        initValue = {};
                      }
                    }
                    formHandler.open(`添加${infomation.entityName}`, initValue, { type: 'add' });
                  }}
                >
                  添加{infomation.entityName}
                </Button>
              </>
            }
            {...infomation.page}
          />
          {props.ready === false ? <Loading /> : <FullTable columns={columns} data={source?.list || []} total={source?.total || 0} loading={isLoading} onRefresh={handleRefresh} />}
        </div>
      </div>
      <Modal>
        <ViewDetail service={props.service} entity={meta?.entity} render={props.renderViewDetail} useRefetch={useRefetchDetail} />
      </Modal>
      <PopupForm onSubmit={handleSubmit}>{(form, formData, state) => <ViewForm service={props.service} entity={formData} form={form} formState={state!} render={pick(props, ['renderViewForm', 'renderViewAdd', 'renderViewUpdate'])} useRefetch={useRefetchUpdate} />}</PopupForm>
      <Confirm />
      <Outlet />
    </div>
  );
}

//调用加载最新数据
function ViewDetail<TEntity>(props: {
  /** 数据服务，提供增删改查等操作 */
  service: API.Service<TEntity>;
  /** 要显示详情的实体数据 */
  entity?: TEntity;
  /** 渲染详情视图的函数 */
  render: PageProps<TEntity>['renderViewDetail'];
  /** 是否重新获取数据 */
  useRefetch: boolean;
}) {
  const [entity, setEntity] = useState<TEntity>();

  useEffect(() => {
    loadEntity();
  }, [props.entity]);

  async function loadEntity() {
    if (props.entity) {
      if (props.useRefetch) {
        const { data } = await props.service.fetch(props.service.primaryKey(props.entity));
        setEntity(data);
      } else {
        setEntity(props.entity);
      }
    }
  }

  if (!props.render || !props.entity) {
    return null;
  }

  if (!entity) {
    return (
      <div>
        <Loading size={20} />
      </div>
    );
  }

  return props.render(props.entity);
}

function ViewForm<TEntity>(props: {
  /** 数据服务，提供增删改查等操作 */
  service: API.Service<TEntity>;
  /** 表单 API，用于操作表单 */
  form: EasyFormApi<TEntity>;
  /** 实体数据，用于表单初始化 */
  entity?: TEntity;
  /** 表单状态，标识是添加还是更新操作 */
  formState: FormState;
  /** 渲染表单视图的函数集合 */
  render: Pick<PageProps<TEntity>, 'renderViewForm' | 'renderViewAdd' | 'renderViewUpdate'>;
  /** 是否重新获取数据 */
  useRefetch: boolean;
}) {
  const [entity, setEntity] = useState<TEntity>();

  useEffect(() => {
    loadEntity();
  }, [props.entity]);

  async function loadEntity() {
    if (props.entity && props.formState.type === 'update') {
      if (props.useRefetch) {
        const { data } = await props.service.fetch(props.service.primaryKey(props.entity));
        setEntity(data);
      } else {
        setEntity(props.entity);
      }
    }
  }

  if (!props.render) {
    return null;
  }

  if (props.formState.type === 'update' && !entity) {
    return (
      <div>
        <Loading size={20} />
      </div>
    );
  }
  if (props.formState.type === 'add' && props.render.renderViewAdd) {
    return props.render.renderViewAdd(props.form);
  }
  if (props.formState.type === 'update' && props.render.renderViewUpdate) {
    return props.render.renderViewUpdate(props.form, entity!);
  }
  if (props.render.renderViewForm) {
    return props.render.renderViewForm(props.form, entity, props.formState.type);
  }
  return null;
}
