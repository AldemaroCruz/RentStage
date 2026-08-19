export const THEME_STORAGE_KEY = "rentstage-theme";

export type Theme = "light" | "dark";

export function storedTheme(value: string | null | undefined): Theme | null {
  return value === "light" || value === "dark" ? value : null;
}

export function resolveTheme(value: string | null | undefined, systemPrefersDark: boolean): Theme {
  return storedTheme(value) ?? (systemPrefersDark ? "dark" : "light");
}

export function oppositeTheme(theme: Theme): Theme {
  return theme === "dark" ? "light" : "dark";
}

export function themeBootstrapScript(): string {
  return `(function(){try{var k=${JSON.stringify(THEME_STORAGE_KEY)};var v=localStorage.getItem(k);var d=v==='dark'||v==='light'?v:(matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light');document.documentElement.dataset.theme=d;document.documentElement.style.colorScheme=d;}catch(e){document.documentElement.dataset.theme='light';}})();`;
}
