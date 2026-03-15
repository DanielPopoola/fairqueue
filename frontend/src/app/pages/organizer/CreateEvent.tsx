import { useState } from "react";
import { useNavigate } from "react-router";
import { 
  Calendar, 
  MapPin, 
  Ticket, 
  Settings, 
  Check, 
  ChevronRight, 
  ChevronLeft, 
  Plus, 
  Trash2, 
  Info, 
  ShieldCheck, 
  Activity, 
  DollarSign, 
  Layers 
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { toast } from "sonner";
import { clsx } from "clsx";

export function CreateEvent() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    date: "",
    time: "",
    venue: "",
    tickets: [{ name: "General Admission", price: 25000, quantity: 1000 }],
    admissionRate: 100,
    strategy: "first-come",
    maxTickets: 4,
  });

  const steps = [
    { id: 1, name: "Basic Info", icon: Layers },
    { id: 2, name: "Ticketing", icon: Ticket },
    { id: 3, name: "Queue Config", icon: Settings },
    { id: 4, name: "Review", icon: Check },
  ];

  const handleNext = () => {
    if (step < 4) setStep(step + 1);
    else {
      toast.success("Event created successfully!", {
        description: "Your event is now live and waiting for the sale to start.",
      });
      navigate("/organizer");
    }
  };

  const handleBack = () => {
    if (step > 1) setStep(step - 1);
  };

  const addTicket = () => {
    setFormData({
      ...formData,
      tickets: [...formData.tickets, { name: "", price: 0, quantity: 0 }],
    });
  };

  const removeTicket = (index: number) => {
    const newTickets = [...formData.tickets];
    newTickets.splice(index, 1);
    setFormData({ ...formData, tickets: newTickets });
  };

  const totalGross = formData.tickets.reduce((acc, t) => acc + (t.price * t.quantity), 0);
  const serviceFee = totalGross * 0.03;
  const netEarnings = totalGross - serviceFee;

  return (
    <div className="flex-1 bg-[#F9FAFB] max-w-5xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8 md:py-12 space-y-12">
      {/* Header */}
      <div className="space-y-4">
        <button
          onClick={() => navigate("/organizer")}
          className="inline-flex items-center gap-2 text-[#6B7280] font-bold text-sm hover:text-[#111827] group"
        >
          <ChevronLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
          Dashboard
        </button>
        <h1 className="text-3xl font-black text-[#111827]">Create New Event</h1>
      </div>

      {/* Stepper */}
      <div className="flex justify-between items-center max-w-2xl mx-auto relative px-4">
        <div className="absolute top-1/2 left-0 right-0 h-0.5 bg-[#E5E7EB] -translate-y-1/2 -z-10" />
        {steps.map((s) => (
          <div key={s.id} className="flex flex-col items-center gap-2 bg-[#F9FAFB] px-2 relative z-10">
            <div className={clsx(
              "w-10 h-10 rounded-full flex items-center justify-center font-bold text-sm border-2 transition-all",
              step === s.id ? "bg-[#2563EB] text-white border-[#2563EB]" : 
              step > s.id ? "bg-green-500 text-white border-green-500" : "bg-white text-[#6B7280] border-[#E5E7EB]"
            )}>
              {step > s.id ? <Check className="w-5 h-5" /> : s.id}
            </div>
            <span className={clsx(
              "text-[10px] font-black uppercase tracking-widest hidden sm:block",
              step === s.id ? "text-[#2563EB]" : "text-[#6B7280]"
            )}>
              {s.name}
            </span>
          </div>
        ))}
      </div>

      {/* Form Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-12 items-start">
        <div className="lg:col-span-2 space-y-8">
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div
                key="step1"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="bg-white border border-[#E5E7EB] rounded-3xl p-8 shadow-sm space-y-6"
              >
                <div className="space-y-4">
                  <h2 className="text-xl font-bold text-[#111827]">Basic Information</h2>
                  <div className="grid grid-cols-1 gap-6">
                    <div className="space-y-2">
                      <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Event Name</label>
                      <input
                        type="text"
                        placeholder="e.g. Burna Boy Live in Lagos"
                        className="w-full px-4 py-3 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
                        value={formData.name}
                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Description</label>
                      <textarea
                        rows={4}
                        placeholder="Tell your audience what to expect..."
                        className="w-full px-4 py-3 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all resize-none"
                        value={formData.description}
                        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                      />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Date</label>
                        <input
                          type="date"
                          className="w-full px-4 py-3 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
                          value={formData.date}
                          onChange={(e) => setFormData({ ...formData, date: e.target.value })}
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Time</label>
                        <input
                          type="time"
                          className="w-full px-4 py-3 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
                          value={formData.time}
                          onChange={(e) => setFormData({ ...formData, time: e.target.value })}
                        />
                      </div>
                    </div>
                    <div className="space-y-2">
                      <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Venue</label>
                      <div className="relative">
                        <MapPin className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-[#6B7280]" />
                        <input
                          type="text"
                          placeholder="Search for a location..."
                          className="w-full pl-11 pr-4 py-3 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
                          value={formData.venue}
                          onChange={(e) => setFormData({ ...formData, venue: e.target.value })}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 2 && (
              <motion.div
                key="step2"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="bg-white border border-[#E5E7EB] rounded-3xl p-8 shadow-sm space-y-6"
              >
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <h2 className="text-xl font-bold text-[#111827]">Ticketing Strategy</h2>
                    <button
                      onClick={addTicket}
                      className="flex items-center gap-2 text-xs font-bold text-[#2563EB] hover:text-[#1d4ed8]"
                    >
                      <Plus className="w-4 h-4" />
                      Add Ticket Type
                    </button>
                  </div>
                  <div className="space-y-4">
                    {formData.tickets.map((ticket, index) => (
                      <div key={index} className="p-6 bg-gray-50 border border-[#E5E7EB] rounded-2xl space-y-4 relative group">
                        <button
                          onClick={() => removeTicket(index)}
                          className="absolute top-4 right-4 text-[#6B7280] hover:text-[#EF4444] opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div className="space-y-2">
                            <label className="text-[10px] font-bold text-[#6B7280] uppercase tracking-widest">Ticket Name</label>
                            <input
                              type="text"
                              placeholder="e.g. VIP"
                              className="w-full px-4 py-2 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 outline-none"
                              value={ticket.name}
                              onChange={(e) => {
                                const newTickets = [...formData.tickets];
                                newTickets[index].name = e.target.value;
                                setFormData({ ...formData, tickets: newTickets });
                              }}
                            />
                          </div>
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <label className="text-[10px] font-bold text-[#6B7280] uppercase tracking-widest">Price (₦)</label>
                              <input
                                type="number"
                                className="w-full px-4 py-2 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 outline-none"
                                value={ticket.price}
                                onChange={(e) => {
                                  const newTickets = [...formData.tickets];
                                  newTickets[index].price = parseInt(e.target.value) || 0;
                                  setFormData({ ...formData, tickets: newTickets });
                                }}
                              />
                            </div>
                            <div className="space-y-2">
                              <label className="text-[10px] font-bold text-[#6B7280] uppercase tracking-widest">Quantity</label>
                              <input
                                type="number"
                                className="w-full px-4 py-2 border border-[#D1D5DB] rounded-xl focus:ring-2 focus:ring-blue-500 outline-none"
                                value={ticket.quantity}
                                onChange={(e) => {
                                  const newTickets = [...formData.tickets];
                                  newTickets[index].quantity = parseInt(e.target.value) || 0;
                                  setFormData({ ...formData, tickets: newTickets });
                                }}
                              />
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div
                key="step3"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="bg-white border border-[#E5E7EB] rounded-3xl p-8 shadow-sm space-y-8"
              >
                <div className="space-y-4">
                  <h2 className="text-xl font-bold text-[#111827]">Queue Configuration</h2>
                  <p className="text-sm text-[#6B7280]">Fine-tune how FairQueue handles your high-demand sale.</p>
                </div>

                <div className="space-y-6">
                  <div className="space-y-4">
                    <div className="flex justify-between items-center">
                      <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest flex items-center gap-2">
                        Admission Rate
                        <Info className="w-3 h-3 text-[#6B7280]" />
                      </label>
                      <span className="text-sm font-black text-[#2563EB]">{formData.admissionRate} users/min</span>
                    </div>
                    <input
                      type="range"
                      min="10"
                      max="1000"
                      step="10"
                      className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-[#2563EB]"
                      value={formData.admissionRate}
                      onChange={(e) => setFormData({ ...formData, admissionRate: parseInt(e.target.value) })}
                    />
                    <div className="flex justify-between text-[10px] font-bold text-[#6B7280] uppercase tracking-widest opacity-50 px-1">
                      <span>Low Load (10)</span>
                      <span>High Load (1000)</span>
                    </div>
                    <div className="p-3 bg-blue-50 rounded-xl border border-blue-100 flex gap-3 items-center">
                      <Activity className="w-4 h-4 text-[#2563EB]" />
                      <p className="text-xs text-blue-700 font-medium leading-relaxed">
                        At <span className="font-bold">{formData.admissionRate} users/min</span>, your sale will take approximately <span className="font-bold">{Math.ceil(formData.tickets.reduce((acc, t) => acc + t.quantity, 0) / (formData.admissionRate || 1))} mins</span> to complete if fully sold out.
                      </p>
                    </div>
                  </div>

                  <div className="space-y-4 pt-4">
                    <label className="text-xs font-bold text-[#6B7280] uppercase tracking-widest">Allocation Strategy</label>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      {[
                        { id: "first-come", name: "First-Come, First-Served", desc: "Fair and transparent for early birds." },
                        { id: "random", name: "Randomized Entry", desc: "Prevents bot advantage (Advanced).", badge: "PRO" },
                      ].map((strat) => (
                        <button
                          key={strat.id}
                          onClick={() => setFormData({ ...formData, strategy: strat.id })}
                          className={clsx(
                            "p-4 border-2 rounded-2xl text-left transition-all",
                            formData.strategy === strat.id ? "border-[#2563EB] bg-[#2563EB]/5" : "border-[#E5E7EB] bg-white hover:border-gray-300"
                          )}
                        >
                          <div className="flex justify-between items-center mb-1">
                            <span className="font-bold text-sm text-[#111827]">{strat.name}</span>
                            {strat.badge && <span className="text-[8px] font-black bg-purple-100 text-purple-600 px-1.5 py-0.5 rounded uppercase tracking-widest">{strat.badge}</span>}
                          </div>
                          <p className="text-xs text-[#6B7280]">{strat.desc}</p>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 4 && (
              <motion.div
                key="step4"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="bg-white border border-[#E5E7EB] rounded-3xl p-8 shadow-sm space-y-8"
              >
                <div className="text-center space-y-2">
                  <div className="w-16 h-16 bg-blue-50 text-[#2563EB] rounded-full flex items-center justify-center mx-auto mb-4 border border-blue-100">
                    <ShieldCheck className="w-8 h-8" />
                  </div>
                  <h2 className="text-2xl font-black text-[#111827]">Ready to Go!</h2>
                  <p className="text-[#6B7280]">Review your event configuration one last time.</p>
                </div>

                <div className="divide-y divide-[#E5E7EB]">
                  <div className="py-4 flex justify-between">
                    <span className="text-sm font-bold text-[#6B7280]">Event Name</span>
                    <span className="text-sm font-black text-[#111827]">{formData.name || "Untitled Event"}</span>
                  </div>
                  <div className="py-4 flex justify-between">
                    <span className="text-sm font-bold text-[#6B7280]">Date & Time</span>
                    <span className="text-sm font-black text-[#111827]">{formData.date} • {formData.time}</span>
                  </div>
                  <div className="py-4 flex justify-between">
                    <span className="text-sm font-bold text-[#6B7280]">Total Inventory</span>
                    <span className="text-sm font-black text-[#111827]">{formData.tickets.reduce((acc, t) => acc + t.quantity, 0).toLocaleString()} tickets</span>
                  </div>
                  <div className="py-4 flex justify-between">
                    <span className="text-sm font-bold text-[#6B7280]">Admission Rate</span>
                    <span className="text-sm font-black text-[#2563EB]">{formData.admissionRate} users/min</span>
                  </div>
                </div>

                <div className="bg-[#2563EB]/5 border border-[#2563EB]/20 rounded-2xl p-6">
                  <div className="flex items-center gap-3 mb-4">
                    <DollarSign className="w-5 h-5 text-[#2563EB]" />
                    <h3 className="font-bold text-[#111827]">Earnings Forecast</h3>
                  </div>
                  <div className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-[#6B7280]">Gross Revenue</span>
                      <span className="font-bold text-[#111827]">₦{totalGross.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-[#6B7280]">FairQueue Fee (3%)</span>
                      <span className="font-bold text-[#EF4444]">-₦{serviceFee.toLocaleString()}</span>
                    </div>
                    <div className="h-px bg-[#E5E7EB] my-2" />
                    <div className="flex justify-between">
                      <span className="font-bold text-[#111827]">Net Payout</span>
                      <span className="text-xl font-black text-[#10B981]">₦{netEarnings.toLocaleString()}</span>
                    </div>
                  </div>
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          <div className="flex justify-between pt-4">
            <button
              onClick={handleBack}
              disabled={step === 1}
              className="px-8 py-3 rounded-xl font-bold text-sm text-[#6B7280] hover:text-[#111827] disabled:opacity-0 transition-all flex items-center gap-2"
            >
              <ChevronLeft className="w-4 h-4" />
              Back
            </button>
            <button
              onClick={handleNext}
              className="bg-[#2563EB] text-white px-10 py-3 rounded-xl font-bold text-sm hover:bg-[#1d4ed8] transition-all shadow-lg shadow-blue-500/20 flex items-center gap-2"
            >
              {step === 4 ? "Publish Event" : "Continue"}
              {step < 4 && <ChevronRight className="w-4 h-4" />}
            </button>
          </div>
        </div>

        <div className="space-y-6">
          <div className="bg-white border border-[#E5E7EB] rounded-3xl p-6 shadow-sm space-y-4">
            <h3 className="font-bold text-[#111827]">Tips for Success</h3>
            <ul className="space-y-4">
              <li className="flex gap-3 text-xs text-[#6B7280] leading-relaxed">
                <div className="w-5 h-5 bg-green-50 text-green-600 rounded-full flex items-center justify-center flex-shrink-0">
                  <Check className="w-3 h-3" />
                </div>
                Admissions rates between 100-200 are optimal for high-capacity venues.
              </li>
              <li className="flex gap-3 text-xs text-[#6B7280] leading-relaxed">
                <div className="w-5 h-5 bg-green-50 text-green-600 rounded-full flex items-center justify-center flex-shrink-0">
                  <Check className="w-3 h-3" />
                </div>
                Upload a high-quality hero image to build trust with your buyers.
              </li>
              <li className="flex gap-3 text-xs text-[#6B7280] leading-relaxed">
                <div className="w-5 h-5 bg-green-50 text-green-600 rounded-full flex items-center justify-center flex-shrink-0">
                  <Check className="w-3 h-3" />
                </div>
                You can adjust admission rates live while the sale is ongoing.
              </li>
            </ul>
          </div>

          <div className="p-6 border border-dashed border-[#E5E7EB] rounded-3xl text-center space-y-4">
            <div className="w-12 h-12 bg-gray-100 rounded-2xl flex items-center justify-center mx-auto text-[#6B7280]">
              <Layers className="w-6 h-6" />
            </div>
            <div>
              <p className="text-xs font-bold text-[#111827]">Event Preview</p>
              <p className="text-[10px] text-[#6B7280]">See what your customers will see before you publish.</p>
            </div>
            <button className="w-full py-2 bg-gray-50 text-[#111827] rounded-lg font-bold text-xs border border-[#E5E7EB] hover:bg-white transition-colors">
              Open Live Preview
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
