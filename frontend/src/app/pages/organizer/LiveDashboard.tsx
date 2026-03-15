import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router";
import { 
  Activity, 
  Users, 
  DollarSign, 
  TrendingUp, 
  AlertCircle, 
  ChevronLeft, 
  Pause, 
  Play, 
  Download, 
  CheckCircle2, 
  Clock, 
  ArrowUpRight, 
  ArrowDownRight, 
  MessageSquare 
} from "lucide-react";
import { 
  LineChart, 
  Line, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer, 
  AreaChart, 
  Area 
} from "recharts";
import { clsx } from "clsx";
import { toast } from "sonner";

const mockChartData = Array.from({ length: 20 }, (_, i) => ({
  time: `${10 + Math.floor(i / 6)}:${(i * 10) % 60}`.padStart(5, "0"),
  sales: Math.floor(Math.random() * 50) + i * 5,
  queue: Math.floor(Math.random() * 200) + 1000 - i * 30,
}));

export function LiveDashboard() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [isPaused, setIsPaused] = useState(false);
  const [admissionRate, setAdmissionRate] = useState(100);
  const [realtimeData, setRealtimeData] = useState({
    ticketsSold: 1234,
    totalTickets: 5000,
    revenue: 30850000,
    queueSize: 2347,
    avgWait: 18,
  });

  // Simulate live updates
  useEffect(() => {
    if (isPaused) return;
    const interval = setInterval(() => {
      setRealtimeData(prev => ({
        ...prev,
        ticketsSold: prev.ticketsSold + Math.floor(Math.random() * 5),
        revenue: prev.revenue + (Math.random() > 0.5 ? 50000 : 25000),
        queueSize: Math.max(0, prev.queueSize - Math.floor(Math.random() * 10) + Math.floor(Math.random() * 8)),
      }));
    }, 3000);
    return () => clearInterval(interval);
  }, [isPaused]);

  const handlePause = () => {
    setIsPaused(!isPaused);
    toast.info(isPaused ? "Admissions Resumed" : "Admissions Paused", {
      description: isPaused ? "Queue is moving again." : "No new users will be admitted.",
    });
  };

  return (
    <div className="flex-1 bg-[#F9FAFB] max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8 md:py-12 space-y-8">
      {/* Top Bar */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-6">
        <div className="space-y-2">
          <button
            onClick={() => navigate("/organizer")}
            className="inline-flex items-center gap-2 text-[#6B7280] font-bold text-sm hover:text-[#111827] group"
          >
            <ChevronLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
            Back to Dashboard
          </button>
          <div className="flex items-center gap-3">
            <h1 className="text-3xl font-black text-[#111827]">Live: Burna Boy Lagos</h1>
            <span className="flex items-center gap-1.5 px-3 py-1 bg-green-50 text-green-700 rounded-full text-[10px] font-bold uppercase border border-green-100 shadow-sm">
              <div className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
              Monitoring
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => toast.success("Broadcasting message...", { description: "Updating all waiting users." })}
            className="p-3 bg-white border border-[#E5E7EB] rounded-xl text-[#6B7280] hover:text-[#2563EB] hover:border-[#2563EB] transition-all shadow-sm group"
          >
            <MessageSquare className="w-5 h-5" />
          </button>
          <button
            onClick={handlePause}
            className={clsx(
              "flex items-center gap-2 px-6 py-3 rounded-xl font-bold text-sm transition-all shadow-lg",
              isPaused 
                ? "bg-green-500 text-white shadow-green-500/20 hover:bg-green-600" 
                : "bg-amber-500 text-white shadow-amber-500/20 hover:bg-amber-600"
            )}
          >
            {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
            {isPaused ? "Resume Admissions" : "Pause Admissions"}
          </button>
        </div>
      </div>

      {/* Main Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
        {[
          { label: "Tickets Sold", value: `${realtimeData.ticketsSold} / ${realtimeData.totalTickets}`, icon: Activity, color: "blue", sub: "24.7% Sold-through" },
          { label: "Total Revenue", value: `₦${realtimeData.revenue.toLocaleString()}`, icon: DollarSign, color: "green", sub: "+₦125k in last min" },
          { label: "People in Queue", value: realtimeData.queueSize.toLocaleString(), icon: Users, color: "purple", sub: "142 incoming / min" },
          { label: "Avg. Wait Time", value: `${realtimeData.avgWait} mins`, icon: Clock, color: "amber", sub: "Steady wait time" },
        ].map((stat, i) => (
          <div key={i} className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm hover:shadow-md transition-shadow">
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
              <span className="text-[10px] font-black uppercase tracking-widest text-[#6B7280] opacity-50">Active</span>
            </div>
            <span className="text-3xl font-black text-[#111827] tracking-tight tabular-nums">{stat.value}</span>
            <p className="text-sm font-bold text-[#6B7280] mt-1">{stat.label}</p>
            <p className="mt-3 text-[10px] font-bold text-[#6B7280] uppercase tracking-widest opacity-60">{stat.sub}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Charts & Graphs */}
        <div className="lg:col-span-2 space-y-6">
          <div className="bg-white border border-[#E5E7EB] rounded-3xl p-8 shadow-sm">
            <div className="flex justify-between items-center mb-8">
              <h3 className="font-bold text-lg text-[#111827]">Sales Velocity</h3>
              <div className="flex items-center gap-4 text-[10px] font-bold uppercase tracking-widest">
                <div className="flex items-center gap-1.5">
                  <div className="w-2.5 h-2.5 rounded-full bg-[#2563EB]" />
                  <span className="text-[#6B7280]">Tickets Sold</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="w-2.5 h-2.5 rounded-full bg-[#E5E7EB]" />
                  <span className="text-[#6B7280]">Queue Size</span>
                </div>
              </div>
            </div>
            <div className="h-[350px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={mockChartData}>
                  <defs>
                    <linearGradient id="colorSales" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#2563EB" stopOpacity={0.1}/>
                      <stop offset="95%" stopColor="#2563EB" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#F1F5F9" />
                  <XAxis 
                    dataKey="time" 
                    axisLine={false} 
                    tickLine={false} 
                    tick={{ fontSize: 10, fontWeight: 700, fill: "#6B7280" }} 
                    dy={10}
                  />
                  <YAxis 
                    axisLine={false} 
                    tickLine={false} 
                    tick={{ fontSize: 10, fontWeight: 700, fill: "#6B7280" }}
                  />
                  <Tooltip 
                    contentStyle={{ borderRadius: "16px", border: "none", boxShadow: "0 10px 15px -3px rgb(0 0 0 / 0.1)", padding: "12px" }}
                    labelStyle={{ fontWeight: 800, color: "#111827", marginBottom: "4px" }}
                  />
                  <Area type="monotone" dataKey="sales" stroke="#2563EB" strokeWidth={3} fillOpacity={1} fill="url(#colorSales)" />
                  <Area type="monotone" dataKey="queue" stroke="#E5E7EB" strokeWidth={2} fillOpacity={0} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm space-y-6">
              <h3 className="font-bold text-[#111827]">Queue Management</h3>
              <div className="space-y-4">
                <div className="flex justify-between items-center">
                  <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Admission Rate</label>
                  <span className="text-sm font-black text-[#2563EB]">{admissionRate} / min</span>
                </div>
                <input
                  type="range"
                  min="50"
                  max="500"
                  step="10"
                  className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-[#2563EB]"
                  value={admissionRate}
                  onChange={(e) => setAdmissionRate(parseInt(e.target.value))}
                />
                <div className="grid grid-cols-2 gap-3 pt-2">
                  <button className="py-2 bg-gray-50 border border-[#E5E7EB] rounded-xl text-xs font-bold text-[#111827] hover:bg-gray-100 transition-colors">
                    Reset Rate
                  </button>
                  <button className="py-2 bg-[#2563EB] text-white rounded-xl text-xs font-bold hover:bg-[#1d4ed8] transition-colors shadow-lg shadow-blue-500/20">
                    Apply Live
                  </button>
                </div>
              </div>
            </div>

            <div className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm space-y-4">
              <h3 className="font-bold text-[#111827]">System Health</h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between p-3 bg-green-50 border border-green-100 rounded-2xl">
                  <div className="flex items-center gap-3">
                    <CheckCircle2 className="w-4 h-4 text-green-600" />
                    <span className="text-xs font-bold text-green-700">Payment Processor</span>
                  </div>
                  <span className="text-[10px] font-black text-green-600 uppercase tracking-widest">Healthy</span>
                </div>
                <div className="flex items-center justify-between p-3 bg-green-50 border border-green-100 rounded-2xl">
                  <div className="flex items-center gap-3">
                    <CheckCircle2 className="w-4 h-4 text-green-600" />
                    <span className="text-xs font-bold text-green-700">Database Sync</span>
                  </div>
                  <span className="text-[10px] font-black text-green-600 uppercase tracking-widest">Healthy</span>
                </div>
                <div className="flex items-center justify-between p-3 bg-amber-50 border border-amber-100 rounded-2xl">
                  <div className="flex items-center gap-3">
                    <AlertCircle className="w-4 h-4 text-amber-600" />
                    <span className="text-xs font-bold text-amber-700">Admission Latency</span>
                  </div>
                  <span className="text-[10px] font-black text-amber-600 uppercase tracking-widest">240ms</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Live Feed */}
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-[#111827]">Real-time Feed</h3>
            <span className="flex items-center gap-1.5 px-2 py-0.5 bg-gray-100 text-[#6B7280] rounded-full text-[8px] font-bold uppercase tracking-widest">
              Live
            </span>
          </div>
          <div className="bg-white border border-[#E5E7EB] rounded-3xl overflow-hidden shadow-sm flex flex-col h-[650px]">
            <div className="p-4 border-b border-[#E5E7EB] bg-gray-50/50">
              <div className="flex gap-2">
                <button className="flex-1 py-1.5 bg-white border border-[#E5E7EB] rounded-lg text-[10px] font-black uppercase text-[#2563EB] shadow-sm">All</button>
                <button className="flex-1 py-1.5 bg-transparent rounded-lg text-[10px] font-black uppercase text-[#6B7280] hover:bg-white hover:text-[#111827] transition-all">Sales</button>
                <button className="flex-1 py-1.5 bg-transparent rounded-lg text-[10px] font-black uppercase text-[#6B7280] hover:bg-white hover:text-[#111827] transition-all">Errors</button>
              </div>
            </div>
            <div className="flex-1 overflow-y-auto divide-y divide-[#E5E7EB]">
              {[
                { type: "sale", msg: "VIP Ticket Purchased", time: "10:23:45", user: "Ade O.", amount: "₦100k" },
                { type: "admit", msg: "50 users admitted", time: "10:23:40", user: "System", amount: "" },
                { type: "sale", msg: "Regular Ticket Purchased", time: "10:23:35", user: "John D.", amount: "₦25k" },
                { type: "error", msg: "Payment Failed", time: "10:23:30", user: "Sara K.", amount: "Card Declined" },
                { type: "sale", msg: "VIP Ticket Purchased", time: "10:23:25", user: "Chioma A.", amount: "₦50k" },
                { type: "sale", msg: "2x Regular Purchased", time: "10:23:20", user: "Musa B.", amount: "₦50k" },
                { type: "admit", msg: "50 users admitted", time: "10:23:15", user: "System", amount: "" },
                { type: "sale", msg: "VIP Ticket Purchased", time: "10:23:10", user: "Ade O.", amount: "₦100k" },
                { type: "sale", msg: "Regular Ticket Purchased", time: "10:23:05", user: "John D.", amount: "₦25k" },
                { type: "error", msg: "Payment Failed", time: "10:23:00", user: "Sara K.", amount: "Insufficient Funds" },
              ].map((item, i) => (
                <div key={i} className="p-4 hover:bg-gray-50 transition-colors animate-in fade-in slide-in-from-top-2 duration-300">
                  <div className="flex justify-between items-start mb-1">
                    <span className="text-[10px] font-bold text-[#6B7280] tabular-nums uppercase tracking-widest">{item.time}</span>
                    <span className={clsx(
                      "text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded",
                      item.type === "sale" && "bg-green-100 text-green-700",
                      item.type === "admit" && "bg-blue-100 text-blue-700",
                      item.type === "error" && "bg-red-100 text-red-700",
                    )}>
                      {item.type}
                    </span>
                  </div>
                  <div className="flex justify-between items-end">
                    <div>
                      <p className="text-xs font-bold text-[#111827]">{item.msg}</p>
                      <p className="text-[10px] text-[#6B7280] font-medium">{item.user}</p>
                    </div>
                    {item.amount && <p className="text-xs font-black text-[#111827]">{item.amount}</p>}
                  </div>
                </div>
              ))}
            </div>
            <div className="p-4 border-t border-[#E5E7EB] bg-gray-50/50">
              <button className="w-full flex items-center justify-center gap-2 py-2 bg-white border border-[#E5E7EB] rounded-xl text-xs font-bold text-[#6B7280] hover:text-[#111827] transition-all">
                <Download className="w-4 h-4" />
                Export Live Feed
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
