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
        primary: { DEFAULT: '#8B4FB9', foreground: '#FFFFFF' },
        secondary: { DEFAULT: '#00D9FF', foreground: '#2B1B38' },
        accent: { DEFAULT: '#FF6EC7', foreground: '#FFFFFF' },
        muted: { DEFAULT: '#F5F0FF', foreground: '#4A3B5C' },

        // Custom pixel palette
        pixel: {
          purple: { deep: '#4A1B6F', bright: '#8B4FB9' },
          cyan: { bright: '#00D9FF' },
          pink: { vibrant: '#FF6EC7' },
          orange: { warm: '#FF8E3C' },
          yellow: { gold: '#FFD23F' },
          green: { lime: '#8EF048', forest: '#3FA54D' },
          bg: { dark: '#2B1B38', medium: '#4A3B5C', light: '#E8D9F3' },
          panel: '#F5F0FF',
          white: '#FFFFFF',
          shadow: { dark: '#1A0E28' },
          outline: '#2B1B38',
          text: { primary: '#2B1B38', secondary: '#4A3B5C', muted: '#8B7B9C' },
        },
      },
      fontFamily: {
        pixel: ['"Press Start 2P"', 'cursive'],
        sans: ['-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'Roboto', 'sans-serif'],
      },
      boxShadow: {
        'pixel-sm': '3px 3px 0 #1A0E28',
        'pixel': '4px 4px 0 #1A0E28, 4px 4px 0 3px #2B1B38',
        'pixel-md': '6px 6px 0 #1A0E28, 6px 6px 0 3px #2B1B38',
        'pixel-lg': '8px 8px 0 #1A0E28, 8px 8px 0 3px #2B1B38',
        'pixel-hover': '5px 5px 0 #1A0E28',
        'pixel-active': '2px 2px 0 #1A0E28',
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
        DEFAULT: '0',
        lg: '0',
        md: '0',
        sm: '0',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}
