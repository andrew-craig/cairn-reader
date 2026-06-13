import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { AuthService } from '../services/auth';
import './Account.css';

export default function Account() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [isChanging, setIsChanging] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  const handleChangePassword = async (event: FormEvent) => {
    event.preventDefault();
    if (!currentPassword.trim() || !newPassword.trim()) {
      setPasswordError('Please fill in both password fields.');
      return;
    }
    setPasswordError(null);
    setPasswordSuccess(false);
    setIsChanging(true);
    try {
      await AuthService.changePassword(currentPassword, newPassword);
      setPasswordSuccess(true);
      setCurrentPassword('');
      setNewPassword('');
    } catch (err) {
      setPasswordError(err instanceof Error ? err.message : 'Failed to change password.');
    } finally {
      setIsChanging(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Error logging out:', error);
    }
    navigate('/login', { replace: true });
  };

  return (
    <div className="account-page">
      <p className="account-email">{user?.email ?? 'Anonymous user'}</p>

      <hr className="account-divider" />

      <form className="account-section" onSubmit={handleChangePassword}>
        <h2 className="account-section-title">Change password</h2>

        <label className="account-label" htmlFor="current-password">
          Current password
        </label>
        <input
          id="current-password"
          className="account-input"
          type="password"
          autoComplete="current-password"
          value={currentPassword}
          onChange={(e) => {
            setCurrentPassword(e.target.value);
            if (passwordError) setPasswordError(null);
            if (passwordSuccess) setPasswordSuccess(false);
          }}
          disabled={isChanging}
        />

        <label className="account-label" htmlFor="new-password">
          New password
        </label>
        <input
          id="new-password"
          className="account-input"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => {
            setNewPassword(e.target.value);
            if (passwordError) setPasswordError(null);
            if (passwordSuccess) setPasswordSuccess(false);
          }}
          disabled={isChanging}
        />

        {passwordError && (
          <p className="account-error" role="alert">
            {passwordError}
          </p>
        )}
        {passwordSuccess && (
          <p className="account-success" role="status">
            Password changed successfully.
          </p>
        )}

        <button type="submit" className="account-btn account-btn--primary" disabled={isChanging}>
          {isChanging ? 'Saving…' : 'Change password'}
        </button>
      </form>

      <hr className="account-divider" />

      <button type="button" className="account-btn account-btn--danger" onClick={handleLogout}>
        Log out
      </button>
    </div>
  );
}
