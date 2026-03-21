<script lang="ts">
  import { auth } from '$lib/stores/auth';
  import { toasts } from '$lib/stores/toast';
</script>

<svelte:head>
  <title>Settings - cronhealth</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-6">
  <h1 class="text-2xl font-semibold mb-6">Settings</h1>

  <section class="mb-8">
    <h2 class="text-sm font-medium text-text-secondary mb-3 uppercase tracking-wide">Profile</h2>
    <div class="bg-surface border border-border rounded-card p-4">
      {#if $auth.user}
        <div class="text-sm">
          <div class="flex justify-between py-2">
            <span class="text-text-secondary">Email</span>
            <span class="font-mono">{$auth.user.email}</span>
          </div>
          <div class="flex justify-between py-2">
            <span class="text-text-secondary">User ID</span>
            <span class="font-mono text-xs text-text-secondary">{$auth.user.user_id}</span>
          </div>
        </div>
      {/if}
    </div>
  </section>

  <section>
    <form
      method="POST"
      action="/auth/logout"
      onsubmit={() => {
        auth.logout();
        toasts.info('Logged out');
      }}
    >
      <button
        type="submit"
        class="border border-border rounded-card px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors"
      >
        Log out
      </button>
    </form>
  </section>
</div>
