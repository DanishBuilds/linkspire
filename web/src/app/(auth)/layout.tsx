import { ThemeToggle } from '@/components/ThemeToggle';

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <a href="#main" className="skip-link">Skip to content</a>
      <main id="main" className="auth-wrap">
        <div className="auth-theme">
          <ThemeToggle />
        </div>
        {children}
      </main>
    </>
  );
}
