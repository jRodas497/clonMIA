/** @type {import('tailwindcss').Config} */
import plugin from 'flowbite/plugin'
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
    'node_modules/flowbite-react/lib/esm/**/*.js'
  ],
  theme: {
    extend: {
      colors: {
        'btn': '#678c99',
        'btn-osc': '#405861',
      }
    },
  },
  plugins: [
    plugin
  ]

}

