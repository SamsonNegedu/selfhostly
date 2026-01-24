import { useEffect, useState } from 'react';
import { Github, AlertCircle, Shield, X } from 'lucide-react';
import { Button } from '@/shared/components/ui/button';
import { loginWithGitHub } from '@/shared/services/api';
import { useSearchParams } from 'react-router-dom';

function Login() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [showError, setShowError] = useState(false);

  useEffect(() => {
    // Check for error parameter in URL (from OAuth callback)
    const error = searchParams.get('error');
    const errorDescription = searchParams.get('error_description');

    if (error) {
      // Set appropriate error message based on error type
      if (error === 'access_denied' || errorDescription?.includes('whitelist') || errorDescription?.includes('not authorized')) {
        setErrorMessage('Access denied: Your GitHub account is not authorized to access this system.');
      } else if (error === 'unauthorized') {
        setErrorMessage('Authentication failed: You are not authorized to access this system.');
      } else {
        setErrorMessage(`Authentication failed: ${errorDescription || 'Please try again'}`);
      }
      setShowError(true);

      // Clear error params from URL
      searchParams.delete('error');
      searchParams.delete('error_description');
      setSearchParams(searchParams, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  const dismissError = () => {
    setShowError(false);
    setTimeout(() => setErrorMessage(null), 300);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-muted/20 to-background flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Error Alert */}
        {showError && errorMessage && (
          <div className="mb-6 bg-destructive/10 border border-destructive/50 rounded-xl p-4 backdrop-blur-sm fade-in">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-destructive font-semibold mb-1">Access Denied</h3>
                <p className="text-destructive/90 text-sm">{errorMessage}</p>
                {errorMessage.includes('not authorized') && (
                  <div className="mt-3 p-3 bg-muted/50 rounded-lg border border-border">
                    <div className="flex items-center gap-2 text-muted-foreground text-xs mb-2">
                      <Shield className="w-4 h-4" />
                      <span className="font-medium">Security Notice</span>
                    </div>
                    <p className="text-muted-foreground text-xs leading-relaxed">
                      Only specific GitHub accounts are authorized to access this system.
                      If you believe you should have access, contact your system administrator
                      to add your GitHub username to whitelist.
                    </p>
                  </div>
                )}
              </div>
              <button
                onClick={dismissError}
                className="text-destructive hover:text-destructive/80 transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}

        {/* Logo and title */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-emerald-400 to-cyan-500 mb-4">
            <svg
              className="w-8 h-8 text-white"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
              />
            </svg>
          </div>
          <h1 className="text-3xl font-bold text-foreground mb-2">
            Selfhostly
          </h1>
          <p className="text-muted-foreground">
            Deploy and manage your self-hosted applications
          </p>
        </div>

        {/* Login card */}
        <div className="bg-card/50 backdrop-blur-xl rounded-2xl border border-border p-8 shadow-xl">
          <h2 className="text-xl font-semibold text-foreground text-center mb-6">
            Sign in to continue
          </h2>

          <Button
            onClick={loginWithGitHub}
            className="w-full h-12 bg-muted hover:bg-muted/80 text-foreground border border-border rounded-xl transition-all duration-200 flex items-center justify-center gap-3"
          >
            <Github className="w-5 h-5" />
            <span>Continue with GitHub</span>
          </Button>

          <p className="text-muted-foreground text-sm text-center mt-6">
            Sign in with your GitHub account to access the dashboard
          </p>
        </div>

        {/* Footer */}
        <p className="text-muted-foreground text-sm text-center mt-8">
          Secure authentication powered by GitHub OAuth
        </p>
      </div>
    </div>
  );
}

export default Login;
