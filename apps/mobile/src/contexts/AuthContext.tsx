import React, { createContext, useContext, useState, useEffect } from 'react';
import { AuthService } from '../services';
import { User } from '../types';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: () => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    checkAuthStatus();
  }, []);

  const checkAuthStatus = async () => {
    try {
      await AuthService.initialize();
      const hasToken = await AuthService.isAuthenticated();

      if (hasToken) {
        // Proactively check and refresh token if expired
        const isValid = await AuthService.ensureValidToken();

        if (isValid) {
          const user = await AuthService.getUser();
          setUser(user);
        } else {
          // Token refresh failed - user needs to re-login
          console.log('Token refresh failed during init, clearing auth state');
          setUser(null);
        }
      }
    } catch (error) {
      console.error('Error checking auth status:', error);
      // On error, clear auth state to force re-login
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  const login = () => {
    // This will be called after successful login
    // The actual login is handled by the LoginScreen
    checkAuthStatus();
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
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
