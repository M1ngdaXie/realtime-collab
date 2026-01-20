import { BrowserRouter, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import ProtectedRoute from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import AuthCallback from "./pages/AuthCallback";
import EditorPage from "./pages/EditorPage.tsx";
import Home from "./pages/Home.tsx";
import KanbanPage from "./pages/KanbanPage.tsx";
import LandingPage from "./pages/LandingPage.tsx";

// Conditional component that shows LandingPage for guests, Home for authenticated users
function ConditionalHome() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  if (!user) {
    return <LandingPage />;
  }

  return <Home />;
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/auth/callback" element={<AuthCallback />} />
          <Route path="/" element={<ConditionalHome />} />
          <Route
            path="/editor/:id"
            element={
              <ProtectedRoute>
                <EditorPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/kanban/:id"
            element={
              <ProtectedRoute>
                <KanbanPage />
              </ProtectedRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
