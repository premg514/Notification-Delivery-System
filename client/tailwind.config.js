/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        paper: "#f5f7fb",
        ink: "#0f172f",
        coral: "#ff4530",
        sand: "#f8efe7",
        mint: "#d8ffea",
        amber: "#fff2c7",
      },
      boxShadow: {
        panel: "0 18px 50px rgba(15, 23, 47, 0.08)",
        soft: "0 10px 30px rgba(255, 69, 48, 0.12)",
      },
      borderRadius: {
        "4xl": "2rem",
      },
      fontFamily: {
        sans: ["Segoe UI Variable", "Aptos", "Trebuchet MS", "sans-serif"],
        display: ["Arial Black", "Segoe UI Variable Display", "sans-serif"],
      },
      backgroundImage: {
        "hero-glow":
          "radial-gradient(circle at top left, rgba(255,69,48,0.18), transparent 32%), radial-gradient(circle at top right, rgba(255,208,157,0.36), transparent 28%)",
      },
    },
  },
  plugins: [],
};
