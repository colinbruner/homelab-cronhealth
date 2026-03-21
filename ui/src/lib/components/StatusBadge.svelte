<script lang="ts">
  import type { CheckStatus } from '$lib/types';

  interface Props {
    status: CheckStatus;
    size?: 'sm' | 'md';
  }

  let { status, size = 'md' }: Props = $props();

  const labels: Record<CheckStatus, string> = {
    new: 'NEW',
    up: 'UP',
    down: 'DOWN',
    alerting: 'ALERTING',
    silenced: 'SILENCED',
  };

  const colors: Record<CheckStatus, string> = {
    new: 'bg-status-new/20 text-status-new',
    up: 'bg-status-up/20 text-status-up',
    down: 'bg-status-down/20 text-status-down',
    alerting: 'bg-status-alerting/20 text-status-alerting',
    silenced: 'bg-status-silenced/20 text-status-silenced',
  };
</script>

<span
  class="inline-flex items-center font-mono font-medium rounded-badge {colors[status]} {size === 'sm' ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-1 text-sm'}"
  aria-label="Status: {labels[status]}"
>
  {labels[status]}
</span>
