import { useState, type FormEvent } from 'react';
import { ArrowLeft } from 'lucide-react';
import AuthLayout from '@/components/auth/AuthLayout';
import AuthLink from '@/components/auth/AuthLink';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { forgotPasswordApi } from '../api/passwordReset.api';
import { extractApiError } from '../utils/apiError';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setLoading(true);
    try {
      await forgotPasswordApi(email);
      setSent(true);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Request failed. Please try again.'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout
      cardClassName="max-w-md"
      title="Reset Password"
      description="Enter your email address and we&apos;ll send you a reset link if an account exists."
    >
      {sent ? (
        <>
          <Alert variant="success">
            <AlertDescription className="text-foreground">
              If an account exists with that email, a password reset link has been sent.
              Check your inbox and spam folder.
            </AlertDescription>
          </Alert>
          <div className="text-center text-sm">
            <AuthLink to="/login" className="inline-flex items-center justify-center gap-2">
              <ArrowLeft className="size-4" />
              Back to Sign In
            </AuthLink>
          </div>
        </>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {error ? (
            <Alert variant="destructive">
              <AlertDescription className="text-foreground">{error}</AlertDescription>
            </Alert>
          ) : null}

          <div className="space-y-2">
            <Label htmlFor="forgot-email">Email</Label>
            <Input
              id="forgot-email"
              autoFocus
              placeholder="name@example.com"
              required
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? 'Sending...' : 'Send Reset Link'}
          </Button>

          <div className="text-center text-sm">
            <AuthLink to="/login" className="inline-flex items-center justify-center gap-2">
              <ArrowLeft className="size-4" />
              Back to Sign In
            </AuthLink>
          </div>
        </form>
      )}
    </AuthLayout>
  );
}
