<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';
  import type { Check, Ping, Alert } from '$lib/types';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import TimeAgo from '$lib/components/TimeAgo.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import SkeletonRow from '$lib/components/SkeletonRow.svelte';

  let check = $state<Check | null>(null);
  let pings = $state<Ping[]>([]);
  let alerts = $state<Alert[]>([]);
  let loading = $state(true);
  let pingsLoading = $state(true);
  let showSnoozeModal = $state(false);
  let showDeleteModal = $state(false);
  let actionLoading = $state(false);

  const checkId = $derived($page.params.id!);
  const pingUrl = $derived(
    check ? `${window.location.origin}/ping/${check.slug}` : ''
  );

  onMount(async () => {
    try {
      check = await api.getCheck(checkId);
      loading = false;

      const [pingsResult, alertsResult] = await Promise.all([
        api.listPings(checkId),
        api.listAlerts(),
      ]);
      pings = pingsResult;
      alerts = alertsResult.filter((a) => a.check_id === checkId);
      pingsLoading = false;
    } catch {
      toasts.error('Failed to load check');
      loading = false;
      pingsLoading = false;
    }
  });

  async function handleSnooze(minutes: number) {
    actionLoading = true;
    try {
      await api.snoozeCheck(checkId, { duration_minutes: minutes });
      toasts.success(`Snoozed for ${minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`}`);
      check = await api.getCheck(checkId);
      if (check) checksStore.updateCheck(check);
      showSnoozeModal = false;
    } catch {
      toasts.error('Snooze failed');
    }
    actionLoading = false;
  }

  async function handleSilence() {
    actionLoading = true;
    try {
      await api.silenceCheck(checkId, {});
      toasts.success('Check silenced');
      check = await api.getCheck(checkId);
      if (check) checksStore.updateCheck(check);
    } catch {
      toasts.error('Silence failed');
    }
    actionLoading = false;
  }

  async function handleRemoveSilence() {
    actionLoading = true;
    try {
      await api.removeSilence(checkId);
      toasts.success('Silence removed');
      check = await api.getCheck(checkId);
      if (check) checksStore.updateCheck(check);
    } catch {
      toasts.error('Failed to remove silence');
    }
    actionLoading = false;
  }

  async function handleDelete() {
    actionLoading = true;
    try {
      await api.deleteCheck(checkId);
      checksStore.removeCheck(checkId);
      toasts.success('Check deleted');
      goto('/');
    } catch {
      toasts.error('Delete failed');
    }
    actionLoading = false;
  }

  function copyPingUrl() {
    navigator.clipboard.writeText(pingUrl);
    toasts.success('Ping URL copied');
  }
</script>

<svelte:head>
  <title>{check?.name ?? 'Check'} - cronhealth</title>
</svelte:head>

<div class="max-w-5xl mx-auto px-4 py-6">
  {#if loading}
    <div class="animate-pulse">
      <div class="h-8 w-64 bg-border rounded mb-4"></div>
      <div class="h-5 w-40 bg-border rounded mb-6"></div>
    </div>
  {:else if check}
    <!-- Header: name + status -->
    <div class="flex items-start justify-between mb-4">
      <div>
        <h1 class="text-2xl font-semibold mb-2">{check.name}</h1>
        <StatusBadge status={check.status} />
      </div>
      <a
        href="/checks/{check.id}/edit"
        class="text-text-secondary hover:text-text-primary text-sm border border-border rounded-card px-3 py-1.5"
      >
        Edit
      </a>
    </div>

    <!-- Timing info -->
    <div class="flex flex-wrap gap-6 mb-6 text-sm">
      <div>
        <span class="text-text-secondary">Last ping:</span>
        <TimeAgo date={check.last_ping_at} fallback="waiting for first ping..." />
      </div>
      <div>
        <span class="text-text-secondary">Period:</span>
        <span class="font-mono">{check.period_seconds}s</span>
      </div>
      <div>
        <span class="text-text-secondary">Grace:</span>
        <span class="font-mono">{check.grace_seconds}s</span>
      </div>
    </div>

    <!-- Ping URL -->
    <div class="bg-surface border border-border rounded-card p-4 mb-6">
      <div class="text-sm text-text-secondary mb-2">Ping URL</div>
      <div class="flex items-center gap-2">
        <code class="font-mono text-sm text-text-primary flex-1 truncate">{pingUrl}</code>
        <button
          onclick={copyPingUrl}
          class="text-text-secondary hover:text-text-primary text-sm border border-border rounded-card px-2 py-1 shrink-0"
        >
          Copy
        </button>
      </div>
      <pre class="mt-3 text-xs text-text-secondary font-mono">curl -fsS -X POST {pingUrl}</pre>
    </div>

    <!-- Two-panel layout: info left, history right on desktop -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <div>
        <!-- Action bar -->
        <div class="flex flex-wrap gap-2 mb-8">
          <button
            onclick={() => (showSnoozeModal = true)}
            disabled={actionLoading}
            class="border border-border rounded-card px-3 py-1.5 text-sm hover:bg-border/30 transition-colors disabled:opacity-50"
          >
            Snooze
          </button>
          {#if check.status === 'silenced'}
            <button
              onclick={handleRemoveSilence}
              disabled={actionLoading}
              class="border border-border rounded-card px-3 py-1.5 text-sm hover:bg-border/30 transition-colors disabled:opacity-50"
            >
              Remove Silence
            </button>
          {:else}
            <button
              onclick={handleSilence}
              disabled={actionLoading}
              class="border border-border rounded-card px-3 py-1.5 text-sm hover:bg-border/30 transition-colors disabled:opacity-50"
            >
              Silence
            </button>
          {/if}
          <button
            onclick={() => (showDeleteModal = true)}
            disabled={actionLoading}
            class="border border-status-down/30 text-status-down rounded-card px-3 py-1.5 text-sm hover:bg-status-down/10 transition-colors disabled:opacity-50"
          >
            Delete
          </button>
        </div>
      </div>

      <div>
        <!-- Ping history -->
        <section class="mb-8">
          <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Ping History</h2>
          {#if pingsLoading}
            {#each Array(5) as _}
              <SkeletonRow />
            {/each}
          {:else if pings.length === 0}
            <p class="text-text-secondary text-sm py-4">No pings yet — waiting for first ping.</p>
          {:else}
            <div class="bg-surface border border-border rounded-card divide-y divide-border">
              {#each pings as ping (ping.id)}
                <div class="flex items-center justify-between px-4 py-2.5 text-sm">
                  <TimeAgo date={ping.received_at} />
                  <div class="flex items-center gap-4">
                    {#if ping.source_ip}
                      <span class="font-mono text-text-secondary text-xs hidden md:inline">{ping.source_ip}</span>
                    {/if}
                    {#if ping.exit_code !== null}
                      <span class="font-mono text-xs {ping.exit_code === 0 ? 'text-status-up' : 'text-status-down'}">
                        exit {ping.exit_code}
                      </span>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </section>

        <!-- Alert log -->
        {#if alerts.length > 0}
          <section>
            <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Alert History</h2>
            <div class="bg-surface border border-border rounded-card divide-y divide-border">
              {#each alerts as alert (alert.id)}
                <div class="flex items-center justify-between px-4 py-2.5 text-sm">
                  <div class="flex items-center gap-3">
                    <span class={alert.resolved_at ? 'text-text-secondary' : 'text-status-down font-medium'}>
                      {alert.resolved_at ? 'Resolved' : 'Active'}
                    </span>
                    <TimeAgo date={alert.started_at} />
                  </div>
                  <span class="font-mono text-text-secondary text-xs">{alert.alert_count} notification{alert.alert_count !== 1 ? 's' : ''}</span>
                </div>
              {/each}
            </div>
          </section>
        {/if}
      </div>
    </div>

    <!-- Snooze modal -->
    <Modal title="Snooze Check" bind:open={showSnoozeModal} onclose={() => (showSnoozeModal = false)}>
      <p class="text-text-secondary text-sm mb-4">Suppress notifications for:</p>
      <div class="grid grid-cols-2 gap-2">
        {#each [30, 60, 240, 1440] as minutes}
          <button
            onclick={() => handleSnooze(minutes)}
            disabled={actionLoading}
            class="border border-border rounded-card py-3 text-sm font-medium hover:bg-border/30 transition-colors disabled:opacity-50"
          >
            {minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`}
          </button>
        {/each}
      </div>
    </Modal>

    <!-- Delete confirm modal -->
    <Modal title="Delete Check" bind:open={showDeleteModal} onclose={() => (showDeleteModal = false)}>
      <p class="text-text-secondary text-sm mb-4">
        Delete <strong class="text-text-primary">{check.name}</strong>? This removes all ping history and cannot be undone.
      </p>
      <div class="flex gap-2 justify-end">
        <button
          onclick={() => (showDeleteModal = false)}
          class="border border-border rounded-card px-4 py-2 text-sm hover:bg-border/30 transition-colors"
        >
          Cancel
        </button>
        <button
          onclick={handleDelete}
          disabled={actionLoading}
          class="bg-status-down/20 text-status-down border border-status-down/30 rounded-card px-4 py-2 text-sm hover:bg-status-down/30 transition-colors disabled:opacity-50"
        >
          Delete
        </button>
      </div>
    </Modal>
  {:else}
    <p class="text-text-secondary">Check not found.</p>
  {/if}
</div>
