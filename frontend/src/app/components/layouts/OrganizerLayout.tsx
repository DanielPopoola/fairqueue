import React from 'react';
import { Outlet, Link, NavLink } from 'react-router';
import { Ticket, LayoutDashboard, CalendarPlus, BarChart3, Settings, Menu, X } from 'lucide-react';

export function OrganizerLayout() {
  const [menuOpen, setMenuOpen] = React.useState(false);
  
  const navItems = [
    { to: "/organizer", icon: LayoutDashboard, label: "Dashboard" },
    { to: "/organizer/create", icon: CalendarPlus, label: "Create Event" },
  ];

  return (
    <div className="min-h-screen flex flex-col md:flex-row bg-[#F9FAFB]">
      {/* Mobile Header */}
      <div className="md:hidden bg-white border-b border-[var(--color-border)] p-4 flex items-center justify-between">
        <Link to="/organizer" className="flex items-center gap-2 text-[var(--color-primary)] font-bold text-xl">
          <Ticket className="w-6 h-6" />
          FairQueue Pro
        </Link>
        <button onClick={() => setMenuOpen(!menuOpen)} className="p-2 text-gray-500">
          {menuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
        </button>
      </div>

      {/* Sidebar Navigation */}
      <nav className={`
        fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-[var(--color-border)] transform transition-transform duration-200 ease-in-out
        md:relative md:translate-x-0 flex flex-col
        ${menuOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        <div className="p-6 hidden md:block">
          <Link to="/organizer" className="flex items-center gap-2 text-[var(--color-primary)] font-bold text-2xl">
            <Ticket className="w-8 h-8" />
            FairQueue Pro
          </Link>
        </div>
        
        <div className="flex-1 px-4 py-6 space-y-2 overflow-y-auto">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/organizer"}
              onClick={() => setMenuOpen(false)}
              className={({ isActive }) => `
                flex items-center gap-3 px-4 py-3 rounded-lg font-medium transition-colors
                ${isActive 
                  ? 'bg-blue-50 text-[var(--color-primary)]' 
                  : 'text-[var(--color-text-secondary)] hover:bg-gray-50 hover:text-[var(--color-text-primary)]'}
              `}
            >
              <item.icon className="w-5 h-5" />
              {item.label}
            </NavLink>
          ))}
        </div>

        <div className="p-4 border-t border-[var(--color-border)]">
          <div className="flex items-center gap-3 px-4 py-3">
            <div className="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-[var(--color-primary)] font-bold">
              OG
            </div>
            <div>
              <div className="text-sm font-semibold text-[var(--color-text-primary)]">Organizer</div>
              <div className="text-xs text-[var(--color-text-secondary)]">Pro Account</div>
            </div>
          </div>
        </div>
      </nav>

      {/* Overlay for mobile */}
      {menuOpen && (
        <div 
          className="fixed inset-0 bg-black/50 z-40 md:hidden"
          onClick={() => setMenuOpen(false)}
        />
      )}

      {/* Main Content */}
      <main className="flex-1 flex flex-col min-w-0 h-screen overflow-hidden">
        <div className="flex-1 overflow-y-auto p-4 sm:p-8">
          <div className="max-w-6xl mx-auto">
            <Outlet />
          </div>
        </div>
      </main>
    </div>
  );
}
