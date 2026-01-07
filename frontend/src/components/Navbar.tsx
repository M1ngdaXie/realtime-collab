import { useAuth } from '../contexts/AuthContext';
import './Navbar.css';

function Navbar() {
  const { user, logout } = useAuth();

  if (!user) return null;

  return (
    <nav className="navbar">
      <div className="navbar-content">
        <div className="navbar-brand">
          <a href="/">Realtime Collab</a>
        </div>

        <div className="navbar-user">
          {user.avatar_url && (
            <img
              src={user.avatar_url}
              alt={user.name}
              className="user-avatar"
            />
          )}
          <span className="user-name">{user.name}</span>
          <button onClick={logout} className="logout-button">
            Logout
          </button>
        </div>
      </div>
    </nav>
  );
}

export default Navbar;
