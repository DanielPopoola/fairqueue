import { useNavigate, useLocation } from "react-router";
import { motion } from "motion/react";
import { CheckCircle2, Download, Share2, Mail, MapPin, Calendar, Smartphone, ChevronRight, LayoutDashboard } from "lucide-react";
import { useEffect, useState } from "react";
import confetti from "canvas-confetti";

export function Confirmation() {
  const navigate = useNavigate();
  const location = useLocation();
  const { finalTotal = 0, orderId = "FQ-123456" } = location.state || {};
  const [isDownloading, setIsDownloading] = useState(false);

  useEffect(() => {
    // Trigger confetti on page load
    const duration = 3 * 1000;
    const animationEnd = Date.now() + duration;
    const defaults = { startVelocity: 30, spread: 360, ticks: 60, zIndex: 0 };

    function randomInRange(min: number, max: number) {
      return Math.random() * (max - min) + min;
    }

    const interval: any = setInterval(function() {
      const timeLeft = animationEnd - Date.now();

      if (timeLeft <= 0) {
        return clearInterval(interval);
      }

      const particleCount = 50 * (timeLeft / duration);
      confetti({ ...defaults, particleCount, origin: { x: randomInRange(0.1, 0.3), y: Math.random() - 0.2 } });
      confetti({ ...defaults, particleCount, origin: { x: randomInRange(0.7, 0.9), y: Math.random() - 0.2 } });
    }, 250);

    return () => clearInterval(interval);
  }, []);

  const handleDownload = () => {
    setIsDownloading(true);
    setTimeout(() => {
      setIsDownloading(false);
      // In a real app, trigger a download link
    }, 2000);
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center max-w-2xl mx-auto w-full px-4 py-8 md:py-16">
      <motion.div
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ type: "spring", damping: 15 }}
        className="w-full text-center space-y-12"
      >
        <div className="space-y-6">
          <div className="w-24 h-24 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-8 border-4 border-white shadow-xl">
            <CheckCircle2 className="w-12 h-12 text-[#10B981]" />
          </div>
          <div className="space-y-3">
            <h1 className="text-4xl font-black text-[#111827]">Payment Successful!</h1>
            <p className="text-[#6B7280] text-lg max-w-md mx-auto font-medium">
              You're going to see <span className="text-[#2563EB] font-bold">Burna Boy Live!</span> Your tickets have been sent to your email.
            </p>
          </div>
        </div>

        <div className="bg-white border border-[#E5E7EB] rounded-3xl overflow-hidden shadow-sm divide-y divide-[#E5E7EB]">
          <div className="p-8 space-y-6">
            <div className="flex justify-between items-center">
              <span className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Order ID</span>
              <span className="font-bold text-[#111827]">{orderId}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Total Paid</span>
              <span className="text-xl font-black text-[#2563EB]">₦{finalTotal.toLocaleString()}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Purchase Date</span>
              <span className="font-bold text-[#111827]">{new Date().toLocaleDateString("en-NG", { day: "numeric", month: "long", year: "numeric" })}</span>
            </div>
          </div>

          <div className="p-8 space-y-6 bg-gray-50/50">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="flex gap-4 items-start text-left">
                <div className="p-2 bg-white rounded-lg border border-[#E5E7EB] shadow-sm">
                  <Mail className="w-5 h-5 text-[#2563EB]" />
                </div>
                <div>
                  <p className="font-bold text-sm text-[#111827]">Confirmation Sent</p>
                  <p className="text-xs text-[#6B7280]">Check your inbox at user@example.com</p>
                </div>
              </div>
              <div className="flex gap-4 items-start text-left">
                <div className="p-2 bg-white rounded-lg border border-[#E5E7EB] shadow-sm">
                  <Smartphone className="w-5 h-5 text-[#2563EB]" />
                </div>
                <div>
                  <p className="font-bold text-sm text-[#111827]">Mobile Tickets</p>
                  <p className="text-xs text-[#6B7280]">Add to Apple or Google Wallet from email</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 w-full">
          <button
            onClick={handleDownload}
            disabled={isDownloading}
            className="flex items-center justify-center gap-3 bg-white border-2 border-[#E5E7EB] text-[#111827] py-4 rounded-2xl font-bold text-base hover:border-[#2563EB] hover:text-[#2563EB] transition-all shadow-sm group"
          >
            {isDownloading ? (
              <div className="w-5 h-5 border-2 border-[#2563EB]/20 border-t-[#2563EB] rounded-full animate-spin" />
            ) : (
              <Download className="w-5 h-5" />
            )}
            Download PDF Tickets
          </button>
          <button
            className="flex items-center justify-center gap-3 bg-[#2563EB] text-white py-4 rounded-2xl font-bold text-base hover:bg-[#1d4ed8] transition-all shadow-lg shadow-blue-500/20 group"
          >
            <Share2 className="w-5 h-5" />
            Share with Friends
          </button>
        </div>

        <div className="pt-4 border-t border-[#E5E7EB]">
          <button
            onClick={() => navigate("/")}
            className="inline-flex items-center gap-2 text-[#6B7280] font-bold text-sm hover:text-[#111827] group"
          >
            Back to Event List
            <ChevronRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
          </button>
        </div>
      </motion.div>
    </div>
  );
}
