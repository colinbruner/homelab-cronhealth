<script lang="ts">
  import type { Toast as ToastType } from '$lib/stores/toast';
  import { toasts } from '$lib/stores/toast';

  interface Props {
    toast: ToastType;
  }

  let { toast }: Props = $props();

  const bgColors = {
    success: 'bg-status-up/10 border-status-up/30',
    error: 'bg-status-down/10 border-status-down/30',
    info: 'bg-surface border-border',
  };
</script>

<div
  class="flex items-center gap-3 px-4 py-3 rounded-card border {bgColors[toast.type]} text-sm"
  role="alert"
>
  <span class="flex-1">{toast.message}</span>
  <button
    class="text-text-secondary hover:text-text-primary"
    onclick={() => toasts.dismiss(toast.id)}
    aria-label="Dismiss"
  >
    &times;
  </button>
</div>
