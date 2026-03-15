import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router';
import { Clock, Info, AlertCircle, ShoppingCart } from 'lucide-react';
import { Button } from '../../components/ui/Button';
import { Card, CardContent } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';

export function InventorySelectionScreen() {
  const { eventId } = useParams();
  const navigate = useNavigate();
  const [timeLeft, setTimeLeft] = useState(300); // 5 minutes
  const [selections, setSelections] = useState<Record<string, number>>({});

  useEffect(() => {
    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          navigate('/error?reason=timeout');
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [navigate]);

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  const tickets = [
    { id: 'vip', name: 'VIP Section', price: 50000, left: 23, max: 4 },
    { id: 'regular', name: 'Regular', price: 25000, left: 347, max: 4 },
    { id: 'early', name: 'Early Bird', price: 15000, left: 0, max: 4 },
  ];

  const handleSelect = (id: string, qty: number) => {
    setSelections(prev => ({ ...prev, [id]: qty }));
  };

  const total = tickets.reduce((sum, ticket) => {
    return sum + (selections[ticket.id] || 0) * ticket.price;
  }, 0);

  const totalTickets = Object.values(selections).reduce((sum, qty) => sum + qty, 0);

  return (
    <div className="max-w-4xl mx-auto space-y-6 animate-in fade-in duration-300">
      {/* Timer Bar */}
      <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 flex items-center justify-between sticky top-16 z-20 shadow-sm">
        <div className="flex items-center gap-2 text-amber-800 font-semibold">
          <Clock className="w-5 h-5" />
          <span>Time Remaining</span>
        </div>
        <div className="text-xl font-bold text-amber-600 tracking-wider font-mono">
          {formatTime(timeLeft)}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-4">
          <h2 className="text-2xl font-bold">Select Tickets</h2>
          <p className="text-[var(--color-text-secondary)]">Max 4 tickets per person. You have {formatTime(timeLeft)} to complete purchase.</p>

          <div className="space-y-4">
            {tickets.map((ticket) => (
              <Card key={ticket.id} className={`overflow-hidden transition-all ${ticket.left === 0 ? 'opacity-50 grayscale' : 'hover:border-blue-200'}`}>
                <CardContent className="p-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-lg">{ticket.name}</h3>
                      {ticket.left > 0 && ticket.left < 50 && (
                        <Badge variant="warning">Selling Fast</Badge>
                      )}
                      {ticket.left === 0 && (
                        <Badge variant="error">Sold Out</Badge>
                      )}
                    </div>
                    <div className="text-xl font-bold">₦{ticket.price.toLocaleString()}</div>
                    {ticket.left > 0 ? (
                      <div className="text-sm text-[var(--color-text-secondary)] flex items-center gap-1">
                        <Info className="w-4 h-4" />
                        {ticket.left} left
                      </div>
                    ) : null}
                  </div>

                  <div className="w-full sm:w-auto mt-4 sm:mt-0">
                    <select
                      className="w-full sm:w-32 h-10 rounded-lg border border-[var(--color-border)] bg-white px-3 focus:ring-2 focus:ring-[var(--color-primary)] outline-none disabled:opacity-50"
                      value={selections[ticket.id] || 0}
                      onChange={(e) => handleSelect(ticket.id, parseInt(e.target.value))}
                      disabled={ticket.left === 0}
                    >
                      {[...Array(ticket.max + 1)].map((_, i) => (
                        <option key={i} value={i}>{i} {i === 1 ? 'ticket' : 'tickets'}</option>
                      ))}
                    </select>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>

        <div className="lg:col-span-1">
          <Card className="sticky top-40">
            <CardContent className="p-6 space-y-6">
              <h3 className="text-xl font-bold flex items-center gap-2 border-b border-[var(--color-border)] pb-4">
                <ShoppingCart className="w-5 h-5" /> Order Summary
              </h3>

              <div className="space-y-3 min-h-[100px]">
                {tickets.map((ticket) => {
                  const qty = selections[ticket.id];
                  if (!qty) return null;
                  return (
                    <div key={ticket.id} className="flex justify-between items-start text-sm">
                      <div>
                        <div className="font-medium">{ticket.name}</div>
                        <div className="text-[var(--color-text-secondary)]">{qty} × ₦{ticket.price.toLocaleString()}</div>
                      </div>
                      <div className="font-semibold">₦{(qty * ticket.price).toLocaleString()}</div>
                    </div>
                  );
                })}
                {totalTickets === 0 && (
                  <div className="text-center text-[var(--color-text-secondary)] py-6 italic">
                    No tickets selected yet
                  </div>
                )}
              </div>

              <div className="border-t border-[var(--color-border)] pt-4">
                <div className="flex justify-between items-center mb-6">
                  <div className="font-semibold text-lg">Subtotal</div>
                  <div className="text-2xl font-bold text-[var(--color-primary)]">₦{total.toLocaleString()}</div>
                </div>

                <Button 
                  asChild 
                  size="lg" 
                  className="w-full" 
                  disabled={totalTickets === 0}
                >
                  <Link to={totalTickets > 0 ? `/payment/${eventId}` : "#"}>
                    Proceed to Payment
                  </Link>
                </Button>
              </div>

              <div className="bg-gray-50 p-3 rounded-lg flex gap-2 text-xs text-[var(--color-text-secondary)] mt-4 items-start">
                <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-gray-400" />
                <p>If time runs out, your spot goes to the next person. No items are reserved until payment completes.</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
