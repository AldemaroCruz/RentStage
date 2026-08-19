"use client";

import { useEffect, useState } from "react";
import {
  oppositeTheme,
  resolveTheme,
  storedTheme,
  THEME_STORAGE_KEY,
  type Theme,
} from "@/lib/theme";

function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

export function ThemeToggle({ className = "" }: { className?: string }) {
  const [theme, setTheme] = useState<Theme>("light");

  useEffect(() => {
    const systemDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    const active = storedTheme(document.documentElement.dataset.theme)
      ?? resolveTheme(window.localStorage.getItem(THEME_STORAGE_KEY), systemDark);
    applyTheme(active);
    setTheme(active);
  }, []);

  function toggle() {
    const next = oppositeTheme(theme);
    window.localStorage.setItem(THEME_STORAGE_KEY, next);
    applyTheme(next);
    setTheme(next);
  }

  const nextLabel = theme === "dark" ? "claro" : "oscuro";
  return (
    <button
      type="button"
      className={`icon-button theme-toggle ${className}`.trim()}
      onClick={toggle}
      aria-label={`Cambiar a modo ${nextLabel}`}
      title={`Cambiar a modo ${nextLabel}`}
    >
      <svg className="theme-icon theme-icon-sun" width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.65 17.65l1.42 1.42M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.65 6.35l1.42-1.42" />
      </svg>
      <svg className="theme-icon theme-icon-moon" width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M21 12.8A8.5 8.5 0 1 1 11.2 3 6.5 6.5 0 0 0 21 12.8z" />
      </svg>
    </button>
  );
}
