import { useImperativeHandle, useRef } from 'react';
import { useForm } from '@tanstack/react-form';
import { useModal } from '../modal';
import { Button } from '../ui/button';
import type { EasyFormApi, EasyFormMeta, EasyFormOptions } from './utils';

type PopupFormAction<T, F> = {
  open(title: string, formData: T | undefined, formState: F): void;
  close(): void;
};

type PopupFormProps<T, F> = {
  onSubmit?: (values: T, meta: EasyFormMeta<T, F>, form?: EasyFormApi<T>) => Promise<boolean>;
  children?: React.ReactNode | ((from: EasyFormApi<T>, formData: T | undefined, stateData: F | undefined) => React.ReactNode);
};

export type DefaultFormState = { action: 'add' | 'edit' };
export type FormSubmit<T> = EasyFormOptions<T>['onSubmit'];

export function usePopupForm<TFormData, TState>() {
  const formRef = useRef<PopupFormAction<TFormData, TState>>(null);
  const submitRef = useRef<PopupFormProps<TFormData, TState>['onSubmit']>(null);

  const form = useForm({
    defaultValues: undefined as TFormData,
    validators: {},
    onSubmit(s) {
      if (submitRef.current) {
        submitRef.current(s.value, s.meta as EasyFormMeta<TFormData, TState>, s.formApi as EasyFormApi<TFormData>).then((r) => {
          if (r) {
            formRef.current?.close();
          }
        });
      }
    },
  });
  const PopupForm = function (props: PopupFormProps<TFormData, TState>) {
    const { Modal, modalHandler, meta } = useModal<EasyFormMeta<TFormData, TState>>();
    if (props.onSubmit) {
      submitRef.current = props.onSubmit;
    }

    useImperativeHandle(formRef, () => ({
      open: (title: string, formData: TFormData | undefined, formState: TState) => {
        form.reset(formData);
        modalHandler.show({ title: title, showCloseButton: false }, { original: formData, state: formState });
      },
      close: () => {
        modalHandler.close();
      },
    }));

    return (
      <Modal
        actions={
          <>
            <Button className="h-8 px-6" variant="secondary" onClick={() => modalHandler.close()}>
              取消
            </Button>
            <Button className="h-8 px-6" onClick={() => form.handleSubmit({ ...meta })}>
              确认
            </Button>
          </>
        }
      >
        {typeof props.children === 'function' ? props.children(form, meta?.original, meta?.state) : props.children}
      </Modal>
    );
  };
  const formHandler: PopupFormAction<TFormData, TState> = {
    open(title: string, formData: TFormData | undefined, formState: TState) {
      formRef.current?.open(title, formData, formState);
    },
    close() {
      formRef.current?.close();
    },
  };
  return { PopupForm, form, formHandler };
}
