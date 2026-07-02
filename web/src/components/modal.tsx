import { useState, useImperativeHandle, useRef, useCallback, useMemo } from 'react';
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from './ui/sheet';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog';

type ModalAction<T> = {
  open(title: string, description?: string, meta?: T): void;
  show(props: Partial<ModalProps>, meta?: T): void;
  close(): void;
};

type ModalProps = Omit<InnerModalProps, 'open' | 'onOpenChange'> & {
  type?: 'drawer' | 'dialog';
};

type InnerModalProps = {
  title?: string;
  description?: string;
  actions?: React.ReactNode;
  showCloseButton?: boolean;
  className?: string;
  children?: React.ReactNode;

  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function useModal<T = any>(initMeta?: T) {
  const modalRef = useRef<ModalAction<T>>(null);
  const [meta, setMeta] = useState<T | undefined>(initMeta);

  const Modal = useCallback((props: ModalProps) => {
    const [visible, setVisible] = useState(false);
    const [override, setOverride] = useState<Partial<InnerModalProps>>();

    useImperativeHandle(modalRef, () => ({
      open: (title: string, description?: string, meta?: T) => {
        setOverride({
          ...override,
          title,
          description,
        });
        setMeta(meta);
        setVisible(true);
      },
      show: (props: Partial<ModalProps>, meta?: T) => {
        setOverride({
          ...override,
          ...props,
        });
        setMeta(meta);
        setVisible(true);
      },
      close: () => {
        setVisible(false);
      },
    }));

    const Comp = props.type === 'dialog' ? ModalDialog : ModalDrawer;
    return <Comp {...props} {...override} open={visible} onOpenChange={setVisible} />;
  }, []);

  const modalHandler = useMemo(
    () => ({
      open: (title: string, description?: string, meta?: T) => {
        modalRef.current?.open(title, description, meta);
      },
      show: (props: Partial<ModalProps>, meta?: T) => {
        modalRef.current?.show(props, meta);
      },
      close: () => {
        modalRef.current?.close();
      },
    }),
    [],
  );

  return { Modal, modalHandler, meta };
}

function ModalDrawer(props: InnerModalProps) {
  const { showCloseButton = true } = props;
  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className="w-200!" side="right" showCloseButton={showCloseButton}>
        {props.title || props.description ? (
          <SheetHeader className="text-left">
            <SheetTitle>
              <div className="flex items-center gap-2">
                <span className="flex-1">{props.title}</span>
                <div className="flex gap-2 items-center">{props.actions}</div>
              </div>
            </SheetTitle>
            <SheetDescription>{props.description}</SheetDescription>
          </SheetHeader>
        ) : null}
        <div className="px-4 overflow-auto pb-8">{props.children}</div>
        <SheetFooter className="hidden"></SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function ModalDialog(props: InnerModalProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className={props.className}>
        {props.title || props.description ? (
          <DialogHeader>
            <DialogTitle>
              <div className="flex items-center gap-2">
                <span className="flex-1">{props.title}</span>
                <div className="flex gap-2 items-center">{props.actions}</div>
              </div>
            </DialogTitle>
            {props.description && <DialogDescription>{props.description}</DialogDescription>}
          </DialogHeader>
        ) : null}
        <div className="overflow-auto max-h-[70vh]">{props.children}</div>
      </DialogContent>
    </Dialog>
  );
}
