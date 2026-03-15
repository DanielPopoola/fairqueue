import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { motion, AnimatePresence } from "motion/react";
import { Clock, Users, ShieldCheck, CheckCircle2, AlertCircle, Phone, ArrowRight } from "lucide-react";
import { toast } from "sonner";
import { clsx } from "clsx";

export function Queue() {
  const navigate = useNavigate();
  const [position, setPosition] = useState(1547);
  const [estimatedWait, setEstimatedWait] = useState(12);
  const [progress, setProgress] = useState(65);
  const [showSmsForm, setShowSmsForm] = useState(false);
  const [isAdmitted, setIsAdmitted] = useState(false);

  // Simulate queue movement
  useEffect(() => {
    const interval = setInterval(() => {
      setPosition((prev) => {
        if (prev <= 1) {
          setIsAdmitted(true);
          return 1;
        }
        const decrease = Math.floor(Math.random() * 45) + 10;
        return Math.max(1, prev - decrease);
      });
    }, 4000);

    return () => clearInterval(interval);
  }, []);

  // Admitted logic
  useEffect(() => {
    if (isAdmitted) {
      toast.success("It's your turn!", {
        description: "You have 5 minutes to complete your purchase.",
        duration: 10000,
      });
    }
  }, [isAdmitted]);

  return (
    <div className="flex-1 flex flex-col items-center justify-center max-w-2xl mx-auto w-full px-4 py-8 md:py-16">
      <AnimatePresence mode="wait">
        {!isAdmitted ? (
          <motion.div
            key="waiting"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95 }}
            className="w-full space-y-8 text-center"
          >
            <div className="space-y-4">
              <div className="inline-flex items-center gap-2 px-4 py-2 bg-blue-50 text-[#2563EB] rounded-full text-sm font-semibold border border-blue-100 mb-2">
                <div className="w-2 h-2 bg-[#2563EB] rounded-full animate-pulse" />
                Live: You are in the queue
              </div>
              <h1 className="text-3xl md:text-4xl font-bold tracking-tight text-[#111827]">
                Hold tight, you're in line.
              </h1>
              <p className="text-[#6B7280] text-lg max-w-lg mx-auto">
                We're admitting users at a controlled rate to prevent the system from crashing. Your spot is reserved.
              </p>
            </div>

            {/* Queue Position Card */}
            <div className="bg-white border border-[#E5E7EB] rounded-3xl p-8 md:p-12 shadow-sm relative overflow-hidden group">
              <div className="absolute top-0 left-0 right-0 h-1.5 bg-[#E5E7EB]">
                <motion.div
                  className="h-full bg-[#2563EB]"
                  initial={{ width: "0%" }}
                  animate={{ width: `${100 - (position / 2000) * 100}%` }}
                  transition={{ duration: 1.5, ease: "easeOut" }}
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-8 divide-y md:divide-y-0 md:divide-x divide-[#E5E7EB]">
                <div className="space-y-2 pb-6 md:pb-0">
                  <span className="text-xs font-bold uppercase tracking-widest text-[#6B7280]">Your Position</span>
                  <div className="flex flex-col items-center">
                    <motion.span
                      key={position}
                      initial={{ scale: 1.1, color: "#2563EB" }}
                      animate={{ scale: 1, color: "#111827" }}
                      className="text-6xl md:text-7xl font-black tabular-nums tracking-tighter"
                    >
                      #{position.toLocaleString()}
                    </motion.span>
                    <span className="text-sm font-medium text-[#10B981] flex items-center gap-1 mt-2">
                      <Users className="w-4 h-4" /> Moving fast
                    </span>
                  </div>
                </div>

                <div className="space-y-2 pt-6 md:pt-0 md:pl-8">
                  <span className="text-xs font-bold uppercase tracking-widest text-[#6B7280]">Est. Wait Time</span>
                  <div className="flex flex-col items-center">
                    <span className="text-6xl md:text-7xl font-black tabular-nums tracking-tighter text-[#111827]">
                      {Math.ceil(position / 120)}
                    </span>
                    <span className="text-sm font-medium text-[#6B7280] flex items-center gap-1 mt-2">
                      <Clock className="w-4 h-4" /> Minutes
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Warning Section */}
            <div className="bg-amber-50 border border-amber-100 rounded-2xl p-4 flex gap-4 items-start text-left">
              <AlertCircle className="w-5 h-5 text-[#F59E0B] flex-shrink-0 mt-0.5" />
              <div>
                <p className="text-[#92400E] font-semibold text-sm">Stay on this page</p>
                <p className="text-[#92400E]/80 text-sm">Do not refresh or leave this tab. If you lose your connection, you may lose your spot in line.</p>
              </div>
            </div>

            {/* SMS Notification Form */}
            {!showSmsForm ? (
              <button
                onClick={() => setShowSmsForm(true)}
                className="inline-flex items-center gap-2 text-[#2563EB] font-bold text-sm hover:underline"
              >
                <Phone className="w-4 h-4" />
                Get notified via SMS when it's your turn
              </button>
            ) : (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                className="bg-white border border-[#E5E7EB] rounded-2xl p-6 shadow-sm max-w-md mx-auto w-full"
              >
                <div className="flex justify-between items-center mb-4">
                  <h3 className="font-bold text-[#111827]">SMS Notification</h3>
                  <button onClick={() => setShowSmsForm(false)} className="text-xs font-bold text-[#6B7280] hover:text-[#111827]">CANCEL</button>
                </div>
                <div className="flex gap-2">
                  <input
                    type="tel"
                    placeholder="e.g. 08012345678"
                    className="flex-1 px-4 py-2 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none text-sm"
                  />
                  <button
                    onClick={() => {
                      toast.success("Notification set!", { description: "We'll text you when you're admitted." });
                      setShowSmsForm(false);
                    }}
                    className="bg-[#2563EB] text-white px-6 py-2 rounded-xl font-bold text-sm hover:bg-[#1d4ed8]"
                  >
                    Set Alert
                  </button>
                </div>
                <p className="text-[10px] text-[#6B7280] mt-3 uppercase tracking-wide">Standard network rates apply. We'll only text you once.</p>
              </motion.div>
            )}

            <div className="flex items-center justify-center gap-6 pt-4">
              <div className="flex flex-col items-center gap-1 opacity-60">
                <ShieldCheck className="w-5 h-5 text-[#10B981]" />
                <span className="text-[10px] font-bold uppercase">Secure</span>
              </div>
              <div className="flex flex-col items-center gap-1 opacity-60">
                <CheckCircle2 className="w-5 h-5 text-[#10B981]" />
                <span className="text-[10px] font-bold uppercase">Verified</span>
              </div>
            </div>
          </motion.div>
        ) : (
          <motion.div
            key="admitted"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            className="w-full text-center space-y-8"
          >
            <div className="w-24 h-24 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-8 border-4 border-white shadow-xl">
              <CheckCircle2 className="w-12 h-12 text-[#10B981]" />
            </div>
            <div className="space-y-4">
              <h1 className="text-4xl font-black text-[#111827]">It's Your Turn!</h1>
              <p className="text-[#6B7280] text-lg max-w-md mx-auto">
                Your spot has been secured. You have exactly 5 minutes to select your tickets and complete the payment.
              </p>
            </div>

            <div className="bg-white border-2 border-[#10B981] rounded-3xl p-8 shadow-xl max-w-md mx-auto">
              <div className="flex items-center justify-between mb-6">
                <span className="text-sm font-bold text-[#6B7280] uppercase tracking-widest">Time Remaining</span>
                <span className="text-2xl font-black text-[#EF4444] tabular-nums">04:59</span>
              </div>
              <button
                onClick={() => navigate("/select-tickets")}
                className="w-full bg-[#10B981] text-white py-4 px-8 rounded-2xl font-bold text-lg hover:bg-green-600 transition-all shadow-lg shadow-green-500/25 flex items-center justify-center gap-3 group"
              >
                Proceed to Selection
                <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
              </button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
