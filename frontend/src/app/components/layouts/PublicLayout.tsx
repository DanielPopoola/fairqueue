import React from 'react';
import { Outlet, Link } from 'react-router';
import { Ticket } from 'lucide-react';

export function PublicLayout() {
  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[var(--color-border)] bg-white sticky top-0 z-10">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2 text-[var(--color-primary)] font-bold text-xl">
            <Ticket className="w-6 h-6" />
            FairQueue
          </Link>
          <div className="text-sm text-[var(--color-text-secondary)] hidden sm:block">
            Fair allocation guaranteed
          </div>
        </div>
      </header>
      <main className="flex-1 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 w-full">
        <Outlet />
      </main>
      <footer className="bg-white border-t border-[var(--color-border)] py-6 mt-auto">
        <div className="max-w-4xl mx-auto px-4 text-center text-sm text-[var(--color-text-secondary)]">
          <p>© {new Date().getFullYear()} FairQueue. All rights reserved.</p>
        </div>
      </footer>
    </div>
  );
}
