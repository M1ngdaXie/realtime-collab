import { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

function AuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { login } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const processedRef = useRef(false);

  useEffect(() => {
    if (processedRef.current) return;
    processedRef.current = true;

    const handleCallback = async () => {
      const token = searchParams.get('token');
      const redirect =
        searchParams.get('redirect') ||
        sessionStorage.getItem('oauth_redirect') ||
        '/';
      sessionStorage.removeItem('oauth_redirect');

      if (!token) {
        setError('No authentication token received');
        setTimeout(() => navigate('/login'), 2000);
        return;
      }

      try {
        await login(token);
        navigate(redirect);
      } catch (err) {
        console.error('Login error:', err);
        setError('Authentication failed. Please try again.');
        setTimeout(() => navigate('/login'), 2000);
      }
    };

    handleCallback();
  }, [searchParams, login, navigate]);

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    }}>
      <div style={{
        background: 'white',
        borderRadius: '16px',
        padding: '48px',
        textAlign: 'center',
      }}>
        {error ? (
          <>
            <h2 style={{ color: '#e53e3e', marginBottom: '8px' }}>Error</h2>
            <p style={{ color: '#718096' }}>{error}</p>
            <p style={{ color: '#718096', fontSize: '14px', marginTop: '16px' }}>
              Redirecting to login...
            </p>
          </>
        ) : (
          <>
            <h2 style={{ color: '#1a202c', marginBottom: '8px' }}>Logging you in...</h2>
            <p style={{ color: '#718096' }}>Please wait while we complete the authentication.</p>
          </>
        )}
      </div>
    </div>
  );
}

export default AuthCallback;
