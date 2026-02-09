import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';
import { LogOut } from 'lucide-react';
import DocNestLogo from '@/assets/docnest/docnest-icon-128.png';
import ThemeToggle from '@/components/ThemeToggle';

function Navbar() {
  const { user, logout } = useAuth();

  if (!user) return null;

  return (
    <nav className="
      bg-surface
      border-b
      border-border
      shadow-soft
      sticky
      top-0
      z-50
      backdrop-blur
    ">
      <div className="max-w-7xl mx-auto px-8 py-4 flex items-center justify-between">
        {/* Brand */}
        <a
          href="/"
          className="
            font-display
            text-lg
            text-text-primary
            hover:text-brand
            transition-colors
            duration-200
            flex
            items-center
            gap-2
          "
        >
          <img
            src={DocNestLogo}
            alt="DocNest"
            className="w-6 h-6"
            style={{ imageRendering: 'pixelated' }}
          />
          DocNest
        </a>

        {/* User Section */}
        <div className="flex items-center gap-4">
          <ThemeToggle className="shadow-soft hover:shadow-card transition-all duration-150" />
          {user.avatar_url && (
            <img
              src={user.avatar_url}
              alt={user.name}
              className="
                w-9
                h-9
                rounded-full
                border
                border-border
                shadow-soft
                object-cover
              "
            />
          )}
          <span className="font-sans font-medium text-text-primary hidden sm:inline">
            {user.name}
          </span>
          <Button
            onClick={logout}
            variant="outline"
            size="sm"
            className="
              border
              border-border
              shadow-soft
              hover:shadow-card
              hover:-translate-y-0.5
              transition-all
              duration-150
              font-sans
            "
          >
            <LogOut className="w-4 h-4 sm:mr-2" />
            <span className="hidden sm:inline">Logout</span>
          </Button>
        </div>
      </div>
    </nav>
  );
}

export default Navbar;
