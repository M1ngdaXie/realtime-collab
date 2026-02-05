import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';
import { LogOut } from 'lucide-react';

function Navbar() {
  const { user, logout } = useAuth();

  if (!user) return null;

  return (
    <nav className="
      bg-pixel-white
      border-b-[3px]
      border-pixel-outline
      shadow-pixel-sm
      sticky
      top-0
      z-50
    ">
      <div className="max-w-7xl mx-auto px-8 py-4 flex items-center justify-between">
        {/* Brand */}
        <a
          href="/"
          className="
            font-pixel
            text-xl
            text-pixel-purple-bright
            hover:text-pixel-cyan-bright
            transition-colors
            duration-200
          "
        >
          Realtime Collab
        </a>

        {/* User Section */}
        <div className="flex items-center gap-4">
          {user.avatar_url && (
            <img
              src={user.avatar_url}
              alt={user.name}
              className="
                w-10
                h-10
                border-[3px]
                border-pixel-outline
                shadow-pixel-sm
              "
            />
          )}
          <span className="font-sans font-medium text-pixel-text-primary hidden sm:inline">
            {user.name}
          </span>
          <Button
            onClick={logout}
            variant="outline"
            size="sm"
            className="
              border-[3px]
              border-pixel-outline
              shadow-pixel-sm
              hover:shadow-pixel-hover
              hover:-translate-y-0.5
              hover:-translate-x-0.5
              transition-all
              duration-75
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
