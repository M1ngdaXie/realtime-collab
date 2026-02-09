import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";
import { applyTheme, getPreferredTheme, type ThemeMode } from "@/lib/theme";
import { Button } from "@/components/ui/button";

type ThemeToggleProps = {
  className?: string;
  size?: "sm" | "default" | "icon";
};

function ThemeToggle({ className, size = "icon" }: ThemeToggleProps) {
  const [theme, setTheme] = useState<ThemeMode>(() => getPreferredTheme());

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  const nextTheme: ThemeMode = theme === "dark" ? "light" : "dark";

  return (
    <Button
      type="button"
      variant="outline"
      size={size}
      onClick={() => setTheme(nextTheme)}
      className={className}
      aria-label={`Switch to ${nextTheme} mode`}
      title={`Switch to ${nextTheme} mode`}
    >
      {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
    </Button>
  );
}

export default ThemeToggle;
