/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        background: '#07090b',
        foreground: '#f7faf9',
        card: '#0c1114',
        muted: '#131a1e',
        'muted-foreground': '#9aa7aa',
        border: '#25323a',
        accent: '#172126',
        primary: '#a879ff',
        success: '#28e0a0',
        warning: '#ffb84d',
        info: '#4c9fff',
        destructive: '#ff476f',
      },
      boxShadow: {
        panel: '0 1px 0 rgba(255,255,255,.025) inset, 0 24px 70px rgba(0,0,0,.22)',
      },
    },
  },
  plugins: [],
}
