<script lang="ts">
  import { checksStore } from '$lib/stores/checks';
  import CheckCard from '$lib/components/CheckCard.svelte';
  import SkeletonCard from '$lib/components/SkeletonCard.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import type { Check } from '$lib/types';

  const failing = $derived(
    $checksStore.checks.filter((c: Check) => c.status === 'down' || c.status === 'alerting')
  );
  const healthy = $derived(
    $checksStore.checks.filter((c: Check) => c.status === 'up')
  );
  const other = $derived(
    $checksStore.checks.filter((c: Check) => c.status === 'new' || c.status === 'silenced')
  );

  const counts = $derived({
    total: $checksStore.checks.length,
    ok: healthy.length,
    down: failing.length,
    silenced: other.filter((c: Check) => c.status === 'silenced').length,
    new_count: other.filter((c: Check) => c.status === 'new').length,
  });
</script>

<svelte:head>
  <title>Dashboard - cronhealth</title>
</svelte:head>

<div class="max-w-5xl mx-auto px-4 py-6">
  {#if $checksStore.loading}
    <div class="flex gap-4 mb-6">
      <div class="h-8 w-48 bg-border rounded animate-pulse"></div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      {#each Array(4) as _}
        <SkeletonCard />
      {/each}
    </div>
  {:else if counts.total === 0}
    <EmptyState
      title="No checks configured yet."
      description="Paste this into your first cron job to get started:"
    >
      <pre class="bg-surface border border-border rounded-card p-4 text-sm font-mono text-text-secondary mb-6 text-left">curl -fsS -X POST \
  http://cronhealth.internal/ping/YOUR-SLUG</pre>
      <a
        href="/checks/new"
        class="bg-status-up/20 text-status-up px-4 py-2 rounded-card font-medium hover:bg-status-up/30 transition-colors"
      >
        + Create your first check
      </a>
    </EmptyState>
  {:else}
    <!-- Status summary bar -->
    <div class="flex items-center gap-4 mb-6 text-sm" aria-live="polite">
      <span class="text-status-up font-medium">{counts.ok} OK</span>
      {#if counts.down > 0}
        <span class="text-status-down font-medium">{counts.down} DOWN</span>
      {/if}
      {#if counts.silenced > 0}
        <span class="text-status-silenced">{counts.silenced} silenced</span>
      {/if}
      {#if counts.new_count > 0}
        <span class="text-status-new">{counts.new_count} waiting</span>
      {/if}
    </div>

    <!-- Failing checks first -->
    {#if failing.length > 0}
      <section class="mb-6">
        <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Failing</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          {#each failing as check (check.id)}
            <CheckCard {check} />
          {/each}
        </div>
      </section>
    {/if}

    <!-- Healthy + other checks -->
    {#if healthy.length > 0 || other.length > 0}
      <section>
        <!-- Mobile: collapsible accordion for healthy checks -->
        <details class="md:hidden" open={failing.length === 0}>
          <summary class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide cursor-pointer">
            All Checks ({healthy.length + other.length})
          </summary>
          <div class="grid grid-cols-1 gap-3">
            {#each [...healthy, ...other] as check (check.id)}
              <CheckCard {check} />
            {/each}
          </div>
        </details>
        <!-- Desktop: always visible grid -->
        <div class="hidden md:block">
          <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">All Checks</h2>
          <div class="grid grid-cols-2 gap-3">
            {#each [...healthy, ...other] as check (check.id)}
              <CheckCard {check} />
            {/each}
          </div>
        </div>
      </section>
    {/if}
  {/if}
</div>
