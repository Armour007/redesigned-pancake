import { writable } from 'svelte/store';

export type ToastVariant = 'info' | 'success' | 'error' | 'warning';
export type ToastItem = { id: number; message: string; variant: ToastVariant; timeout: number };

function createToastStore() {
  const { subscribe, update } = writable<ToastItem[]>([]);
  let idCounter = 1;

  function push(message: string, variant: ToastVariant = 'info', timeout = 3000) {
    const id = idCounter++;
    const item: ToastItem = { id, message, variant, timeout };
    update((list) => [...list, item]);
    setTimeout(() => {
      update((list) => list.filter((t) => t.id !== id));
    }, timeout);
  }

  function remove(id: number) {
    update((list) => list.filter((t) => t.id !== id));
  }

  return { subscribe, push, remove };
}

export const toasts = createToastStore();
export const toast = (message: string, variant: ToastVariant = 'info', timeout = 3000) => toasts.push(message, variant, timeout);
