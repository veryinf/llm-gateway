import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useForm } from '@tanstack/react-form';
import { toast } from 'sonner';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { PageHeader } from '@/components/page-header';
import { FormFieldInput } from '@/components/form';
import { Button } from '@/components/ui/button';
import { Loading } from '@/components/loader';
import { configService, type Config } from '@/services/config';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
});

const CONFIG_FIELDS = [
  { key: 'audit_retention_days', title: '审计日志保留天数', type: 'number' as const, placeholder: '90' },
];

function SettingsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const queryClient = useQueryClient();

  useEffect(() => {
    setBreadcrumbs([{ title: '设置' }]);
  }, []);

  const { data: configs, isLoading } = useQuery({
    queryKey: ['configs'],
    queryFn: () => configService.list(),
  });

  const configMap = Object.fromEntries((configs ?? []).map((c: Config) => [c.key, c.value]));

  const mutation = useMutation({
    mutationFn: (params: { key: string; value: string }) => configService.update(params.key, params.value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['configs'] });
      toast.success('保存成功');
    },
    onError: () => {
      toast.error('保存失败');
    },
  });

  const form = useForm({
    defaultValues: {
      audit_retention_days: '',
    },
    onSubmit: async ({ value }) => {
      await Promise.all(
        CONFIG_FIELDS.map((field) => {
          const val = String(value[field.key as keyof typeof value] ?? '');
          return mutation.mutateAsync({ key: field.key, value: val });
        }),
      );
    },
  });

  useEffect(() => {
    if (configs) {
      form.setFieldValue('audit_retention_days', configMap['audit_retention_days'] ?? '90');
    }
  }, [configs]);

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <Loading size={32} />
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <PageHeader title="系统设置" description="管理系统全局配置参数" />
      <div className="rounded-lg border bg-card p-6">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            e.stopPropagation();
            form.handleSubmit();
          }}
          className="flex flex-col gap-6 max-w-xl"
        >
          <FormFieldInput form={form} name="audit_retention_days" title="审计日志保留天数" type="number" placeholder="90" tips="审计日志超过此天数将被自动清理" />
          <div>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? '保存中...' : '保存设置'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
