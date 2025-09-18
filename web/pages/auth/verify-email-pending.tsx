/**
 * Email Verification Pending Page
 * Shown after user registration to inform them to check their email
 */

import React, { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { apiClient } from '../../lib/api-client';
import { getErrorMessage } from '../../lib/api-utils';

export default function VerifyEmailPendingPage() {
  const router = useRouter();
  const email = router.query.email as string;

  const [isResending, setIsResending] = useState(false);
  const [resendMessage, setResendMessage] = useState<string | null>(null);
  const [resendError, setResendError] = useState<string | null>(null);

  const handleResendEmail = async () => {
    if (!email) {
      setResendError('Email address not found. Please try registering again.');
      return;
    }

    setIsResending(true);
    setResendError(null);
    setResendMessage(null);

    try {
      // TODO: Implement resendVerificationEmail endpoint in backend
      setResendError('Resend functionality is not yet implemented. Please try registering again or contact support.');
    } catch (err) {
      setResendError(getErrorMessage(err));
    } finally {
      setIsResending(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">🃏</h1>
          <h2 className="text-3xl font-extrabold text-gray-900">
            Check Your Email
          </h2>
        </div>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
          <div className="text-center">
            <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-blue-100 mb-4">
              <svg className="h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>

            <h3 className="text-lg font-medium text-gray-900 mb-2">
              Verify your email address
            </h3>

            <p className="text-sm text-gray-600 mb-6">
              We've sent a verification email to{' '}
              {email ? (
                <strong>{email}</strong>
              ) : (
                'your email address'
              )}
              . Click the link in the email to verify your account and start playing.
            </p>

            <div className="space-y-4">
              {/* Resend Status Messages */}
              {resendMessage && (
                <div className="bg-green-50 border border-green-200 rounded-md p-3">
                  <p className="text-green-600 text-sm">{resendMessage}</p>
                </div>
              )}

              {resendError && (
                <div className="bg-red-50 border border-red-200 rounded-md p-3">
                  <p className="text-red-600 text-sm">{resendError}</p>
                </div>
              )}

              {/* Resend Button */}
              <button
                onClick={handleResendEmail}
                disabled={isResending}
                className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isResending ? (
                  <>
                    <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white inline" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Sending...
                  </>
                ) : (
                  'Resend verification email'
                )}
              </button>

              <div className="text-xs text-gray-500">
                <p className="mb-2">
                  <strong>Didn't receive the email?</strong>
                </p>
                <ul className="text-left space-y-1">
                  <li>• Check your spam or junk folder</li>
                  <li>• Make sure you entered the correct email address</li>
                  <li>• Wait a few minutes and try again</li>
                </ul>
              </div>
            </div>

            {/* Links */}
            <div className="mt-8 pt-6 border-t border-gray-200">
              <div className="space-y-4">
                <Link
                  href="/auth/login"
                  className="block font-medium text-blue-600 hover:text-blue-500"
                >
                  Already verified? Sign in
                </Link>

                <Link
                  href="/auth/register"
                  className="block font-medium text-blue-600 hover:text-blue-500"
                >
                  Use a different email address
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}