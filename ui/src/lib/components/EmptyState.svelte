<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    description?: string;
    children?: Snippet;
  }

  let { title, description, children }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-16 px-4 text-center">
  <h3 class="text-lg font-medium text-text-primary mb-2">{title}</h3>
  {#if description}
    <p class="text-text-secondary text-sm mb-6 max-w-md">{description}</p>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</div>
