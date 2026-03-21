<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    open: boolean;
    onclose: () => void;
    children: Snippet;
  }

  let { title, open = $bindable(), onclose, children }: Props = $props();

  let dialogEl: HTMLDialogElement | undefined = $state();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onclose();
    }
  }

  $effect(() => {
    if (open && dialogEl) {
      dialogEl.showModal();
    } else if (dialogEl?.open) {
      dialogEl.close();
    }
  });
</script>

{#if open}
  <dialog
    bind:this={dialogEl}
    class="bg-surface border border-border rounded-card p-0 text-text-primary backdrop:bg-black/60 max-w-md w-full"
    aria-labelledby="modal-title"
    aria-modal="true"
    onkeydown={handleKeydown}
    onclose={onclose}
  >
    <div class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 id="modal-title" class="text-lg font-semibold">{title}</h2>
        <button
          class="text-text-secondary hover:text-text-primary text-xl"
          onclick={onclose}
          aria-label="Close"
        >
          &times;
        </button>
      </div>
      {@render children()}
    </div>
  </dialog>
{/if}
