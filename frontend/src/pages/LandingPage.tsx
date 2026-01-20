import FloatingGem from '../components/PixelSprites/FloatingGem';
import PixelIcon from '../components/PixelIcon/PixelIcon';
import './LandingPage.css';

const API_BASE_URL = import.meta.env.VITE_API_URL || "https://docnest-backend-mingda.fly.dev/api";

function LandingPage() {
  const handleGoogleLogin = () => {
    window.location.href = `${API_BASE_URL}/auth/google`;
  };

  const handleGitHubLogin = () => {
    window.location.href = `${API_BASE_URL}/auth/github`;
  };

  return (
    <div className="landing-page">
      {/* Hero Section */}
      <section className="landing-hero">
        <FloatingGem position={{ top: '10%', left: '8%' }} delay={0} size={44} />
        <FloatingGem position={{ top: '15%', right: '12%' }} delay={1.5} size={36} />
        <FloatingGem position={{ top: '45%', left: '5%' }} delay={2.5} size={28} />
        <FloatingGem position={{ bottom: '25%', right: '8%' }} delay={3.5} size={40} />
        <FloatingGem position={{ bottom: '15%', left: '15%' }} delay={4} size={32} />
        <FloatingGem position={{ top: '60%', right: '20%' }} delay={1} size={24} />

        <div className="hero-content">
          <div className="hero-logo">
            <PixelIcon name="gem" size={56} color="var(--pixel-yellow-gold)" />
            <h1 className="hero-brand">DocNest</h1>
          </div>

          <h2 className="hero-headline">Create Together. In Real-Time.</h2>
          <p className="hero-tagline">
            Collaborative documents and Kanban boards that sync instantly.
            <br />
            Work with your team from anywhere, even offline.
          </p>

          <div className="hero-login-buttons">
            <button className="landing-login-button google" onClick={handleGoogleLogin}>
              <GoogleIcon />
              <span>Sign in with Google</span>
            </button>
            <button className="landing-login-button github" onClick={handleGitHubLogin}>
              <GitHubIcon />
              <span>Sign in with GitHub</span>
            </button>
          </div>
        </div>

        <div className="hero-scroll-hint">
          <PixelIcon name="back-arrow" size={24} style={{ transform: 'rotate(-90deg)' }} />
        </div>
      </section>

      {/* Features Section */}
      <section className="landing-features">
        <h2 className="section-title">Why DocNest?</h2>
        <div className="features-grid">
          <FeatureCard
            icon="sync-arrows"
            title="Real-Time Collaboration"
            description="See changes as they happen. Multiple cursors show who's working where."
            color="var(--pixel-cyan-bright)"
          />
          <FeatureCard
            icon="document"
            title="Rich Documents"
            description="Create formatted documents with headings, lists, and more."
            color="var(--pixel-purple-bright)"
          />
          <FeatureCard
            icon="kanban"
            title="Kanban Boards"
            description="Drag-and-drop task management with real-time updates."
            color="var(--pixel-orange-warm)"
          />
          <FeatureCard
            icon="shield"
            title="Offline Support"
            description="Your work syncs automatically when you're back online."
            color="var(--pixel-green-lime)"
          />
        </div>
      </section>

      {/* Footer CTA */}
      <footer className="landing-footer">
        <FloatingGem position={{ top: '20%', left: '10%' }} delay={0.5} size={28} />
        <FloatingGem position={{ bottom: '30%', right: '12%' }} delay={2} size={32} />

        <div className="footer-content">
          <h3 className="footer-headline">Ready to collaborate?</h3>
          <p className="footer-tagline">Join thousands of teams creating together.</p>

          <div className="footer-login-buttons">
            <button className="landing-login-button google large" onClick={handleGoogleLogin}>
              <GoogleIcon />
              <span>Get Started Free</span>
            </button>
          </div>

          <p className="footer-tech">Built with Yjs, React, and Go</p>
        </div>
      </footer>
    </div>
  );
}

function FeatureCard({ icon, title, description, color }: {
  icon: string;
  title: string;
  description: string;
  color: string;
}) {
  return (
    <div className="feature-card">
      <div className="feature-icon" style={{ color }}>
        <PixelIcon name={icon} size={48} color={color} />
      </div>
      <h3 className="feature-title">{title}</h3>
      <p className="feature-description">{description}</p>
    </div>
  );
}

function GoogleIcon() {
  return (
    <svg className="oauth-icon" width="20" height="20" viewBox="0 0 24 24">
      <path fill="#4285f4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
      <path fill="#34a853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
      <path fill="#fbbc05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
      <path fill="#ea4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
    </svg>
  );
}

function GitHubIcon() {
  return (
    <svg className="oauth-icon" width="20" height="20" viewBox="0 0 24 24">
      <path fill="currentColor" d="M12 2A10 10 0 0 0 2 12c0 4.42 2.87 8.17 6.84 9.5.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34-.46-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.6.07-.6 1 .07 1.53 1.03 1.53 1.03.87 1.52 2.34 1.07 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.55-1.11-4.55-4.92 0-1.11.38-2 1.03-2.71-.1-.25-.45-1.29.1-2.64 0 0 .84-.27 2.75 1.02.79-.22 1.65-.33 2.5-.33.85 0 1.71.11 2.5.33 1.91-1.29 2.75-1.02 2.75-1.02.55 1.35.2 2.39.1 2.64.65.71 1.03 1.6 1.03 2.71 0 3.82-2.34 4.66-4.57 4.91.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0 0 12 2z" />
    </svg>
  );
}

export default LandingPage;
