<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    date: string | null;
    fallback?: string;
  }

  let { date, fallback = 'never' }: Props = $props();

  let display = $state(fallback);

  function format(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const seconds = Math.floor(diff / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ${minutes % 60}m ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h ago`;
  }

  function tick() {
    if (date) display = format(date);
  }

  onMount(() => {
    tick();
    const interval = setInterval(tick, 15000);
    return () => clearInterval(interval);
  });
</script>

{#if date}
  <time
    datetime={date}
    title={new Date(date).toLocaleString()}
    class="font-mono text-text-secondary"
  >
    {display}
  </time>
{:else}
  <span class="text-text-secondary">{fallback}</span>
{/if}
