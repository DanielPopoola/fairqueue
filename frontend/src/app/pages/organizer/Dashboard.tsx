import { useNavigate, Link } from "react-router";
import { 
  Users, 
  TrendingUp, 
  PlusCircle, 
  Calendar, 
  ChevronRight, 
  Activity, 
  DollarSign, 
  ArrowUpRight, 
  AlertCircle, 
  MoreVertical, 
  ExternalLink 
} from "lucide-react";
import { useState } from "react";
import { clsx } from "clsx";

export function Dashboard() {
  const navigate = useNavigate();
  const [activeEvents] = useState([
    { 
      id: "burna-2025", 
      name: "Burna Boy: Live in Lagos", 
      status: "Active Sale", 
      ticketsSold: 1234, 
      totalTickets: 5000, 
      revenue: 30850000, 
      queueSize: 2347, 
      health: "Healthy" 
    },
    { 
      id: "tech-summit-25", 
      name: "Nigeria Tech Summit 2025", 
      status: "Upcoming", 
      ticketsSold: 0, 
      totalTickets: 1200, 
      revenue: 0, 
      queueSize: 0, 
      health: "Standby" 
    }
  ]);

  const stats = [
    { name: "Active Events", value: "3", icon: Activity, trend: "+1 this month", color: "blue" },
    { name: "Total Sales", value: "₦2.45M", icon: DollarSign, trend: "+12.5% vs last month", color: "green" },
    { name: "Tickets Sold", value: "1,234", icon: Users, trend: "85% sell-through rate", color: "purple" },
    { name: "Upcoming Sales", value: "2", icon: Calendar, trend: "Starts in 3 days", color: "amber" },
  ];

  return (
    <div className="flex-1 bg-[#F9FAFB] max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8 md:py-12 space-y-12">
      {/* Welcome Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-6">
        <div>
          <h1 className="text-3xl font-black text-[#111827]">Organizer Dashboard</h1>
          <p className="text-[#6B7280] font-medium mt-1">Manage your events and monitor sales in real-time.</p>
        </div>
        <button
          onClick={() => navigate("/organizer/create")}
          className="bg-[#2563EB] text-white px-6 py-3 rounded-xl font-bold text-sm hover:bg-[#1d4ed8] transition-all shadow-lg shadow-blue-500/20 flex items-center gap-2 group hover:scale-[1.02] active:scale-[0.98]"
        >
          <PlusCircle className="w-5 h-5 group-hover:rotate-90 transition-transform duration-300" />
          Create New Event
        </button>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map((stat) => (
          <div key={stat.name} className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm hover:shadow-md transition-shadow group">
            <div className="flex justify-between items-start mb-4">
              <div className={clsx(
                "p-3 rounded-2xl",
                stat.color === "blue" && "bg-blue-50 text-blue-600",
                stat.color === "green" && "bg-green-50 text-green-600",
                stat.color === "purple" && "bg-purple-50 text-purple-600",
                stat.color === "amber" && "bg-amber-50 text-amber-600",
              )}>
                <stat.icon className="w-6 h-6" />
              </div>
              <span className="text-[10px] font-black uppercase tracking-widest text-[#6B7280] opacity-50 group-hover:opacity-100 transition-opacity">Realtime</span>
            </div>
            <div>
              <span className="text-3xl font-black text-[#111827] tracking-tight">{stat.value}</span>
              <p className="text-sm font-bold text-[#6B7280] mt-1">{stat.name}</p>
            </div>
            <div className="mt-4 flex items-center gap-1 text-xs font-bold text-green-600">
              <TrendingUp className="w-3 h-3" />
              {stat.trend}
            </div>
          </div>
        ))}
      </div>

      {/* Active Events Section */}
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h2 className="text-xl font-bold text-[#111827] flex items-center gap-2">
            <Activity className="w-5 h-5 text-[#2563EB]" />
            Active Sales
          </h2>
          <Link to="/organizer/events" className="text-sm font-bold text-[#2563EB] hover:underline">View All Events</Link>
        </div>

        <div className="grid grid-cols-1 gap-4">
          {activeEvents.map((event) => (
            <div key={event.id} className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm hover:border-[#2563EB] transition-colors group">
              <div className="flex flex-col lg:flex-row justify-between items-start lg:items-center gap-6">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3 mb-2">
                    <span className={clsx(
                      "px-3 py-1 rounded-full text-[10px] font-bold uppercase tracking-widest",
                      event.status === "Active Sale" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"
                    )}>
                      {event.status}
                    </span>
                    <span className="flex items-center gap-1.5 px-3 py-1 bg-blue-50 text-blue-700 rounded-full text-[10px] font-bold uppercase border border-blue-100 shadow-sm">
                      <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-pulse" />
                      {event.health}
                    </span>
                  </div>
                  <h3 className="text-xl font-black text-[#111827] truncate group-hover:text-[#2563EB] transition-colors">
                    {event.name}
                  </h3>
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-8 flex-shrink-0">
                  <div className="space-y-1">
                    <span className="text-[10px] font-bold text-[#6B7280] uppercase tracking-widest">Revenue</span>
                    <p className="font-black text-[#111827]">₦{event.revenue.toLocaleString()}</p>
                  </div>
                  <div className="space-y-1">
                    <span className="text-[10px] font-bold text-[#6B7280] uppercase tracking-widest">Sold</span>
                    <div className="flex items-center gap-2">
                      <p className="font-black text-[#111827]">{event.ticketsSold}</p>
                      <span className="text-[10px] font-medium text-[#6B7280]">/ {event.totalTickets}</span>
                    </div>
                  </div>
                  <div className="space-y-1">
                    <span className="text-[10px] font-bold text-[#6B7280] uppercase tracking-widest">In Queue</span>
                    <p className="font-black text-[#2563EB]">{event.queueSize}</p>
                  </div>
                  <div className="flex items-center">
                    <button
                      onClick={() => navigate(`/organizer/live/${event.id}`)}
                      className="bg-gray-50 text-[#111827] px-4 py-2 rounded-xl font-bold text-xs hover:bg-[#2563EB] hover:text-white transition-all flex items-center gap-2 shadow-sm border border-[#E5E7EB] group-hover:border-transparent"
                    >
                      Monitor Live
                      <ArrowUpRight className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Recent Activity & Feed */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-[#111827] flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-[#2563EB]" />
              Real-time Feed
            </h2>
          </div>
          <div className="bg-white border border-[#E5E7EB] rounded-3xl overflow-hidden shadow-sm">
            <div className="divide-y divide-[#E5E7EB]">
              {[
                { time: "10:15:32", msg: "User purchased 2x VIP tickets", amount: "₦100,000", status: "success" },
                { time: "10:15:28", msg: "50 users admitted from queue", amount: "", status: "info" },
                { time: "10:15:15", msg: "Payment failed for Order #12345 (card declined)", amount: "", status: "error" },
                { time: "10:14:55", msg: "Queue rate increased to 200 users/min", amount: "", status: "info" },
                { time: "10:14:10", msg: "User purchased 1x Regular ticket", amount: "₦25,000", status: "success" },
              ].map((item, i) => (
                <div key={i} className="p-4 hover:bg-gray-50 transition-colors flex items-center gap-4">
                  <span className="text-xs font-bold text-[#6B7280] tabular-nums whitespace-nowrap">{item.time}</span>
                  <div className={clsx(
                    "w-2 h-2 rounded-full flex-shrink-0",
                    item.status === "success" && "bg-green-500",
                    item.status === "info" && "bg-blue-500",
                    item.status === "error" && "bg-red-500",
                  )} />
                  <p className="text-sm font-medium text-[#111827] flex-1 truncate">{item.msg}</p>
                  {item.amount && <span className="text-sm font-black text-[#111827]">{item.amount}</span>}
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="space-y-6">
          <h2 className="text-xl font-bold text-[#111827]">Quick Actions</h2>
          <div className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm space-y-3">
            {[
              { label: "Download Sales Report", icon: ChevronRight, desc: "Excel, CSV or PDF" },
              { label: "Update Queue Settings", icon: ChevronRight, desc: "Admission rates & strategy" },
              { label: "Connect Paystack", icon: ExternalLink, desc: "Manage payments" },
              { label: "Team Permissions", icon: ChevronRight, desc: "Invite collaborators" },
            ].map((action, i) => (
              <button key={i} className="w-full flex items-center justify-between p-4 rounded-2xl hover:bg-gray-50 transition-colors text-left group">
                <div>
                  <span className="block font-bold text-sm text-[#111827]">{action.label}</span>
                  <span className="text-xs text-[#6B7280]">{action.desc}</span>
                </div>
                <action.icon className="w-4 h-4 text-[#6B7280] group-hover:text-[#2563EB] group-hover:translate-x-1 transition-all" />
              </button>
            ))}
          </div>

          <div className="p-6 bg-[#2563EB] rounded-3xl text-white space-y-4 shadow-xl shadow-blue-500/20 relative overflow-hidden">
            <div className="absolute top-0 right-0 p-4 opacity-20 rotate-12">
              <TrendingUp className="w-24 h-24" />
            </div>
            <h3 className="font-black text-xl leading-tight">Scale Your Events Safely.</h3>
            <p className="text-white/80 text-sm leading-relaxed">Need custom pricing or higher admission limits? Talk to our sales team about Enterprise features.</p>
            <button className="w-full py-3 bg-white text-[#2563EB] rounded-xl font-bold text-sm hover:bg-blue-50 transition-colors">
              Talk to Sales
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
