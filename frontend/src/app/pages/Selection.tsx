import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { motion, AnimatePresence } from "motion/react";
import { Minus, Plus, ShoppingCart, Clock, ArrowRight, Info, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { clsx } from "clsx";

export function Selection() {
  const navigate = useNavigate();
  const [timeLeft, setTimeLeft] = useState(272); // 4:32 in seconds
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  
  const ticketTypes = [
    { id: "vip", name: "VIP Section", price: 50000, description: "Includes priority entry and exclusive bar access.", availability: 23, max: 4 },
    { id: "regular", name: "Regular Admission", price: 25000, description: "General entry to the main concert arena.", availability: 347, max: 4 },
    { id: "earlybird", name: "Early Bird", price: 15000, description: "Limited availability discounted tickets.", availability: 0, max: 0 },
  ];

  useEffect(() => {
    const timer = setInterval(() => {
      setTimeLeft((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  };

  const handleUpdateQuantity = (id: string, delta: number, max: number) => {
    setQuantities((prev) => {
      const current = prev[id] || 0;
      const next = Math.max(0, Math.min(max, current + delta));
      return { ...prev, [id]: next };
    });
  };

  const totalPrice = Object.entries(quantities).reduce((acc, [id, qty]) => {
    const ticket = ticketTypes.find((t) => t.id === id);
    return acc + (ticket?.price || 0) * qty;
  }, 0);

  const totalQty = Object.values(quantities).reduce((acc, qty) => acc + qty, 0);

  const handleProceed = () => {
    if (totalQty === 0) {
      toast.error("Please select at least one ticket.");
      return;
    }
    navigate("/payment", { state: { quantities, totalPrice, totalQty } });
  };

  return (
    <div className="flex-1 max-w-4xl mx-auto w-full px-4 py-8 md:py-12">
      <div className="sticky top-[64px] z-40 bg-[#F9FAFB] pb-6 mb-6">
        <div className="flex flex-col md:flex-row justify-between items-center bg-white border border-[#E5E7EB] rounded-2xl p-4 shadow-sm gap-4">
          <div className="flex items-center gap-3">
            <div className={clsx(
              "p-2 rounded-lg",
              timeLeft < 60 ? "bg-red-50" : "bg-blue-50"
            )}>
              <Clock className={clsx("w-5 h-5", timeLeft < 60 ? "text-[#EF4444]" : "text-[#2563EB]")} />
            </div>
            <div>
              <span className="text-xs font-bold uppercase tracking-widest text-[#6B7280]">Time to Checkout</span>
              <span className={clsx(
                "block text-xl font-black tabular-nums",
                timeLeft < 60 ? "text-[#EF4444] animate-pulse" : "text-[#111827]"
              )}>
                {formatTime(timeLeft)}
              </span>
            </div>
          </div>

          <div className="h-px w-full md:h-8 md:w-px bg-[#E5E7EB]" />

          <div className="flex flex-col items-center md:items-end">
            <span className="text-xs font-bold uppercase tracking-widest text-[#6B7280]">Order Total</span>
            <span className="text-2xl font-black text-[#111827]">₦{totalPrice.toLocaleString()}</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 items-start">
        <div className="lg:col-span-2 space-y-6">
          <h1 className="text-2xl font-bold text-[#111827]">Select Ticket Types</h1>
          
          <div className="space-y-4">
            {ticketTypes.map((ticket) => {
              const isSoldOut = ticket.availability === 0;
              const qty = quantities[ticket.id] || 0;

              return (
                <div
                  key={ticket.id}
                  className={clsx(
                    "relative bg-white border-2 rounded-2xl p-6 transition-all",
                    qty > 0 ? "border-[#2563EB] shadow-lg shadow-blue-500/5" : "border-[#E5E7EB]",
                    isSoldOut && "opacity-60 grayscale"
                  )}
                >
                  <div className="flex justify-between items-start gap-4">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-bold text-lg text-[#111827]">{ticket.name}</h3>
                        {isSoldOut ? (
                          <span className="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 bg-gray-100 text-gray-500 rounded-full border border-gray-200">SOLD OUT</span>
                        ) : (
                          <span className="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 bg-green-50 text-green-600 rounded-full border border-green-100">
                            {ticket.availability} Left
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-[#6B7280] leading-relaxed pr-4">
                        {ticket.description}
                      </p>
                      <span className="block font-black text-[#2563EB] text-xl mt-4">
                        ₦{ticket.price.toLocaleString()}
                      </span>
                    </div>

                    {!isSoldOut && (
                      <div className="flex items-center gap-3 bg-gray-50 p-2 rounded-xl border border-gray-100">
                        <button
                          onClick={() => handleUpdateQuantity(ticket.id, -1, ticket.max)}
                          disabled={qty === 0}
                          className="w-10 h-10 flex items-center justify-center rounded-lg bg-white border border-gray-200 text-[#111827] disabled:opacity-30 hover:border-[#2563EB] hover:text-[#2563EB] transition-colors shadow-sm"
                        >
                          <Minus className="w-4 h-4" />
                        </button>
                        <span className="w-6 text-center font-black text-lg tabular-nums">
                          {qty}
                        </span>
                        <button
                          onClick={() => handleUpdateQuantity(ticket.id, 1, ticket.max)}
                          disabled={qty >= ticket.max}
                          className="w-10 h-10 flex items-center justify-center rounded-lg bg-white border border-gray-200 text-[#111827] disabled:opacity-30 hover:border-[#2563EB] hover:text-[#2563EB] transition-colors shadow-sm"
                        >
                          <Plus className="w-4 h-4" />
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          <div className="bg-blue-50/50 border border-blue-100 rounded-2xl p-4 flex gap-3 items-start">
            <Info className="w-5 h-5 text-[#2563EB] flex-shrink-0 mt-0.5" />
            <p className="text-sm text-[#1e40af]">
              You can select a maximum of 4 tickets per person. Your selection is not reserved until you complete the payment process.
            </p>
          </div>
        </div>

        <div className="space-y-6 lg:sticky lg:top-40">
          <div className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm space-y-6">
            <h2 className="font-bold text-lg text-[#111827] flex items-center gap-2">
              <ShoppingCart className="w-5 h-5" />
              Order Summary
            </h2>

            <div className="space-y-4">
              {Object.entries(quantities).map(([id, qty]) => {
                if (qty === 0) return null;
                const ticket = ticketTypes.find((t) => t.id === id);
                return (
                  <div key={id} className="flex justify-between items-center text-sm">
                    <div className="text-[#6B7280]">
                      <span className="font-bold text-[#111827]">{qty}x</span> {ticket?.name}
                    </div>
                    <span className="font-semibold text-[#111827]">₦{((ticket?.price || 0) * qty).toLocaleString()}</span>
                  </div>
                );
              })}
              {totalQty === 0 && (
                <p className="text-sm text-[#6B7280] italic text-center py-4 border-2 border-dashed border-gray-100 rounded-xl">
                  No tickets selected yet
                </p>
              )}
            </div>

            <div className="h-px bg-[#E5E7EB]" />

            <div className="flex justify-between items-center">
              <span className="font-bold text-[#111827]">Total</span>
              <span className="text-2xl font-black text-[#2563EB]">₦{totalPrice.toLocaleString()}</span>
            </div>

            <button
              onClick={handleProceed}
              disabled={totalQty === 0}
              className="w-full bg-[#2563EB] text-white py-4 px-8 rounded-2xl font-bold text-lg hover:bg-[#1d4ed8] disabled:opacity-50 disabled:cursor-not-allowed hover:scale-[1.02] active:scale-[0.98] transition-all shadow-lg shadow-blue-500/25 flex items-center justify-center gap-3 group"
            >
              Checkout Now
              <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </button>
          </div>

          <div className="bg-amber-50 border border-amber-100 rounded-2xl p-4 flex gap-3 items-start">
            <AlertTriangle className="w-5 h-5 text-[#F59E0B] flex-shrink-0 mt-0.5" />
            <p className="text-xs text-[#92400E]">
              If time runs out, your spot goes to the next person in line. Please finish quickly to guarantee your tickets.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
