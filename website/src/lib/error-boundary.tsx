"use client";

/**
 * Global Error Boundary (plan.md §10).
 * Never leaves UI in undefined state.
 */

import { Component, type ReactNode } from "react";
import { AlertCircle, RefreshCw } from "lucide-react";

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    // Log to monitoring service in production
    console.error("[ErrorBoundary]", error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="min-h-[60vh] flex flex-col items-center justify-center p-8 text-center">
          <div className="p-4 rounded-full bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)] mb-6">
            <AlertCircle size={40} />
          </div>
          <h2 className="text-2xl font-bold text-[var(--oj-text)] mb-2">
            Something went wrong
          </h2>
          <p className="text-[var(--oj-muted)] mb-6 max-w-md">
            {this.state.error?.message || "An unexpected error occurred."}
          </p>
          <button
            onClick={this.handleReset}
            className="cursor-pointer inline-flex items-center gap-2 px-6 py-2.5 rounded-lg bg-[var(--oj-accent)] text-white font-medium hover:bg-[var(--oj-accent-dk)] transition-colors active:scale-[0.98]"
          >
            <RefreshCw size={16} />
            Try Again
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
