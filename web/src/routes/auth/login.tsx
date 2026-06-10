import { Shield } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Field, FieldGroup } from '@/components/ui/field';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useForm } from '@tanstack/react-form';
import { login } from '@/services/auth';
import { FormFieldInput } from '@/components/form';
import { useAuth } from '@/hooks/use-auth';

export const Route = createFileRoute('/auth/login')({
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const form = useForm({
    defaultValues: { username: '', password: '' },
    onSubmit: async ({ value }) => {
      try {
        await login(value);
        toast.success('登录成功');
        navigate({ to: '/dashboard' });
      } catch (err: any) {
        toast.error('登录失败', { description: err.message || '用户名或密码错误' });
      }
    },
  });

  if (isAuthenticated) {
    navigate({ to: '/' });
    return null;
  }

  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      <div className="bg-muted relative hidden lg:flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-muted-foreground">
          <Shield className="size-16" />
          <span className="text-lg font-medium">LLM Gateway</span>
        </div>
      </div>
      <div className="flex flex-col gap-4 p-6 md:p-10">
        <div className="flex justify-center gap-2 md:justify-start">
          <a href="#" className="flex items-center gap-2 font-medium">
            <div className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
              <Shield className="size-4" />
            </div>
            LLM Gateway
          </a>
        </div>
        <div className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-xs">
            <form
              onSubmit={(e) => {
                e.preventDefault();
                form.handleSubmit();
              }}
            >
              <FieldGroup className="gap-6">
                <div className="flex flex-col items-center gap-1 text-center">
                  <h1 className="text-2xl font-bold">管理后台登录</h1>
                  <p className="text-muted-foreground text-sm text-balance">请输入您的用户名和密码</p>
                </div>
                <FormFieldInput form={form} name="username" title="用户名" placeholder="请输入用户名" required />
                <FormFieldInput form={form} name="password" title="密码" placeholder="请输入密码" type="password" required />
                <Field>
                  <form.Subscribe>
                    {(state) => {
                      const submitting = state.isSubmitting || state.isSubmitted;
                      return (
                        <Button type="submit" className="w-full" disabled={submitting}>
                          {submitting ? '登录中...' : '登录'}
                        </Button>
                      );
                    }}
                  </form.Subscribe>
                </Field>
              </FieldGroup>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
