<script lang="ts">
  import type { Check } from '$lib/types';
  import StatusBadge from './StatusBadge.svelte';
  import TimeAgo from './TimeAgo.svelte';

  interface Props {
    check: Check;
  }

  let { check }: Props = $props();

  const borderColors: Record<string, string> = {
    up: 'border-l-status-up',
    down: 'border-l-status-down',
    alerting: 'border-l-status-alerting',
    new: 'border-l-status-new',
    silenced: 'border-l-status-silenced',
  };
</script>

<a
  href="/checks/{check.id}"
  class="block bg-surface border border-border {borderColors[check.status]} border-l-2 rounded-card p-4 hover:bg-border/30 transition-colors"
  role="article"
  aria-label="{check.name}, status {check.status}"
>
  <div class="flex items-center justify-between mb-2">
    <h3 class="font-medium truncate">{check.name}</h3>
    <StatusBadge status={check.status} size="sm" />
  </div>
  <div class="flex items-center gap-3 text-sm">
    <span class="text-text-secondary">Last ping:</span>
    <TimeAgo date={check.last_ping_at} fallback="waiting..." />
  </div>
</a>
