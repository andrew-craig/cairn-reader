import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { AuthService } from '../services/auth';
import './Login.css';

type Mode = 'login' | 'register';

export default function Login() {
  const navigate = useNavigate();
  const { login } = useAuth();

  const [mode, setMode] = useState<Mode>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const switchMode = (next: Mode) => {
    setMode(next);
    setError(null);
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!email.trim() || !password.trim()) {
      setError('Please enter both email and password.');
      return;
    }

    setError(null);
    setIsLoading(true);
    try {
      const credentials = { email: email.trim(), password };
      if (mode === 'login') {
        await AuthService.loginWithEmail(credentials);
      } else {
        await AuthService.registerWithEmail(credentials);
      }
      // Refresh context state from the freshly stored session, then enter the app.
      await login();
      navigate('/read', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="login-page">
      <form className="login-card" onSubmit={handleSubmit}>
        <h1 className="login-title">Cairn</h1>
        <p className="login-subtitle">Read and discover what you love</p>

        <div className="login-tabs" role="tablist">
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'login'}
            className={mode === 'login' ? 'login-tab login-tab--active' : 'login-tab'}
            onClick={() => switchMode('login')}
          >
            Login
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'register'}
            className={mode === 'register' ? 'login-tab login-tab--active' : 'login-tab'}
            onClick={() => switchMode('register')}
          >
            Register
          </button>
        </div>

        <label className="login-label" htmlFor="email">
          Email
        </label>
        <input
          id="email"
          className="login-input"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={isLoading}
        />

        <label className="login-label" htmlFor="password">
          Password
        </label>
        <input
          id="password"
          className="login-input"
          type="password"
          autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={isLoading}
        />

        {error && (
          <p className="login-error" role="alert">
            {error}
          </p>
        )}

        <button type="submit" className="login-submit" disabled={isLoading}>
          {isLoading ? 'Please wait…' : mode === 'login' ? 'Login' : 'Create account'}
        </button>
      </form>
    </div>
  );
}
