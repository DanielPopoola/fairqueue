import { useNavigate, useLocation } from "react-router";
import { useState, useEffect } from "react";
import { motion } from "motion/react";
import { Lock, CreditCard, ShieldCheck, ChevronLeft, CreditCard as CardIcon, Building2, Smartphone, AlertCircle, ShoppingCart, Info } from "lucide-react";
import { toast } from "sonner";
import { clsx } from "clsx";

export function Payment() {
  const navigate = useNavigate();
  const location = useLocation();
  const { totalPrice = 0, totalQty = 0, quantities = {} } = location.state || {};
  
  const [isProcessing, setIsProcessing] = useState(false);
  const [paymentMethod, setPaymentMethod] = useState("card");

  const serviceFee = totalPrice * 0.03;
  const transactionFee = 100;
  const finalTotal = totalPrice + serviceFee + transactionFee;

  const handlePayment = () => {
    setIsProcessing(true);
    // Simulate payment processing
    setTimeout(() => {
      setIsProcessing(false);
      navigate("/confirmation", { state: { finalTotal, orderId: "FQ-" + Math.floor(Math.random() * 900000 + 100000) } });
    }, 3000);
  };

  const paymentMethods = [
    { id: "card", name: "Debit/Credit Card", icon: CardIcon, description: "Visa, Mastercard, Verve" },
    { id: "bank", name: "Bank Transfer", icon: Building2, description: "Direct bank transfer" },
    { id: "ussd", name: "USSD", icon: Smartphone, description: "Dial a code on your phone" },
  ];

  if (!totalPrice) {
    navigate("/");
    return null;
  }

  return (
    <div className="flex-1 max-w-4xl mx-auto w-full px-4 py-8 md:py-12">
      <button
        onClick={() => navigate(-1)}
        className="inline-flex items-center gap-2 text-[#6B7280] font-bold text-sm hover:text-[#111827] mb-8 group"
      >
        <ChevronLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
        Back to Selection
      </button>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-start">
        <div className="space-y-8">
          <div className="space-y-4">
            <h1 className="text-3xl font-black text-[#111827]">Complete Purchase</h1>
            <p className="text-[#6B7280] text-lg">Secure payment via Paystack. Your tickets will be emailed immediately after payment.</p>
          </div>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-widest text-[#6B7280]">Select Payment Method</h3>
            <div className="grid grid-cols-1 gap-3">
              {paymentMethods.map((method) => (
                <button
                  key={method.id}
                  onClick={() => setPaymentMethod(method.id)}
                  className={clsx(
                    "flex items-center gap-4 p-4 rounded-2xl border-2 transition-all text-left",
                    paymentMethod === method.id 
                      ? "border-[#2563EB] bg-[#2563EB]/5 shadow-md shadow-blue-500/10" 
                      : "border-[#E5E7EB] bg-white hover:border-[#D1D5DB]"
                  )}
                >
                  <div className={clsx(
                    "p-3 rounded-xl",
                    paymentMethod === method.id ? "bg-[#2563EB] text-white" : "bg-gray-100 text-[#6B7280]"
                  )}>
                    <method.icon className="w-6 h-6" />
                  </div>
                  <div className="flex-1">
                    <span className="block font-bold text-[#111827]">{method.name}</span>
                    <span className="text-xs text-[#6B7280]">{method.description}</span>
                  </div>
                  {paymentMethod === method.id && (
                    <div className="w-6 h-6 rounded-full bg-[#2563EB] flex items-center justify-center">
                      <div className="w-2.5 h-2.5 rounded-full bg-white" />
                    </div>
                  )}
                </button>
              ))}
            </div>
          </div>

          <div className="p-6 bg-white border border-[#E5E7EB] rounded-3xl shadow-sm space-y-6">
            <div className="flex items-center gap-3">
              <ShieldCheck className="w-6 h-6 text-[#10B981]" />
              <div className="flex-1">
                <span className="block font-bold text-sm text-[#111827]">Secure Transaction</span>
                <span className="text-xs text-[#6B7280]">Your data is encrypted and secure</span>
              </div>
              <img src="https://paystack.com/assets/img/paystack-logo-blue.svg" alt="Paystack" className="h-6" />
            </div>

            <button
              onClick={handlePayment}
              disabled={isProcessing}
              className="w-full relative bg-[#09A5DB] text-white py-4 px-8 rounded-2xl font-bold text-lg hover:bg-[#088fbe] disabled:opacity-70 disabled:cursor-not-allowed transition-all shadow-lg shadow-[#09A5DB]/20 overflow-hidden"
            >
              {isProcessing ? (
                <div className="flex items-center justify-center gap-3">
                  <div className="w-5 h-5 border-3 border-white/30 border-t-white rounded-full animate-spin" />
                  Processing...
                </div>
              ) : (
                <div className="flex items-center justify-center gap-3">
                  <Lock className="w-5 h-5" />
                  Pay ₦{finalTotal.toLocaleString()}
                </div>
              )}
            </button>
          </div>
        </div>

        <div className="lg:sticky lg:top-32 space-y-6">
          <div className="bg-white border border-[#E5E7EB] rounded-3xl p-8 shadow-sm space-y-6">
            <h2 className="font-bold text-lg text-[#111827] flex items-center gap-2">
              <ShoppingCart className="w-5 h-5" />
              Purchase Summary
            </h2>

            <div className="space-y-6">
              <div className="space-y-3">
                <span className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Tickets</span>
                <div className="space-y-2">
                  {Object.entries(quantities).map(([id, qty]) => {
                    if (qty === 0) return null;
                    const ticketPrice = id === "vip" ? 50000 : 25000;
                    const ticketName = id === "vip" ? "VIP Section" : "Regular Admission";
                    return (
                      <div key={id} className="flex justify-between items-center text-sm">
                        <span className="text-[#6B7280] font-medium">{qty}x {ticketName}</span>
                        <span className="font-bold text-[#111827]">₦{(ticketPrice * (qty as number)).toLocaleString()}</span>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="h-px bg-[#E5E7EB]" />

              <div className="space-y-3">
                <span className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Fees & Taxes</span>
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-[#6B7280] font-medium">Service Fee (3%)</span>
                    <span className="font-bold text-[#111827]">₦{serviceFee.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-[#6B7280] font-medium">Transaction Fee</span>
                    <span className="font-bold text-[#111827]">₦{transactionFee.toLocaleString()}</span>
                  </div>
                </div>
              </div>

              <div className="h-px bg-[#E5E7EB]" />

              <div className="flex justify-between items-center pt-2">
                <div>
                  <span className="block font-black text-2xl text-[#111827]">₦{finalTotal.toLocaleString()}</span>
                  <span className="text-xs text-[#6B7280] font-medium">Total inclusive of all fees</span>
                </div>
                <div className="text-right">
                  <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-green-50 text-green-700 rounded-full text-[10px] font-bold uppercase border border-green-100 shadow-sm">
                    <div className="w-1.5 h-1.5 bg-green-500 rounded-full" />
                    Live Price
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-blue-50 border border-blue-100 rounded-2xl p-5 flex gap-4 items-start">
            <div className="p-2 bg-white rounded-lg border border-blue-100 shadow-sm">
              <Info className="w-5 h-5 text-[#2563EB]" />
            </div>
            <div>
              <p className="text-sm text-[#1e40af] font-semibold mb-1">Confirmation Email</p>
              <p className="text-xs text-[#1e40af]/80">You'll receive an email confirmation with your PDF tickets and QR codes immediately after payment completes.</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
