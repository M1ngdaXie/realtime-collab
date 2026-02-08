/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // shadcn system colors
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: { DEFAULT: 'hsl(var(--primary))', foreground: 'hsl(var(--primary-foreground))' },
        secondary: { DEFAULT: 'hsl(var(--secondary))', foreground: 'hsl(var(--secondary-foreground))' },
        accent: { DEFAULT: 'hsl(var(--accent))', foreground: 'hsl(var(--accent-foreground))' },
        muted: { DEFAULT: 'hsl(var(--muted))', foreground: 'hsl(var(--muted-foreground))' },
        surface: {
          DEFAULT: 'hsl(var(--surface))',
          muted: 'hsl(var(--surface-muted))',
        },
        brand: {
          DEFAULT: 'hsl(var(--brand))',
          dark: 'hsl(var(--brand-dark))',
          teal: 'hsl(var(--brand-teal))',
          tealDark: 'hsl(var(--brand-teal-dark))',
        },
        text: {
          primary: 'hsl(var(--text-primary))',
          secondary: 'hsl(var(--text-secondary))',
          muted: 'hsl(var(--text-muted))',
        },

        // Legacy pixel palette aliases (mapped to modern tokens)
        pixel: {
          purple: { deep: '#1E40AF', bright: '#2563EB' },
          cyan: { bright: '#14B8A6' },
          pink: { vibrant: '#38BDF8' },
          orange: { warm: '#F59E0B' },
          yellow: { gold: '#FBBF24' },
          green: { lime: '#22C55E', forest: '#16A34A' },
          bg: { dark: '#0F172A', medium: '#1E293B', light: '#F5F7FB' },
          panel: '#FFFFFF',
          white: '#FFFFFF',
          shadow: { dark: 'rgba(15, 23, 42, 0.2)' },
          outline: '#E2E8F0',
          text: { primary: '#0F172A', secondary: '#334155', muted: '#64748B' },
        },
      },
      fontFamily: {
        display: ['"Manrope"', 'Inter', 'sans-serif'],
        sans: ['"Inter"', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'Roboto', 'sans-serif'],
      },
      boxShadow: {
        'soft': 'var(--shadow-sm)',
        'card': 'var(--shadow-md)',
        'float': 'var(--shadow-lg)',
      },
      keyframes: {
        'pixel-float': {
          '0%, 100%': { transform: 'translateY(0) rotate(0deg)' },
          '25%': { transform: 'translateY(-12px) rotate(2deg)' },
          '50%': { transform: 'translateY(-8px) rotate(0deg)' },
          '75%': { transform: 'translateY(-12px) rotate(-2deg)' },
        },
        'pixel-bounce': {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-16px)' },
        },
      },
      animation: {
        'pixel-float': 'pixel-float 6s ease-in-out infinite',
        'pixel-bounce': 'pixel-bounce 1s ease-in-out infinite',
      },
      borderRadius: {
        DEFAULT: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        md: 'var(--radius-md)',
        sm: 'var(--radius-sm)',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}
