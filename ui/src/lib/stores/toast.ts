import { writable } from 'svelte/store';

export interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
}

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);

  function add(message: string, type: Toast['type'] = 'info') {
    const id = crypto.randomUUID();
    update((toasts) => [...toasts, { id, message, type }]);
    setTimeout(() => dismiss(id), 4000);
  }

  function dismiss(id: string) {
    update((toasts) => toasts.filter((t) => t.id !== id));
  }

  return {
    subscribe,
    success: (msg: string) => add(msg, 'success'),
    error: (msg: string) => add(msg, 'error'),
    info: (msg: string) => add(msg, 'info'),
    dismiss,
  };
}

export const toasts = createToastStore();
