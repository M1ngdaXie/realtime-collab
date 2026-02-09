export type ThemeMode = "light" | "dark";

export const getStoredTheme = (): ThemeMode | null => {
  try {
    const value = localStorage.getItem("theme");
    if (value === "light" || value === "dark") {
      return value;
    }
  } catch {
    // Ignore storage access errors
  }
  return null;
};

export const getPreferredTheme = (): ThemeMode => {
  const stored = getStoredTheme();
  if (stored) return stored;

  if (typeof window !== "undefined" && window.matchMedia) {
    return window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
  }

  return "light";
};

export const applyTheme = (theme: ThemeMode) => {
  if (typeof document === "undefined") return;
  document.documentElement.classList.toggle("dark", theme === "dark");

  try {
    localStorage.setItem("theme", theme);
  } catch {
    // Ignore storage access errors
  }
};
