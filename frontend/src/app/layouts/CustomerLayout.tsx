import { Outlet, Link } from "react-router";
import { Users } from "lucide-react";

export function CustomerLayout() {
  return (
    <div className="min-h-screen bg-gray-50 font-sans text-gray-900 flex flex-col">
      <header className="bg-white border-b border-gray-200 py-4 px-4 sm:px-6 lg:px-8">
        <div className="max-w-5xl mx-auto flex justify-between items-center">
          <Link to="/" className="flex items-center gap-2">
            <div className="bg-blue-600 p-2 rounded-lg text-white">
              <Users className="w-5 h-5" />
            </div>
            <span className="font-bold text-xl text-gray-900 tracking-tight">FairQueue</span>
          </Link>
          <div className="flex items-center gap-4 text-sm font-medium">
            <Link to="/organizer" className="text-gray-500 hover:text-blue-600 transition-colors">
              Organizer Login
            </Link>
          </div>
        </div>
      </header>
      <main className="flex-grow w-full max-w-5xl mx-auto p-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
      <footer className="bg-white border-t border-gray-200 py-6 text-center text-sm text-gray-500">
        <p>Powered by FairQueue &copy; 2026. Fair allocation guaranteed.</p>
      </footer>
    </div>
  );
}
