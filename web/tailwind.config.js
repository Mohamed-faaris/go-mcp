/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ["./index.html","./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        border: "hsl(0 0% 88%)",
        background: "hsl(0 0% 98%)",
        foreground: "hsl(0 0% 7%)",
        card: "hsl(0 0% 100%)",
        'card-foreground': "hsl(0 0% 7%)",
        primary: { DEFAULT: "hsl(0 0% 7%)", foreground: "hsl(0 0% 100%)" },
        muted: { DEFAULT: "hsl(0 0% 95%)", foreground: "hsl(0 0% 42%)" },
        accent: { DEFAULT: "hsl(0 0% 95%)", foreground: "hsl(0 0% 7%)" },
      },
    },
  },
  plugins: [],
};
