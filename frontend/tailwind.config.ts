import type { Config } from "tailwindcss";
const config: Config = {
  darkMode: ["class"], content: ["./src/**/*.{ts,tsx}"],
  theme: { extend: { colors: { background: "hsl(var(--background))", foreground: "hsl(var(--foreground))", card: "hsl(var(--card))", primary: "hsl(var(--primary))", muted: "hsl(var(--muted))", border: "hsl(var(--border))", accent: "hsl(var(--accent))" }, borderRadius: { xl: "var(--radius)" }, boxShadow: { soft: "0 16px 50px rgba(36, 54, 43, .06)" } } },
  plugins: [],
};
export default config;
