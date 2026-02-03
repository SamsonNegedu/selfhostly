import { useEffect, useState } from 'react';
import { Github, AlertCircle, Shield, X, Server } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
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
    <div className="h-screen overflow-hidden bg-gradient-to-br from-background via-muted/20 to-background flex items-center justify-center p-3 sm:p-4">
      <div className="w-full max-w-md max-h-full overflow-y-auto scrollbar-hide">
        {/* Error Alert */}
        {showError && errorMessage && (
          <div className="mb-4 sm:mb-6 bg-destructive/10 border border-destructive/50 rounded-xl p-3 sm:p-4 backdrop-blur-sm fade-in">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-destructive font-semibold mb-1">Access Denied</h3>
                <p className="text-destructive/90 text-sm">{errorMessage}</p>
                {errorMessage.includes('not authorized') && (
                  <div className="mt-2 sm:mt-3 p-2.5 sm:p-3 bg-muted/50 rounded-lg border border-border">
                    <div className="flex items-center gap-2 text-muted-foreground text-xs mb-1.5 sm:mb-2">
                      <Shield className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
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
        <div className="text-center mb-6 sm:mb-8">
          <div className="inline-flex items-center justify-center w-14 h-14 sm:w-16 sm:h-16 rounded-2xl bg-primary mb-3 sm:mb-4">
            <Server className="w-7 h-7 sm:w-8 sm:h-8 text-primary-foreground" />
          </div>
          <h1 className="text-2xl sm:text-3xl font-bold text-foreground mb-2">
            Selfhostly
          </h1>
          <p className="text-sm sm:text-base text-muted-foreground">
            Deploy and manage your self-hosted applications
          </p>
        </div>

        {/* Login card */}
        <div className="bg-card/50 backdrop-blur-xl rounded-2xl border border-border p-6 sm:p-8 shadow-xl">
          <h2 className="text-lg sm:text-xl font-semibold text-foreground text-center mb-5 sm:mb-6">
            Sign in to continue
          </h2>

          <Button
            onClick={loginWithGitHub}
            className="w-full h-11 sm:h-12 bg-muted hover:bg-muted/80 text-foreground border border-border rounded-xl transition-all duration-200 flex items-center justify-center gap-3 text-sm sm:text-base"
          >
            <Github className="w-5 h-5" />
            <span>Continue with GitHub</span>
          </Button>

          <p className="text-muted-foreground text-xs sm:text-sm text-center mt-5 sm:mt-6">
            Sign in with your GitHub account to access the dashboard
          </p>
        </div>

        {/* Footer */}
        <p className="text-muted-foreground text-xs sm:text-sm text-center mt-6 sm:mt-8 mb-3 sm:mb-0">
          Secure authentication powered by GitHub OAuth
        </p>
      </div>
    </div>
  );
}

export default Login;
