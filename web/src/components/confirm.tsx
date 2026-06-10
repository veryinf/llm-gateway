import { useState, useEffect, useImperativeHandle, useRef, useCallback, useMemo } from 'react';
import { Button } from './ui/button';
import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from './ui/alert-dialog';
import { cn } from '@/lib/utils';
import { Loading } from './loader';

type ConfirmAction = {
  confirm(title: string, description?: string, destructive?: boolean): Promise<void>;
  confirmInvoke(title: string, invoker: () => Promise<boolean>, description?: string, destructive?: boolean): void;
};

type ConfirmProps = Omit<InnerConfirmProps, 'open' | 'onOpenChange'> & {};

type InnerConfirmProps = {
  title?: string;
  description?: string;
  destructive?: boolean;
  className?: string;
  children?: React.ReactNode;

  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function useConfirm() {
  const confirmRef = useRef<ConfirmAction>(null);

  const Confirm = useCallback((props: ConfirmProps) => {
    const callbackRef = useRef<{ resolve: () => void; reject: () => void; isConfirm: boolean }>(null);
    const invokerRef = useRef<() => Promise<boolean>>(null);
    const [visible, setVisible] = useState(false);
    const [loading, setLoading] = useState(false);
    const [override, setOverride] = useState<Partial<InnerConfirmProps>>();

    useImperativeHandle(confirmRef, () => ({
      confirm(title: string, description?: string, destructive?: boolean): Promise<void> {
        setOverride({
          ...override,
          title,
          description,
          destructive,
        });
        setVisible(true);
        return new Promise<void>((resolve, reject) => {
          if (callbackRef.current?.reject) {
            callbackRef.current.reject();
          }
          callbackRef.current = { resolve, reject, isConfirm: false };
        });
      },
      confirmInvoke(title, invoker, description, destructive) {
        setOverride({
          ...override,
          title,
          description,
          destructive,
        });
        setVisible(true);
        invokerRef.current = invoker;
      },
    }));

    useEffect(() => {
      if (!visible) {
        if (callbackRef.current?.isConfirm) {
          callbackRef.current?.resolve();
        } else {
          callbackRef.current?.reject();
        }
        callbackRef.current = null;
        invokerRef.current = null;
      }
    }, [visible]);

    async function handleConfirm() {
      if (invokerRef.current) {
        setLoading(true);
        const ret = await invokerRef.current();
        setLoading(false);
        if (!ret) {
          return;
        }
      }
      if (callbackRef.current) {
        callbackRef.current.isConfirm = true;
      }
      setVisible(false);
    }

    const finalProps = { ...props, ...override };

    return (
      <AlertDialog open={visible} onOpenChange={setVisible}>
        <AlertDialogContent className={cn(finalProps.className && finalProps.className)}>
          <AlertDialogHeader className="text-start">
            <AlertDialogTitle>{finalProps.title}</AlertDialogTitle>
            {finalProps.description && <AlertDialogDescription>{finalProps.description}</AlertDialogDescription>}
          </AlertDialogHeader>
          {finalProps.children}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={loading} onClick={() => setVisible(false)}>
              取消
            </AlertDialogCancel>
            <Button variant={finalProps.destructive ? 'destructive' : 'default'} disabled={loading} onClick={handleConfirm}>
              {loading ? (
                <>
                  <Loading />
                  处理中...
                </>
              ) : (
                '确认'
              )}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  }, []);

  const confirmHandler: ConfirmAction = useMemo(
    () => ({
      confirm: async (title, description, destructive) => {
        await confirmRef.current?.confirm(title, description, destructive);
      },
      confirmInvoke: (title, invoker, description, destructive) => {
        confirmRef.current?.confirmInvoke(title, invoker, description, destructive);
      },
    }),
    [],
  );

  return { Confirm, confirmHandler };
}
