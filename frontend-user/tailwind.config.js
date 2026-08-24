/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        paper: "#F4EDE1",
        ink: "#2A2118",
        terracotta: "#C45C26",
        leaf: "#3F6B4A",
        clay: "#E8D2B5",
        soot: "#5C5146",
        chili: "#B33A2B",
        yolk: "#E0A43A",
      },
      fontFamily: {
        display: ['"Noto Serif SC"', "serif"],
        sans: ['"Noto Sans SC"', "ui-sans-serif", "system-ui"],
      },
      boxShadow: {
        ticket: "0 10px 30px rgba(42,33,24,0.12)",
        lift: "0 18px 40px rgba(196,92,38,0.18)",
      },
    },
  },
  plugins: [],
};
