import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import { toast } from 'sonner';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { PageHeader } from '@/components/page-header';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Loading } from '@/components/loader';
import { request } from '@/lib';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { configService, type Config } from '@/services/config';

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
});

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

  const configMap = Object.fromEntries((configs?.dataSet ?? []).map((c: Config) => [c.key, c.value]));

  const retentionMutation = useMutation({
    mutationFn: (params: { key: string; value: string }) => configService.update(params.key, params.value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['configs'] });
      toast.success('保存成功');
    },
    onError: () => {
      toast.error('保存失败');
    },
  });

  const detailMutation = useMutation({
    mutationFn: (value: string) =>
      request.put('/admin/config/request-detail', { value }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['configs'] });
      toast.success('开关已更新');
    },
    onError: () => {
      toast.error('更新失败');
    },
  });

  const logDetailEnabled = configMap['log_request_detail'] === 'true';

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

      {/* Request Detail Toggle */}
      <div className="rounded-lg border bg-card p-6">
        <h3 className="mb-4 text-lg font-medium">请求详情记录</h3>
        <div className="flex items-center gap-3">
          <Switch
            id="log-detail"
            checked={logDetailEnabled}
            onCheckedChange={(checked) => {
              detailMutation.mutate(checked ? 'true' : 'false');
            }}
            disabled={detailMutation.isPending}
          />
          <Label htmlFor="log-detail" className="cursor-pointer">
            {logDetailEnabled ? '已开启' : '已关闭'}
          </Label>
        </div>
        <p className="text-muted-foreground mt-2 text-sm">
          开启后将记录完整的请求/响应 body 数据。非流式请求记录在 response_body 字段，流式请求记录在 request_chunks 表中。
        </p>
      </div>

      {/* Retention Config */}
      <div className="rounded-lg border bg-card p-6">
        <h3 className="mb-4 text-lg font-medium">数据保留</h3>
        <div className="flex flex-col gap-4 max-w-xl">
          <div className="flex items-center gap-4">
            <Label htmlFor="retention-days" className="w-40 shrink-0">请求日志保留天数</Label>
            <input
              id="retention-days"
              type="number"
              defaultValue={configMap['request_log_retention_days'] ?? '90'}
              onBlur={(e) => {
                retentionMutation.mutate({ key: 'request_log_retention_days', value: e.target.value });
              }}
              className="border-input bg-background ring-ring h-9 w-32 rounded-md border px-3 text-sm"
            />
          </div>
          <p className="text-muted-foreground text-sm">
            超过此天数的请求日志和流式 chunks 将被自动清理。
          </p>
        </div>
      </div>
    </div>
  );
}
