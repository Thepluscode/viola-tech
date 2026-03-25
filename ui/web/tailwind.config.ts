import type { Config } from "tailwindcss";
import animate from "tailwindcss-animate";

const config: Config = {
  darkMode: ["class"],
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        "viola-bg": "#0a0f1e",
        "viola-surface": "#0d1526",
        "viola-border": "#1e2d4a",
        "viola-accent": "#00d4ff",
        "viola-text": "#e2e8f0",
        "viola-muted": "#94a3b8",
        "severity-critical": "#ef4444",
        "severity-critical-bg": "#450a0a",
        "severity-critical-border": "#991b1b",
        "severity-high": "#f97316",
        "severity-high-bg": "#431407",
        "severity-high-border": "#9a3412",
        "severity-medium": "#eab308",
        "severity-medium-bg": "#422006",
        "severity-medium-border": "#854d0e",
        "severity-low": "#3b82f6",
        "severity-low-bg": "#172554",
        "severity-low-border": "#1e40af",
        "status-open": "#00d4ff",
        "status-open-bg": "#0c1f2e",
        "status-open-border": "#164e63",
        "status-ack": "#eab308",
        "status-ack-bg": "#422006",
        "status-ack-border": "#854d0e",
        "status-closed": "#64748b",
        "status-closed-bg": "#1e293b",
        "status-closed-border": "#334155",
      },
      fontFamily: {
        mono: ["JetBrains Mono", "Fira Code", "Consolas", "monospace"],
      },
      borderRadius: {
        DEFAULT: "0.375rem",
      },
      backgroundImage: {
        "gradient-radial": "radial-gradient(var(--tw-gradient-stops))",
      },
    },
  },
  plugins: [animate],
};

export default config;
