import { Github } from 'lucide-react';
import { Button } from '@/shared/components/ui/button';
import { loginWithGitHub } from '@/shared/services/api';

function Login() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
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
          <h1 className="text-3xl font-bold text-white mb-2">
            Selfhostly
          </h1>
          <p className="text-slate-400">
            Deploy and manage your self-hosted applications
          </p>
        </div>

        {/* Login card */}
        <div className="bg-slate-800/50 backdrop-blur-xl rounded-2xl border border-slate-700/50 p-8 shadow-2xl">
          <h2 className="text-xl font-semibold text-white text-center mb-6">
            Sign in to continue
          </h2>

          <Button
            onClick={loginWithGitHub}
            className="w-full h-12 bg-slate-700 hover:bg-slate-600 text-white border border-slate-600 rounded-xl transition-all duration-200 flex items-center justify-center gap-3"
          >
            <Github className="w-5 h-5" />
            <span>Continue with GitHub</span>
          </Button>

          <p className="text-slate-500 text-sm text-center mt-6">
            Sign in with your GitHub account to access the dashboard
          </p>
        </div>

        {/* Footer */}
        <p className="text-slate-600 text-sm text-center mt-8">
          Secure authentication powered by GitHub OAuth
        </p>
      </div>
    </div>
  );
}

export default Login;
