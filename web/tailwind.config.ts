import type { Config } from 'tailwindcss'

export default {
  darkMode: ['class'],
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        deep: '#0E0E0E',
        mist: '#F5F5F7',
        plasma: '#007AFF',
        violet: '#7B61FF',
        electric: '#00E0FF'
      },
      borderRadius: {
        '2xl': '1rem'
      },
      backgroundImage: {
        auraGradient: 'linear-gradient(135deg, #007AFF, #7B61FF)'
      }
    }
  },
  plugins: []
} satisfies Config
