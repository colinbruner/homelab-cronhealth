/** @type {import('tailwindcss').Config} */
export default {
  theme: {
    extend: {
      colors: {
        bg: '#0f1117',
        surface: '#1a1d24',
        border: '#2a2d36',
        'text-primary': '#e2e8f0',
        'text-secondary': '#94a3b8',
        status: {
          up: '#22c55e',
          down: '#ef4444',
          silenced: '#6b7280',
          new: '#f59e0b',
          alerting: '#f97316',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      fontSize: {
        body: '14px',
      },
      borderRadius: {
        card: '4px',
        badge: '2px',
      },
    },
  },
};
