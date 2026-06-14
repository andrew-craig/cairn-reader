import {
  createContext,
  useContext,
  useState,
  useEffect,
  type ReactNode,
} from 'react';
import { type User } from '@cairn/shared';
import { loadServerUrl } from '../config/api';
import { AuthService } from '../services/auth';

// Same shape as the mobile AuthContext (web_app_requirements.md §Auth state):
// user, isAuthenticated, isLoading, login, logout, plus the "clear tokens →
// broadcast logout" listener that resets state when a service wipes tokens.
interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    void checkAuthStatus();

    // Reset local state whenever tokens are cleared elsewhere (e.g. a failed
    // refresh inside fetchWithAuth), which redirects the user back to /login.
    const unsubscribe = AuthService.onAuthStateChange((isAuthenticated) => {
      if (!isAuthenticated) {
        setUser(null);
      }
    });

    return unsubscribe;
  }, []);

  const checkAuthStatus = async () => {
    try {
      await loadServerUrl();
      await AuthService.initialize();
      const hasToken = await AuthService.isAuthenticated();

      if (hasToken) {
        // Proactively refresh an expired/expiring token before trusting the session.
        const isValid = await AuthService.ensureValidToken();
        setUser(isValid ? await AuthService.getUser() : null);
      } else {
        // No token: ensure any prior user state is cleared (keeps re-runs idempotent).
        setUser(null);
      }
    } catch (error) {
      console.error('Error checking auth status:', error);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  // Called by the Login screen after AuthService stores tokens; re-reads the
  // persisted session so isAuthenticated flips to true.
  const login = async () => {
    await checkAuthStatus();
  };

  const logout = async () => {
    try {
      await AuthService.logout();
      setUser(null);
    } catch (error) {
      console.error('Error during logout:', error);
      throw error;
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: user !== null,
        isLoading,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
