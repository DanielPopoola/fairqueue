import { Outlet, Link, useLocation } from "react-router";
import { 
  Users, 
  LayoutDashboard, 
  CalendarPlus, 
  Activity, 
  BarChart3, 
  Settings, 
  LogOut,
  Menu,
  X
} from "lucide-react";
import { useState } from "react";
import { cn } from "../lib/utils";

const navigation = [
  { name: 'Dashboard', href: '/organizer', icon: LayoutDashboard },
  { name: 'Create Event', href: '/organizer/create', icon: CalendarPlus },
  { name: 'Live Event', href: '/organizer/live/1', icon: Activity },
  { name: 'Analytics', href: '/organizer/analytics/1', icon: BarChart3 },
];

export function OrganizerLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const location = useLocation();

  return (
    <div className="min-h-screen bg-gray-50 flex font-sans">
      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div 
          className="fixed inset-0 z-40 bg-gray-900/80 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div className={cn(
        "fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 flex flex-col transition-transform duration-300 ease-in-out lg:static lg:translate-x-0",
        sidebarOpen ? "translate-x-0" : "-translate-x-full"
      )}>
        <div className="flex h-16 items-center px-6 border-b border-gray-200">
          <Link to="/" className="flex items-center gap-2">
            <div className="bg-blue-600 p-1.5 rounded-lg text-white">
              <Users className="w-5 h-5" />
            </div>
            <span className="font-bold text-xl text-gray-900 tracking-tight">FairQueue Organizer</span>
          </Link>
          <button 
            className="ml-auto lg:hidden text-gray-500 hover:text-gray-900"
            onClick={() => setSidebarOpen(false)}
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
          {navigation.map((item) => {
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.name}
                to={item.href}
                className={cn(
                  "group flex items-center px-3 py-2 text-sm font-medium rounded-md",
                  isActive
                    ? "bg-blue-50 text-blue-700"
                    : "text-gray-700 hover:bg-gray-100 hover:text-gray-900"
                )}
                onClick={() => setSidebarOpen(false)}
              >
                <item.icon
                  className={cn(
                    "mr-3 flex-shrink-0 h-5 w-5",
                    isActive ? "text-blue-700" : "text-gray-400 group-hover:text-gray-500"
                  )}
                  aria-hidden="true"
                />
                {item.name}
              </Link>
            );
          })}
        </nav>
        
        <div className="p-4 border-t border-gray-200 space-y-2">
          <button className="flex items-center px-3 py-2 text-sm font-medium rounded-md text-gray-700 hover:bg-gray-100 hover:text-gray-900 w-full">
            <Settings className="mr-3 flex-shrink-0 h-5 w-5 text-gray-400" />
            Settings
          </button>
          <button className="flex items-center px-3 py-2 text-sm font-medium rounded-md text-red-700 hover:bg-red-50 hover:text-red-900 w-full">
            <LogOut className="mr-3 flex-shrink-0 h-5 w-5 text-red-500" />
            Log Out
          </button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header className="bg-white border-b border-gray-200 h-16 flex items-center px-4 lg:hidden">
          <button
            className="text-gray-500 hover:text-gray-900 focus:outline-none"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="w-6 h-6" />
          </button>
          <span className="ml-4 font-semibold text-gray-900">FairQueue Organizer</span>
        </header>

        <main className="flex-1 overflow-y-auto bg-gray-50 focus:outline-none">
          <div className="py-6 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
