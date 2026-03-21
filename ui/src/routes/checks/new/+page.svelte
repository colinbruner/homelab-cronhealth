<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { toasts } from '$lib/stores/toast';
  import { checksStore } from '$lib/stores/checks';

  let name = $state('');
  let periodMinutes = $state(5);
  let graceMinutes = $state(5);
  let submitting = $state(false);
  let error = $state('');

  async function handleSubmit() {
    error = '';
    if (!name.trim()) {
      error = 'Name is required';
      return;
    }
    if (periodMinutes < 1) {
      error = 'Period must be at least 1 minute';
      return;
    }

    submitting = true;
    try {
      const check = await api.createCheck({
        name: name.trim(),
        period_seconds: periodMinutes * 60,
        grace_seconds: graceMinutes * 60,
      });
      checksStore.addCheck(check);
      toasts.success('Check created');
      goto(`/checks/${check.id}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to create check';
    }
    submitting = false;
  }
</script>

<svelte:head>
  <title>New Check - cronhealth</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">New Check</h1>

  <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
    <div>
      <label for="name" class="block text-sm text-text-secondary mb-1">Name</label>
      <input
        id="name"
        type="text"
        bind:value={name}
        placeholder="e.g. nightly-backup"
        class="w-full bg-surface border border-border rounded-card px-3 py-2 text-sm text-text-primary placeholder:text-text-secondary/50 focus:outline-none focus:border-status-up/50"
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
      <p class="text-xs text-text-secondary mt-1">How often should this job ping?</p>
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
      <p class="text-xs text-text-secondary mt-1">How long to wait after a missed ping before alerting.</p>
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
        {submitting ? 'Creating...' : 'Create Check'}
      </button>
      <a
        href="/"
        class="border border-border rounded-card px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors"
      >
        Cancel
      </a>
    </div>
  </form>
</div>
