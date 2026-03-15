import { Link } from "react-router";
import { ImageWithFallback } from "../components/figma/ImageWithFallback";
import { Calendar, MapPin, Users, ShieldCheck, CheckCircle2, ChevronRight, Info } from "lucide-react";
import { useState } from "react";
import { motion } from "motion/react";
import { clsx } from "clsx";

export function Home() {
  const [showHowItWorks, setShowHowItWorks] = useState(false);

  const eventData = {
    title: "Burna Boy: Live in Lagos",
    date: "December 28, 2025",
    time: "6:00 PM",
    venue: "Tafawa Balewa Square (TBS), Lagos",
    priceRange: "₦25,000 - ₦150,000",
    image: "https://images.unsplash.com/photo-1766019463317-1cc801c15e61?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&ixid=M3w3Nzg4Nzd8MHwxfHNlYXJjaHwxfHxidXJuYSUyMGJveSUyMGNvbmNlcnQlMjBjcm93ZCUyMHN0YWdlJTIwbGlnaHRpbmd8ZW58MXx8fHwxNzczNTM5NzQxfDA&ixlib=rb-4.1.0&q=80&w=1080&utm_source=figma&utm_medium=referral",
    activeUsers: "2,347",
    status: "On Sale Now",
  };

  return (
    <div className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8 md:py-12">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-start">
        {/* Event Content */}
        <div className="space-y-8 animate-in fade-in slide-in-from-left duration-700">
          <div className="space-y-4">
            <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-green-100 text-green-700 uppercase tracking-wider">
              {eventData.status}
            </span>
            <h1 className="text-4xl md:text-5xl font-bold tracking-tight text-[#111827]">
              {eventData.title}
            </h1>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
            <div className="flex items-center gap-3 text-[#6B7280]">
              <div className="p-2 bg-white rounded-lg border border-[#E5E7EB]">
                <Calendar className="w-5 h-5 text-[#2563EB]" />
              </div>
              <div className="flex flex-col">
                <span className="text-xs font-medium uppercase tracking-wide text-gray-400">Date & Time</span>
                <span className="font-semibold text-[#111827]">{eventData.date} • {eventData.time}</span>
              </div>
            </div>
            <div className="flex items-center gap-3 text-[#6B7280]">
              <div className="p-2 bg-white rounded-lg border border-[#E5E7EB]">
                <MapPin className="w-5 h-5 text-[#2563EB]" />
              </div>
              <div className="flex flex-col">
                <span className="text-xs font-medium uppercase tracking-wide text-gray-400">Venue</span>
                <span className="font-semibold text-[#111827] truncate max-w-[200px]">{eventData.venue}</span>
              </div>
            </div>
          </div>

          <div className="p-6 bg-[#2563EB]/5 border border-[#2563EB]/20 rounded-2xl flex items-center justify-between">
            <div className="flex items-center gap-4 text-[#2563EB]">
              <Users className="w-6 h-6" />
              <div>
                <span className="block font-bold text-xl">{eventData.activeUsers}</span>
                <span className="text-sm font-medium opacity-80">People in queue right now</span>
              </div>
            </div>
            <div className="hidden sm:flex items-center gap-2 text-green-600 bg-white px-3 py-1.5 rounded-full border border-green-100 shadow-sm">
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-xs font-bold uppercase tracking-wider">Live Updates</span>
            </div>
          </div>

          <div className="space-y-4">
            <Link
              to="/queue"
              className="w-full flex items-center justify-center gap-2 bg-[#2563EB] text-white py-4 px-8 rounded-xl font-bold text-lg hover:bg-[#1d4ed8] hover:scale-[1.02] active:scale-[0.98] transition-all shadow-lg shadow-blue-500/25 group"
            >
              Join the Queue
              <ChevronRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </Link>
            <p className="text-center text-[#6B7280] text-sm flex items-center justify-center gap-2">
              <ShieldCheck className="w-4 h-4 text-[#10B981]" />
              Fair allocation guaranteed by FairQueue
            </p>
          </div>

          {/* How It Works Section */}
          <div className="border border-[#E5E7EB] rounded-2xl overflow-hidden bg-white shadow-sm">
            <button
              onClick={() => setShowHowItWorks(!showHowItWorks)}
              className="w-full p-4 flex items-center justify-between text-left hover:bg-gray-50 transition-colors"
            >
              <div className="flex items-center gap-3">
                <Info className="w-5 h-5 text-[#2563EB]" />
                <span className="font-semibold text-[#111827]">How the queue works</span>
              </div>
              <ChevronRight className={clsx("w-5 h-5 transition-transform duration-300", showHowItWorks && "rotate-90")} />
            </button>
            <motion.div
              initial={false}
              animate={{ height: showHowItWorks ? "auto" : 0, opacity: showHowItWorks ? 1 : 0 }}
              className="overflow-hidden"
            >
              <div className="p-6 pt-0 space-y-4 text-sm text-[#6B7280] border-t border-[#E5E7EB]/50">
                <p>To ensure a fair experience and prevent website crashes, we've implemented a virtual waiting room.</p>
                <ul className="space-y-3">
                  <li className="flex gap-3">
                    <CheckCircle2 className="w-5 h-5 text-[#10B981] flex-shrink-0" />
                    <span>Everyone has an equal chance. When the sale starts, users are assigned a random position.</span>
                  </li>
                  <li className="flex gap-3">
                    <CheckCircle2 className="w-5 h-5 text-[#10B981] flex-shrink-0" />
                    <span>Your position is securely linked to your session. Don't refresh or leave the page.</span>
                  </li>
                  <li className="flex gap-3">
                    <CheckCircle2 className="w-5 h-5 text-[#10B981] flex-shrink-0" />
                    <span>Once it's your turn, you'll have a set amount of time to complete your purchase.</span>
                  </li>
                </ul>
              </div>
            </motion.div>
          </div>
        </div>

        {/* Hero Image / Sidebar */}
        <div className="space-y-8 animate-in fade-in slide-in-from-right duration-700">
          <div className="relative aspect-[4/3] rounded-3xl overflow-hidden shadow-2xl group">
            <ImageWithFallback
              src={eventData.image}
              alt={eventData.title}
              className="object-cover w-full h-full group-hover:scale-105 transition-transform duration-700"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent" />
            <div className="absolute bottom-8 left-8 right-8 text-white">
              <span className="text-[#F59E0B] font-bold text-sm tracking-widest uppercase mb-2 block">Featured Event</span>
              <h3 className="text-3xl font-bold mb-2">{eventData.title}</h3>
              <p className="text-white/80 line-clamp-2">The African Giant returns to Lagos for an unforgettable night of music and culture at TBS.</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="p-4 bg-white border border-[#E5E7EB] rounded-2xl text-center">
              <span className="text-[#6B7280] text-xs font-semibold uppercase block mb-1">Starting from</span>
              <span className="text-[#111827] text-xl font-bold">₦25,000</span>
            </div>
            <div className="p-4 bg-white border border-[#E5E7EB] rounded-2xl text-center">
              <span className="text-[#6B7280] text-xs font-semibold uppercase block mb-1">Availability</span>
              <span className="text-green-600 text-xl font-bold">Good</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
