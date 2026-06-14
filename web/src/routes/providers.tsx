import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo, useState } from 'react';
import { useStore } from '@tanstack/react-form';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Page, type PageInformation } from '@/components/full-page';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { providerService, fetchProviderModels, batchImportProviderModels, type Provider } from '@/services/provider';
import { modelService, type Model } from '@/services/model';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useModal } from '@/components/modal';
import { useConfirm } from '@/components/confirm';
import { toast } from 'sonner';
import { request } from '@/lib';
import type { API } from '@/typings';

export const Route = createFileRoute('/providers')({
  component: ProvidersPage,
});

interface UpstreamModel {
  name: string;
  api_type: 'openai' | 'anthropic';
  is_chat: boolean;
}

interface EditableModel extends UpstreamModel {
  id?: number;
}

const pageInformation: PageInformation = {
  name: 'providers',
  entityName: 'Provider',
  page: { title: 'Provider 管理', description: '管理 LLM 服务商配置' },
  breadcrumbs: [{ title: '上游' }, { title: 'LLM 服务商' }],
};

function formatTokens(value: number) {
  if (!value) return '-';
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}K`;
  return String(value);
}

function formatPrice(value: number) {
  if (!value) return '-';
  return `$${value.toFixed(2)}`;
}

const columns: ColumnDef<Provider, any>[] = [
  {
    accessorKey: 'name',
    header: '名称',
    meta: { label: '名称', className: 'w-[160px]', viewDetail: true },
  },
  {
    accessorKey: 'base_url',
    header: 'Base URL',
    meta: { label: 'Base URL', className: 'w-[240px]' },
    cell: ({ row }) => <span className="font-mono text-xs">{row.original.base_url}</span>,
  },
  {
    accessorKey: 'protocol',
    header: '协议',
    meta: { label: '协议', className: 'w-[120px]' },
    cell: ({ row }) => (
      <div className="flex gap-1 flex-wrap">
        {row.original.support_openai && <Badge variant="secondary">OpenAI</Badge>}
        {row.original.support_anthropic && <Badge variant="secondary">Anthropic</Badge>}
      </div>
    ),
  },
  {
    accessorKey: 'preferred_api',
    header: '优先接口',
    meta: { label: '优先接口', className: 'w-[80px]' },
    cell: ({ row }) => (
      <Badge variant="outline">{row.original.preferred_api === 'openai' ? 'OpenAI' : 'Anthropic'}</Badge>
    ),
  },
  {
    accessorKey: 'model_count',
    header: '模型数',
    meta: { label: '模型数', className: 'w-[80px]' },
    cell: ({ row }) => {
      const count = row.original.model_count ?? 0;
      return (
        <Badge variant={count > 0 ? 'default' : 'secondary'}>
          {count}
        </Badge>
      );
    },
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
];

// ---------- Shared ModelFieldList component ----------

function ModelFieldList({
  modelList,
  setModelList,
  preferredApi,
  baseUrl,
  apiKey,
}: {
  modelList: EditableModel[];
  setModelList: React.Dispatch<React.SetStateAction<EditableModel[]>>;
  preferredApi: string;
  baseUrl: string;
  apiKey: string;
}) {
  const [isFetching, setIsFetching] = useState(false);

  async function handleFetchModels() {
    if (!baseUrl) {
      toast.error('请先填写 BaseURL 根地址');
      return;
    }
    setIsFetching(true);
    try {
      const fetched = await fetchProviderModels(baseUrl, apiKey, preferredApi);
      setModelList((prev) => {
        const existingNames = new Set(prev.map((m) => m.name));
        const newModels: EditableModel[] = fetched
          .filter((m) => !existingNames.has(m.id))
          .map((m) => ({
            name: m.id,
            api_type: (preferredApi as 'openai' | 'anthropic') || 'openai',
            is_chat: true,
          }));
        return [...prev, ...newModels];
      });
      toast.success(`获取到 ${fetched.length} 个模型`);
    } catch {
      toast.error('获取模型列表失败');
    } finally {
      setIsFetching(false);
    }
  }

  function addEmptyModel() {
    setModelList((prev) => [
      ...prev,
      { name: '', api_type: (preferredApi as 'openai' | 'anthropic') || 'openai', is_chat: true },
    ]);
  }

  function removeModel(index: number) {
    setModelList((prev) => prev.filter((_, i) => i !== index));
  }

  function updateModel(index: number, field: keyof EditableModel, value: any) {
    setModelList((prev) => prev.map((m, i) => (i === index ? { ...m, [field]: value } : m)));
  }

  return (
    <>
      <div className="text-sm font-medium text-muted-foreground border-t pt-4">
        上游模型 ({modelList.length})
      </div>
      <div className="flex gap-2">
        <Button type="button" size="sm" variant="outline" onClick={handleFetchModels} disabled={isFetching}>
          {isFetching ? '获取中...' : '获取模型'}
        </Button>
        <Button type="button" size="sm" variant="outline" onClick={addEmptyModel}>
          添加
        </Button>
      </div>
      {modelList.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>模型名称</TableHead>
              <TableHead>协议类型</TableHead>
              <TableHead>Chat</TableHead>
              <TableHead className="w-16"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {modelList.map((m, idx) => (
              <TableRow key={idx}>
                <TableCell>
                  <input
                    type="text"
                    value={m.name}
                    onChange={(e) => updateModel(idx, 'name', e.target.value)}
                    placeholder="例如: gpt-4o"
                    className="border-input bg-background flex h-8 w-full rounded-md border px-2 py-1 text-sm"
                  />
                </TableCell>
                <TableCell>
                  <select
                    value={m.api_type}
                    onChange={(e) => updateModel(idx, 'api_type', e.target.value)}
                    className="border-input bg-background flex h-8 rounded-md border px-2 py-1 text-sm"
                  >
                    <option value="openai">OpenAI</option>
                    <option value="anthropic">Anthropic</option>
                  </select>
                </TableCell>
                <TableCell>
                  <input
                    type="checkbox"
                    checked={m.is_chat}
                    onChange={(e) => updateModel(idx, 'is_chat', e.target.checked)}
                    className="h-4 w-4"
                  />
                </TableCell>
                <TableCell>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="text-destructive"
                    onClick={() => removeModel(idx)}
                  >
                    删除
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </>
  );
}

// ---------- AddForm ----------

function AddForm({ form }: { form: any }) {
  const [modelList, setModelList] = useState<EditableModel[]>([]);

  const v = useStore(form.store ?? form, (s: any) => s.values ?? s) as any;
  const supportOpenai = v?.support_openai ?? false;
  const supportAnthropic = v?.support_anthropic ?? false;
  const baseUrl = v?.base_url ?? '';

  const preferredApiOptions = useMemo(() => {
    const opts: { label: string; value: string }[] = [];
    if (supportOpenai) opts.push({ label: 'OpenAI', value: 'openai' });
    if (supportAnthropic) opts.push({ label: 'Anthropic', value: 'anthropic' });
    return opts;
  }, [supportOpenai, supportAnthropic]);

  // Sync modelList to form's models field
  useEffect(() => {
    form.setFieldValue('models', modelList);
  }, [modelList, form]);

  // Auto-adjust preferred_api when supported protocols change
  useEffect(() => {
    if (preferredApiOptions.length > 0 && !preferredApiOptions.find((o) => o.value === v?.preferred_api)) {
      form.setFieldValue('preferred_api', preferredApiOptions[0].value);
    }
  }, [preferredApiOptions, v?.preferred_api, form]);

  return (
    <div className="flex flex-col gap-4 max-h-[70vh] overflow-y-auto pr-2">
      <div className="text-sm font-medium text-muted-foreground">基础信息</div>
      <FormFieldInput form={form} name="name" title="名称" required placeholder="请输入 Provider 名称" />
      <FormFieldInput form={form} name="base_url" title="BaseURL 根地址" required placeholder="https://api.example.com" />
      <FormFieldInput form={form} name="api_key" title="API Key" placeholder="sk-..." type="password" />

      <div className="grid grid-cols-4 gap-4">
        <FormFieldSwitch form={form} name="support_openai" title="支持 OpenAI" switchLabel="支持 OpenAI" />
        <FormFieldSwitch form={form} name="support_anthropic" title="支持 Anthropic" switchLabel="支持 Anthropic" />
        <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用" />
        {preferredApiOptions.length > 0 && (
          <FormFieldSelect form={form} name="preferred_api" title="优先接口" options={preferredApiOptions} />
        )}
      </div>

      {supportOpenai && (
        <FormFieldInput
          form={form}
          name="openai_base_url"
          title="OpenAI BaseURL"
          placeholder={baseUrl ? `${baseUrl}/v1` : 'https://api.example.com/v1'}
        />
      )}

      {supportAnthropic && (
        <FormFieldInput
          form={form}
          name="anthropic_base_url"
          title="Anthropic BaseURL"
          placeholder={baseUrl ? `${baseUrl}/anthropic/v1` : 'https://api.example.com/anthropic/v1'}
        />
      )}

      <ModelFieldList
        modelList={modelList}
        setModelList={setModelList}
        preferredApi={v?.preferred_api || 'openai'}
        baseUrl={baseUrl}
        apiKey={v?.api_key || ''}
      />
    </div>
  );
}

// ---------- EditForm ----------

function EditForm({ form, entity }: { form: any; entity: Provider }) {
  const pid = entity.id;
  const [modelList, setModelList] = useState<EditableModel[]>([]);

  const v = useStore(form.store ?? form, (s: any) => s.values ?? s) as any;
  const supportOpenai = v?.support_openai ?? false;
  const supportAnthropic = v?.support_anthropic ?? false;
  const baseUrl = v?.base_url ?? '';

  const preferredApiOptions = useMemo(() => {
    const opts: { label: string; value: string }[] = [];
    if (supportOpenai) opts.push({ label: 'OpenAI', value: 'openai' });
    if (supportAnthropic) opts.push({ label: 'Anthropic', value: 'anthropic' });
    return opts;
  }, [supportOpenai, supportAnthropic]);

  // Load existing models for this provider
  useQuery({
    queryKey: ['provider-models', pid],
    queryFn: async () => {
      const res = await request.get<API.DataSet<Model>>(`/admin/models?provider_id=${pid}`);
      const models = (res.data.dataSet ?? []) as Model[];
      setModelList(
        models.map((m) => ({
          id: m.id,
          name: m.name,
          api_type: m.api_type || 'openai',
          is_chat: m.is_chat ?? true,
        })),
      );
      return models;
    },
  });

  // Sync modelList to form's models field
  useEffect(() => {
    form.setFieldValue('models', modelList);
  }, [modelList, form]);

  // Auto-adjust preferred_api when supported protocols change
  useEffect(() => {
    if (preferredApiOptions.length > 0 && !preferredApiOptions.find((o) => o.value === v?.preferred_api)) {
      form.setFieldValue('preferred_api', preferredApiOptions[0].value);
    }
  }, [preferredApiOptions, v?.preferred_api, form]);

  return (
    <div className="flex flex-col gap-4 max-h-[70vh] overflow-y-auto pr-2">
      <div className="text-sm font-medium text-muted-foreground">基础信息</div>
      <FormFieldInput form={form} name="name" title="名称" required />
      <FormFieldInput form={form} name="base_url" title="Base URL" required placeholder="https://api.example.com" />
      <FormFieldInput form={form} name="api_key" title="API Key" placeholder="留空不修改" type="password" />

      <div className="grid grid-cols-4 gap-4">
        <FormFieldSwitch form={form} name="support_openai" title="支持 OpenAI" switchLabel="支持 OpenAI" />
        <FormFieldSwitch form={form} name="support_anthropic" title="支持 Anthropic" switchLabel="支持 Anthropic" />
        <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用" />
        {preferredApiOptions.length > 0 && (
          <FormFieldSelect form={form} name="preferred_api" title="优先接口" options={preferredApiOptions} />
        )}
      </div>

      {supportOpenai && (
        <FormFieldInput
          form={form}
          name="openai_base_url"
          title="OpenAI BaseURL"
          placeholder={baseUrl ? `${baseUrl}/v1` : 'https://api.example.com/v1'}
        />
      )}

      {supportAnthropic && (
        <FormFieldInput
          form={form}
          name="anthropic_base_url"
          title="Anthropic BaseURL"
          placeholder={baseUrl ? `${baseUrl}/anthropic/v1` : 'https://api.example.com/anthropic/v1'}
        />
      )}

      <ModelFieldList
        modelList={modelList}
        setModelList={setModelList}
        preferredApi={v?.preferred_api || 'openai'}
        baseUrl={baseUrl}
        apiKey={v?.api_key || ''}
      />
    </div>
  );
}

// ---------- ProvidersPage ----------

function ProvidersPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  return (
    <Page<Provider>
      infomation={pageInformation}
      columns={columns}
      service={providerService}
      options={{ showSelectColumn: false }}
      renderViewDetail={(entity) => <ProviderDetail entity={entity} />}
      formInitialValue={(_type, entity) => ({
        id: 0,
        name: entity?.name ?? '',
        base_url: entity?.base_url ?? '',
        api_key: '',
        support_openai: entity?.support_openai ?? true,
        openai_base_url: entity?.openai_base_url ?? '',
        support_anthropic: entity?.support_anthropic ?? true,
        anthropic_base_url: entity?.anthropic_base_url ?? '',
        preferred_api: entity?.preferred_api ?? 'openai',
        is_active: entity?.is_active ?? true,
        models: [],
        created_at: '',
        updated_at: '',
      })}
      renderViewAdd={(form) => <AddForm form={form} />}
      renderViewUpdate={(form, entity) => <EditForm form={form} entity={entity} />}
    />
  );
}

// ---------- ProviderDetail ----------

function ProviderDetail({ entity }: { entity: Provider }) {
  const pid = entity.id;
  const queryClient = useQueryClient();
  const { Modal, modalHandler } = useModal();
  const { Confirm, confirmHandler } = useConfirm();
  const [editingModel, setEditingModel] = useState<Model | null>(null);
  const [formName, setFormName] = useState('');
  const [formDisplayName, setFormDisplayName] = useState('');
  const [formApiType, setFormApiType] = useState<'openai' | 'anthropic'>('openai');
  const [formMaxContext, setFormMaxContext] = useState('');
  const [formMaxOutput, setFormMaxOutput] = useState('');
  const [formInputPrice, setFormInputPrice] = useState('');
  const [formOutputPrice, setFormOutputPrice] = useState('');
  const [formDescription, setFormDescription] = useState('');
  const [formIsActive, setFormIsActive] = useState(true);

  // Fetch models dialog state
  const [fetchedModels, setFetchedModels] = useState<{ id: string }[]>([]);
  const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
  const [isFetching, setIsFetching] = useState(false);
  const [isImporting, setIsImporting] = useState(false);
  const { Modal: FetchModal, modalHandler: fetchModalHandler } = useModal();

  const { data: models = [], isLoading: modelsLoading } = useQuery({
    queryKey: ['provider-models', pid],
    queryFn: async () => {
      const res = await request.get<API.DataSet<Model>>(`/admin/models?provider_id=${pid}`);
      return (res.data.dataSet ?? []) as Model[];
    },
  });

  const deleteModelMutation = useMutation({
    mutationFn: (modelId: number) => modelService.delete(modelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['provider-models', pid] });
      queryClient.invalidateQueries({ queryKey: ['full-page', 'providers'] });
      toast.success('模型已删除');
    },
  });

  const saveModelMutation = useMutation({
    mutationFn: async (params: any) => {
      if (editingModel) {
        await modelService.update(editingModel.id, params);
      } else {
        await modelService.add({ ...params, provider_id: pid });
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['provider-models', pid] });
      queryClient.invalidateQueries({ queryKey: ['full-page', 'providers'] });
      modalHandler.close();
      toast.success(editingModel ? '模型已更新' : '模型已添加');
      resetForm();
    },
  });

  function resetForm() {
    setEditingModel(null);
    setFormName('');
    setFormDisplayName('');
    setFormApiType('openai');
    setFormMaxContext('');
    setFormMaxOutput('');
    setFormInputPrice('');
    setFormOutputPrice('');
    setFormDescription('');
    setFormIsActive(true);
  }

  async function handleFetchModels() {
    setIsFetching(true);
    setFetchedModels([]);
    setSelectedModels(new Set());
    fetchModalHandler.open('获取上游模型');
    try {
      const models = await fetchProviderModels(entity.base_url, entity.api_key || '', entity.preferred_api);
      setFetchedModels(models);
      // Auto-select all models that don't already exist
      const existingNames = new Set(models.map((m: any) => m.name));
      const newModels = models.filter((m: any) => !existingNames.has(m.name));
      setSelectedModels(new Set(newModels.map((m: any) => m.id)));
    } catch {
      toast.error('获取模型列表失败');
    } finally {
      setIsFetching(false);
    }
  }

  async function handleImportModels() {
    if (selectedModels.size === 0) return;
    setIsImporting(true);
    try {
      const result = await batchImportProviderModels(pid, Array.from(selectedModels));
      toast.success(`导入完成：新增 ${result.created} 个，跳过 ${result.skipped} 个`);
      queryClient.invalidateQueries({ queryKey: ['provider-models', pid] });
      queryClient.invalidateQueries({ queryKey: ['full-page', 'providers'] });
      fetchModalHandler.close();
    } catch {
      toast.error('导入失败');
    } finally {
      setIsImporting(false);
    }
  }

  function toggleModelSelection(modelId: string) {
    setSelectedModels((prev) => {
      const next = new Set(prev);
      if (next.has(modelId)) {
        next.delete(modelId);
      } else {
        next.add(modelId);
      }
      return next;
    });
  }

  function toggleAllModels() {
    if (selectedModels.size === fetchedModels.length) {
      setSelectedModels(new Set());
    } else {
      setSelectedModels(new Set(fetchedModels.map((m) => m.id)));
    }
  }

  function openAddModel() {
    resetForm();
    modalHandler.open('添加模型');
  }

  function openEditModel(m: Model) {
    setEditingModel(m);
    setFormName(m.name);
    setFormDisplayName(m.display_name);
    setFormApiType(m.api_type || 'openai');
    setFormMaxContext(m.max_context_tokens ? String(m.max_context_tokens) : '');
    setFormMaxOutput(m.max_output_tokens ? String(m.max_output_tokens) : '');
    setFormInputPrice(m.input_price ? String(m.input_price) : '');
    setFormOutputPrice(m.output_price ? String(m.output_price) : '');
    setFormDescription(m.description);
    setFormIsActive(m.is_active);
    modalHandler.open(`编辑模型 - ${m.name}`);
  }

  function handleSaveModel() {
    if (!formName) {
      toast.error('模型名称不能为空');
      return;
    }
    saveModelMutation.mutate({
      name: formName,
      display_name: formDisplayName,
      api_type: formApiType,
      max_context_tokens: Number(formMaxContext) || 0,
      max_output_tokens: Number(formMaxOutput) || 0,
      input_price: Number(formInputPrice) || 0,
      output_price: Number(formOutputPrice) || 0,
      description: formDescription,
      is_active: formIsActive,
      is_chat: true,
    });
  }

  return (
    <div className="flex flex-col gap-4">
      <Card className="border-t">
        <CardHeader>
          <CardTitle>服务商信息</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-2 md:grid-cols-2">
            <InfoRow label="名称" value={entity.name} />
            <InfoRow label="Base URL" value={<span className="font-mono text-xs">{entity.base_url}</span>} />
            <InfoRow
              label="协议"
              value={
                <div className="flex gap-1">
                  {entity.support_openai && <Badge variant="secondary">OpenAI</Badge>}
                  {entity.support_anthropic && <Badge variant="secondary">Anthropic</Badge>}
                </div>
              }
            />
            <InfoRow label="优先接口" value={<Badge variant="outline">{entity.preferred_api === 'openai' ? 'OpenAI' : 'Anthropic'}</Badge>} />
            <InfoRow
              label="状态"
              value={
                <Badge variant={entity.is_active ? 'default' : 'destructive'}>
                  {entity.is_active ? '启用' : '禁用'}
                </Badge>
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>上游模型 ({models.length})</CardTitle>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={handleFetchModels} disabled={isFetching}>
              {isFetching ? '获取中...' : '获取模型'}
            </Button>
            <Button size="sm" onClick={openAddModel}>
              新增模型
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {modelsLoading ? (
            <div className="text-muted-foreground text-sm">加载中...</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>模型名称</TableHead>
                  <TableHead>展示名</TableHead>
                  <TableHead>协议</TableHead>
                  <TableHead>上下文</TableHead>
                  <TableHead>最大输出</TableHead>
                  <TableHead>TPM</TableHead>
                  <TableHead>QPM</TableHead>
                  <TableHead>输入单价</TableHead>
                  <TableHead>输出单价</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-20">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={11} className="text-muted-foreground text-center">
                      暂无模型
                    </TableCell>
                  </TableRow>
                ) : (
                  models.map((m) => (
                    <TableRow key={m.id}>
                      <TableCell className="font-mono text-xs">{m.name}</TableCell>
                      <TableCell>{m.display_name || '-'}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">{m.api_type === 'openai' ? 'OpenAI' : 'Anthropic'}</Badge>
                      </TableCell>
                      <TableCell>{formatTokens(m.max_context_tokens)}</TableCell>
                      <TableCell>{formatTokens(m.max_output_tokens)}</TableCell>
                      <TableCell>{m.tpm ? formatTokens(m.tpm) : '-'}</TableCell>
                      <TableCell>{m.qpm || '-'}</TableCell>
                      <TableCell>{formatPrice(m.input_price)}</TableCell>
                      <TableCell>{formatPrice(m.output_price)}</TableCell>
                      <TableCell>
                        <Badge variant={m.is_active ? 'default' : 'destructive'}>
                          {m.is_active ? '启用' : '禁用'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          <Button variant="ghost" size="sm" onClick={() => openEditModel(m)}>
                            编辑
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive"
                            onClick={() =>
                              confirmHandler.confirmInvoke(
                                '确认删除',
                                async () => {
                                  await deleteModelMutation.mutateAsync(m.id);
                                  return true;
                                },
                                `确认要删除模型「${m.name}」吗？`,
                                true,
                              )
                            }
                          >
                            删除
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Modal>
        <div className="flex flex-col gap-4">
          <FormFieldInput
            form={{ state: { value: formName } } as any}
            name="formName"
            title="模型名称"
            required
            placeholder="例如: gpt-4o"
          />
          <div className="space-y-2">
            <label className="text-sm font-medium">展示名</label>
            <input
              value={formDisplayName}
              onChange={(e) => setFormDisplayName(e.target.value)}
              placeholder="用户友好的显示名称"
              className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">协议类型</label>
            <select
              value={formApiType}
              onChange={(e) => setFormApiType(e.target.value as 'openai' | 'anthropic')}
              className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
            >
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
            </select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">最大上下文 (tokens)</label>
              <input
                type="number"
                value={formMaxContext}
                onChange={(e) => setFormMaxContext(e.target.value)}
                placeholder="128000"
                className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">最大输出 (tokens)</label>
              <input
                type="number"
                value={formMaxOutput}
                onChange={(e) => setFormMaxOutput(e.target.value)}
                placeholder="4096"
                className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">输入单价 ($) per 1M tokens</label>
              <input
                type="number"
                value={formInputPrice}
                onChange={(e) => setFormInputPrice(e.target.value)}
                placeholder="0.00"
                className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">输出单价 ($) per 1M tokens</label>
              <input
                type="number"
                value={formOutputPrice}
                onChange={(e) => setFormOutputPrice(e.target.value)}
                placeholder="0.00"
                className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
              />
            </div>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">描述</label>
            <input
              value={formDescription}
              onChange={(e) => setFormDescription(e.target.value)}
              placeholder="模型描述信息"
              className="border-input bg-background ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm"
            />
          </div>
          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="model-active"
              checked={formIsActive}
              onChange={(e) => setFormIsActive(e.target.checked)}
              className="h-4 w-4"
            />
            <label htmlFor="model-active" className="text-sm font-medium">
              启用
            </label>
          </div>
          <Button disabled={saveModelMutation.isPending} onClick={handleSaveModel}>
            {saveModelMutation.isPending ? '保存中...' : '保存'}
          </Button>
        </div>
      </Modal>
      <Confirm />

      <FetchModal>
        <div className="flex flex-col gap-4 max-h-[60vh]">
          {isFetching ? (
            <div className="text-muted-foreground text-sm py-4">正在从上游获取模型列表...</div>
          ) : fetchedModels.length === 0 ? (
            <div className="text-muted-foreground text-sm py-4">未获取到任何模型</div>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">
                  共 {fetchedModels.length} 个模型，已选择 {selectedModels.size} 个
                </span>
                <Button variant="ghost" size="sm" onClick={toggleAllModels}>
                  {selectedModels.size === fetchedModels.length ? '取消全选' : '全选'}
                </Button>
              </div>
              <div className="flex flex-col gap-1 overflow-y-auto max-h-[40vh] pr-1">
                {fetchedModels.map((m) => (
                  <label
                    key={m.id}
                    className="flex items-center gap-2 rounded-md border p-2 hover:bg-muted cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      checked={selectedModels.has(m.id)}
                      onChange={() => toggleModelSelection(m.id)}
                      className="h-4 w-4"
                    />
                    <span className="font-mono text-sm">{m.id}</span>
                  </label>
                ))}
              </div>
              <Button disabled={isImporting || selectedModels.size === 0} onClick={handleImportModels}>
                {isImporting ? '导入中...' : `导入选中的 ${selectedModels.size} 个模型`}
              </Button>
            </>
          )}
        </div>
      </FetchModal>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground text-sm">{label}:</span>
      <span className="text-sm">{value}</span>
    </div>
  );
}
