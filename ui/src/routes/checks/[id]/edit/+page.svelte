<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';
  import type { Check } from '$lib/types';

  let check: Check | null = $state(null);
  let name = $state('');
  let periodMinutes = $state(5);
  let graceMinutes = $state(5);
  let submitting = $state(false);
  let loading = $state(true);
  let error = $state('');

  const checkId = $derived($page.params.id);

  onMount(async () => {
    try {
      check = await api.getCheck(checkId);
      if (check) {
        name = check.name;
        periodMinutes = Math.round(check.period_seconds / 60);
        graceMinutes = Math.round(check.grace_seconds / 60);
      }
    } catch {
      toasts.error('Failed to load check');
    }
    loading = false;
  });

  async function handleSubmit() {
    error = '';
    if (!name.trim()) {
      error = 'Name is required';
      return;
    }

    submitting = true;
    try {
      const updated = await api.updateCheck(checkId, {
        name: name.trim(),
        period_seconds: periodMinutes * 60,
        grace_seconds: graceMinutes * 60,
      });
      if (updated) checksStore.updateCheck(updated);
      toasts.success('Check updated');
      goto(`/checks/${checkId}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to update check';
    }
    submitting = false;
  }
</script>

<svelte:head>
  <title>Edit {check?.name ?? 'Check'} - cronhealth</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">Edit Check</h1>

  {#if loading}
    <div class="animate-pulse space-y-4">
      <div class="h-10 w-full bg-border rounded"></div>
      <div class="h-10 w-full bg-border rounded"></div>
      <div class="h-10 w-full bg-border rounded"></div>
    </div>
  {:else if check}
    <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
      <div>
        <label for="name" class="block text-sm text-text-secondary mb-1">Name</label>
        <input
          id="name"
          type="text"
          bind:value={name}
          class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
        />
      </div>

      <div>
        <label for="period" class="block text-sm text-text-secondary mb-1">Expected period (minutes)</label>
        <input
          id="period"
          type="number"
          bind:value={periodMinutes}
          min="1"
          inputmode="numeric"
          class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
        />
      </div>

      <div>
        <label for="grace" class="block text-sm text-text-secondary mb-1">Grace period (minutes)</label>
        <input
          id="grace"
          type="number"
          bind:value={graceMinutes}
          min="1"
          inputmode="numeric"
          class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-status-up/50"
        />
      </div>

      {#if error}
        <p class="text-status-down text-sm">{error}</p>
      {/if}

      <div class="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={submitting}
          class="bg-status-up/20 text-status-up px-4 py-2 rounded-card text-sm font-medium hover:bg-status-up/30 transition-colors disabled:opacity-50"
        >
          {submitting ? 'Saving...' : 'Save Changes'}
        </button>
        <a
          href="/checks/{checkId}"
          class="border border-border rounded-card px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors"
        >
          Cancel
        </a>
      </div>
    </form>
  {:else}
    <p class="text-text-secondary">Check not found.</p>
  {/if}
</div>
