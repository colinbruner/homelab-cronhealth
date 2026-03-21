<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';
  import type { Alert } from '$lib/types';
  import TimeAgo from '$lib/components/TimeAgo.svelte';
  import SkeletonRow from '$lib/components/SkeletonRow.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';

  let alerts: Alert[] = $state([]);
  let loading = $state(true);
  let actionLoading = $state<string | null>(null);

  const active = $derived(alerts.filter((a) => !a.resolved_at));
  const resolved = $derived(alerts.filter((a) => a.resolved_at));

  async function handleSnooze(checkId: string, minutes: number) {
    actionLoading = checkId;
    try {
      await api.snoozeCheck(checkId, { duration_minutes: minutes });
      toasts.success(`Snoozed for ${minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`}`);
      checksStore.updateCheckStatus(checkId, 'silenced');
      alerts = await api.listAlerts();
    } catch {
      toasts.error('Snooze failed');
    }
    actionLoading = null;
  }

  onMount(async () => {
    try {
      alerts = await api.listAlerts();
    } catch {
      toasts.error('Failed to load alerts');
    }
    loading = false;
  });
</script>

<svelte:head>
  <title>Alerts - cronhealth</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">Alerts</h1>

  {#if loading}
    {#each Array(5) as _}
      <SkeletonRow />
    {/each}
  {:else if alerts.length === 0}
    <EmptyState
      title="No alerts"
      description="Everything looks healthy. No active or recent alerts."
    />
  {:else}
    {#if active.length > 0}
      <section class="mb-8">
        <h2 class="text-sm font-medium text-status-down mb-3 uppercase tracking-wide">Active</h2>
        <div class="bg-surface border border-border rounded-card divide-y divide-border">
          {#each active as alert (alert.id)}
            <div class="flex items-center justify-between px-4 py-3 text-sm">
              <a href="/checks/{alert.check_id}" class="flex items-center gap-3 hover:text-white transition-colors">
                <span class="text-status-down font-medium">{alert.check_name}</span>
                <span class="text-text-secondary">firing since</span>
                <TimeAgo date={alert.started_at} />
              </a>
              <div class="flex items-center gap-2">
                <span class="font-mono text-text-secondary text-xs">{alert.alert_count} sent</span>
                <button
                  onclick={(e) => { e.stopPropagation(); handleSnooze(alert.check_id, 60); }}
                  disabled={actionLoading === alert.check_id}
                  class="border border-border rounded-badge px-2 py-0.5 text-xs text-text-secondary hover:text-text-primary transition-colors disabled:opacity-50"
                >
                  Snooze 1h
                </button>
              </div>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    {#if resolved.length > 0}
      <section>
        <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Resolved</h2>
        <div class="bg-surface border border-border rounded-card divide-y divide-border">
          {#each resolved as alert (alert.id)}
            <a
              href="/checks/{alert.check_id}"
              class="flex items-center justify-between px-4 py-3 text-sm hover:bg-border/30 transition-colors text-text-secondary"
            >
              <div class="flex items-center gap-3">
                <span>{alert.check_name}</span>
                <TimeAgo date={alert.resolved_at} />
              </div>
              <span class="font-mono text-xs">{alert.alert_count} sent</span>
            </a>
          {/each}
        </div>
      </section>
    {/if}
  {/if}
</div>
