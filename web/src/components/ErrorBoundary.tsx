import React from 'react';
import styles from './ErrorBoundary.module.css';

type State = { hasError: boolean; error: Error | null };

class ErrorBoundary extends React.Component<React.PropsWithChildren<{}>, State> {
  constructor(props: React.PropsWithChildren<{}>) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    console.error(error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): React.ReactNode {
    if (this.state.hasError) {
      return (
        <div className={styles.overlay}>
          <div className={styles.card}>
            <div className={styles.icon}>⚠</div>
            <h2 className={styles.heading}>Something went wrong</h2>
            <p className={styles.message}>{this.state.error?.message ?? 'An unexpected error occurred'}</p>
            <button
              className={styles.retryBtn}
              onClick={this.handleRetry}
              type="button"
            >
              Retry
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
