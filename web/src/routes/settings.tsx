import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import { toast } from 'sonner';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Field, FieldGroup } from '@/components/ui/field';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch } from '@/components/form';
import { Loading } from '@/components/loader';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { configService, CONFIG_KEYS } from '@/services/config';

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
});

const passthroughOptions = [
  { label: '禁用透传', value: 'none' },
  { label: '用户级透传', value: 'user' },
  { label: '提供商级透传', value: 'provider' },
];

const settingsSchema = z.object({
  logRetention: z.number().int().min(1, '必须是大于 0 的整数'),
  passthrough: z.enum(['none', 'user', 'provider']),
  requestDetail: z.boolean(),
  requestRetentionDays: z.number().int().min(1, '必须是大于 0 的整数'),
});

type SettingType = z.infer<typeof settingsSchema>;
const defaultSettingValue: SettingType = {
  logRetention: 7,
  passthrough: 'none',
  requestDetail: false,
  requestRetentionDays: 7,
};

function SettingsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const queryClient = useQueryClient();

  useEffect(() => {
    setBreadcrumbs([{ title: '设置' }]);
  }, []);

  const keys = Object.values(CONFIG_KEYS);
  const { data: configObject, isLoading } = useQuery({
    queryKey: ['configs'],
    queryFn: async () => {
      const configMap = await configService.get(keys);
      return {
        logRetention: Number(configMap[CONFIG_KEYS.LOG_RETENTION]) || defaultSettingValue.logRetention,
        passthrough: configMap[CONFIG_KEYS.ROUTER_PASSTHROUGH] || defaultSettingValue.passthrough,
        requestDetail: configMap[CONFIG_KEYS.REQUEST_LOG_DETAIL] == 'true',
        requestRetentionDays: Number(configMap[CONFIG_KEYS.REQUEST_RETENTION_DAYS]) || defaultSettingValue.requestRetentionDays,
      } as SettingType;
    },
  });

  const form = useForm({
    defaultValues: configObject ?? defaultSettingValue,
    validators: {
      onChange: settingsSchema,
    },
    onSubmit: async ({ value }) => {
      try {
        await configService.save({
          [CONFIG_KEYS.REQUEST_LOG_DETAIL]: value.requestDetail ? 'true' : 'false',
          [CONFIG_KEYS.LOG_RETENTION]: String(value.logRetention),
          [CONFIG_KEYS.ROUTER_PASSTHROUGH]: value.passthrough,
          [CONFIG_KEYS.REQUEST_RETENTION_DAYS]: String(value.requestRetentionDays),
        });
        queryClient.invalidateQueries({ queryKey: ['configs'] });
        toast.success('保存成功');
      } catch {
        toast.error('保存失败');
      }
    },
  });

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <Loading size={32} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 p-4">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">系统设置</h2>
        <p className="text-muted-foreground text-sm">管理系统全局配置参数</p>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.handleSubmit();
        }}
      >
        <FieldGroup className="gap-4 max-w-2xl">
          {/* 请求配置 */}
          <div className="rounded-lg border bg-card p-4">
            <div className="grid gap-4 md:grid-cols-2">
              <FormFieldSwitch
                form={form}
                name="requestDetail"
                title="请求详情记录"
                description="记录完整的请求/响应 body 数据"
                switchLabel="启用"
              />
              <FormFieldInput
                form={form}
                name="requestRetentionDays"
                title="请求日志保留天数"
                type="number"
                description="超过此天数的请求日志和流式 chunks 将被自动清理，默认 7 天"
              />
            </div>
          </div>

          {/* 日志文件配置 */}
          <div className="rounded-lg border bg-card p-4">
            <FormFieldInput
              form={form}
              name="logRetention"
              title="日志文件保留天数"
              type="number"
              description="超过此天数的日志文件将被自动清理，默认 7 天"
            />
          </div>

          {/* 路由配置 */}
          <div className="rounded-lg border bg-card p-4">
            <FormFieldSelect
              form={form}
              name="passthrough"
              title="透传级别"
              options={passthroughOptions}
              description={<>用户级：跳过 UserModel 直接匹配 ProviderModel<br />提供商级：跳过 ProviderModel 直接使用默认 Provider</>}
            />
          </div>

          {/* 保存按钮 */}
          <Field>
            <form.Subscribe>
              {(state) => (
                <Button type="submit" disabled={state.isSubmitting}>
                  {state.isSubmitting ? '保存中...' : '保存设置'}
                </Button>
              )}
            </form.Subscribe>
          </Field>
        </FieldGroup>
      </form>
    </div>
  );
}
