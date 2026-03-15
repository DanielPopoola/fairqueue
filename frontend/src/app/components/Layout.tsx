import { Outlet, Link, useLocation } from "react-router";
import { Users, LayoutDashboard, PlusCircle, LogOut, Menu, X, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { clsx } from "clsx";

export function Layout() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const location = useLocation();

  const isOrganizer = location.pathname.startsWith("/organizer");

  const navigation = isOrganizer
    ? [
        { name: "Dashboard", href: "/organizer", icon: LayoutDashboard },
        { name: "Create Event", href: "/organizer/create", icon: PlusCircle },
      ]
    : [
        { name: "Featured Events", href: "/", icon: Users },
      ];

  return (
    <div className="min-h-screen bg-[#F9FAFB] flex flex-col font-['Inter',_system-ui,_sans-serif] text-[#111827]">
      {/* Navbar */}
      <nav className="sticky top-0 z-50 bg-white border-b border-[#E5E7EB]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <Link to="/" className="flex items-center gap-2">
                <div className="bg-[#2563EB] p-1.5 rounded-lg">
                  <ShieldCheck className="w-6 h-6 text-white" />
                </div>
                <span className="text-xl font-bold tracking-tight text-[#111827]">
                  Fair<span className="text-[#2563EB]">Queue</span>
                </span>
              </Link>
              
              <div className="hidden sm:ml-8 sm:flex sm:space-x-8">
                {navigation.map((item) => (
                  <Link
                    key={item.name}
                    to={item.href}
                    className={clsx(
                      "inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition-colors",
                      location.pathname === item.href
                        ? "border-[#2563EB] text-[#2563EB]"
                        : "border-transparent text-[#6B7280] hover:text-[#111827] hover:border-[#E5E7EB]"
                    )}
                  >
                    {item.name}
                  </Link>
                ))}
              </div>
            </div>

            <div className="hidden sm:flex sm:items-center sm:gap-4">
              {isOrganizer ? (
                <button className="flex items-center gap-2 text-[#6B7280] hover:text-[#EF4444] transition-colors text-sm font-medium">
                  <LogOut className="w-4 h-4" />
                  Sign Out
                </button>
              ) : (
                <Link
                  to="/organizer"
                  className="bg-[#2563EB] text-white px-4 py-2 rounded-lg text-sm font-semibold hover:bg-[#1d4ed8] transition-colors"
                >
                  Organizer Login
                </Link>
              )}
            </div>

            {/* Mobile menu button */}
            <div className="flex items-center sm:hidden">
              <button
                onClick={() => setIsMenuOpen(!isMenuOpen)}
                className="inline-flex items-center justify-center p-2 rounded-md text-[#6B7280] hover:text-[#111827] hover:bg-gray-100"
              >
                {isMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
              </button>
            </div>
          </div>
        </div>

        {/* Mobile menu */}
        {isMenuOpen && (
          <div className="sm:hidden bg-white border-b border-[#E5E7EB]">
            <div className="pt-2 pb-3 space-y-1">
              {navigation.map((item) => (
                <Link
                  key={item.name}
                  to={item.href}
                  onClick={() => setIsMenuOpen(false)}
                  className={clsx(
                    "block pl-3 pr-4 py-2 border-l-4 text-base font-medium transition-colors",
                    location.pathname === item.href
                      ? "bg-blue-50 border-[#2563EB] text-[#2563EB]"
                      : "border-transparent text-[#6B7280] hover:bg-gray-50 hover:border-[#E5E7EB]"
                  )}
                >
                  <div className="flex items-center gap-3">
                    <item.icon className="w-5 h-5" />
                    {item.name}
                  </div>
                </Link>
              ))}
              {!isOrganizer && (
                <Link
                  to="/organizer"
                  onClick={() => setIsMenuOpen(false)}
                  className="block pl-3 pr-4 py-2 border-l-4 border-transparent text-base font-medium text-[#2563EB] hover:bg-gray-50"
                >
                  Organizer Dashboard
                </Link>
              )}
            </div>
          </div>
        )}
      </nav>

      {/* Main Content */}
      <main className="flex-grow flex flex-col">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="bg-white border-t border-[#E5E7EB] py-8">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <div className="flex items-center gap-2 grayscale opacity-50">
              <ShieldCheck className="w-5 h-5 text-[#111827]" />
              <span className="text-sm font-semibold tracking-tight text-[#111827]">
                FairQueue
              </span>
            </div>
            <p className="text-[#6B7280] text-sm">
              &copy; {new Date().getFullYear()} FairQueue. All rights reserved. Made for Nigeria's high-demand sales.
            </p>
            <div className="flex gap-6">
              <a href="#" className="text-sm text-[#6B7280] hover:text-[#111827]">Help Center</a>
              <a href="#" className="text-sm text-[#6B7280] hover:text-[#111827]">Privacy Policy</a>
              <a href="#" className="text-sm text-[#6B7280] hover:text-[#111827]">Terms of Service</a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
