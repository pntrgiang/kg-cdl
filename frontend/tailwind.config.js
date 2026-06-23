/** @type {import('tailwindcss').Config} */
export default {
  theme: {
    extend: {
      colors: {
        // Tông tím chủ đạo (lấy từ lapel áo outfit.png)
        brand: {
          50: '#f4f1fb',
          100: '#e7e0f6',
          200: '#d0c3ee',
          300: '#b09ce1',
          400: '#8a6fd0',
          500: '#6b4cbf',
          600: '#573aa6',
          700: '#472f85',
          800: '#3b2a78', // tím chính
          900: '#2a1d5c', // tím đậm
          950: '#1b1240',
        },
        // Điểm nhấn vàng gold (logo KG)
        gold: {
          400: '#d8be7e',
          500: '#c6a15b',
          600: '#a9863f',
        },
      },
      fontFamily: {
        serif: ['Georgia', 'Cambria', 'serif'],
      },
    },
  },
}
