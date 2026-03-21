<script lang="ts">
  import { auth } from '$lib/stores/auth';
  import { page } from '$app/stores';

  const navItems = [
    { href: '/', label: 'Dashboard' },
    { href: '/alerts', label: 'Alerts' },
    { href: '/settings', label: 'Settings' },
  ];

  function isActive(href: string, pathname: string): boolean {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  }
</script>

<!-- Desktop nav -->
<nav class="hidden md:flex items-center justify-between px-6 py-3 bg-surface border-b border-border">
  <div class="flex items-center gap-6">
    <a href="/" class="font-semibold text-text-primary hover:text-white">cronhealth</a>
    {#each navItems as item}
      <a
        href={item.href}
        class="text-sm {isActive(item.href, $page.url.pathname) ? 'text-text-primary' : 'text-text-secondary hover:text-text-primary'}"
      >
        {item.label}
      </a>
    {/each}
  </div>
  <div class="flex items-center gap-4">
    <a
      href="/checks/new"
      class="bg-status-up/20 text-status-up px-3 py-1.5 rounded-card text-sm font-medium hover:bg-status-up/30 transition-colors"
    >
      + New Check
    </a>
    {#if $auth.user}
      <span class="text-text-secondary text-sm">{$auth.user.email}</span>
    {/if}
  </div>
</nav>

<!-- Mobile bottom nav -->
<nav class="md:hidden fixed bottom-0 left-0 right-0 bg-surface border-t border-border flex items-center justify-around py-2 z-40">
  {#each navItems as item}
    <a
      href={item.href}
      class="flex flex-col items-center gap-0.5 px-3 py-1 text-xs {isActive(item.href, $page.url.pathname) ? 'text-text-primary' : 'text-text-secondary'}"
    >
      {item.label}
    </a>
  {/each}
  <a
    href="/checks/new"
    class="flex flex-col items-center gap-0.5 px-3 py-1 text-xs text-status-up"
  >
    + New
  </a>
</nav>
