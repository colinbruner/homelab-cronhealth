<script lang="ts">
  import '../app.css';
  import { onMount, onDestroy } from 'svelte';
  import { auth } from '$lib/stores/auth';
  import { checksStore } from '$lib/stores/checks';
  import { connectSSE, disconnectSSE, onSSEEvent } from '$lib/sse';
  import Nav from '$lib/components/Nav.svelte';
  import ToastContainer from '$lib/components/ToastContainer.svelte';

  let { children } = $props();

  let unsubSSE: (() => void) | undefined;

  onMount(async () => {
    await auth.load();
    await checksStore.load();
    connectSSE();
    unsubSSE = onSSEEvent((payload) => {
      checksStore.updateCheckStatus(payload.check_id, payload.status);
    });
  });

  onDestroy(() => {
    disconnectSSE();
    unsubSSE?.();
  });
</script>

{#if $auth.loading}
  <div class="flex items-center justify-center h-screen">
    <span class="text-text-secondary">Loading...</span>
  </div>
{:else}
  <Nav />
  <main class="pb-16 md:pb-0" role="main">
    {@render children()}
  </main>
  <ToastContainer />
{/if}
