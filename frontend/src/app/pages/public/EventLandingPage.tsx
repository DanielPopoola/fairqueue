import React from 'react';
import { useParams, Link } from 'react-router';
import { Calendar, MapPin, Users, ShieldCheck, Ticket } from 'lucide-react';
import { Button } from '../components/ui/Button';
import { Card, CardContent } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';

export function EventLandingPage() {
  const { id } = useParams();
  
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      {/* Hero Section */}
      <div className="relative h-64 sm:h-96 rounded-2xl overflow-hidden shadow-lg">
        <img 
          src="https://images.unsplash.com/photo-1540039155732-67503716a4b1?ixlib=rb-4.0.3&auto=format&fit=crop&w=1200&q=80" 
          alt="Concert Crowd" 
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/40 to-transparent" />
        <div className="absolute bottom-0 left-0 p-6 sm:p-8 w-full">
          <Badge variant="success" className="mb-3">Selling Fast</Badge>
          <h1 className="text-3xl sm:text-5xl font-bold text-white mb-2 leading-tight">Burna Boy Live in Lagos</h1>
          <div className="flex flex-wrap gap-4 text-white/90 text-sm sm:text-base">
            <span className="flex items-center gap-1.5"><Calendar className="w-4 h-4" /> June 1, 2024 • 7:00 PM</span>
            <span className="flex items-center gap-1.5"><MapPin className="w-4 h-4" /> Eko Convention Centre</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-8">
          <section>
            <h2 className="text-2xl font-bold mb-4">About the Event</h2>
            <p className="text-[var(--color-text-secondary)] leading-relaxed">
              Experience the African Giant live in his hometown. This highly anticipated concert will feature 
              performances from his latest album alongside all his classic hits. 
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">How FairQueue Works</h2>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              {[
                { title: "Join Queue", desc: "Click the button to secure your spot in the virtual line.", icon: Users },
                { title: "Wait Calmly", desc: "Keep the page open. We'll show you exactly when it's your turn.", icon: Calendar },
                { title: "Buy Tickets", desc: "You have 5 minutes to complete your purchase stress-free.", icon: Ticket }
              ].map((step, i) => (
                <div key={i} className="text-center p-4 rounded-xl bg-gray-50 border border-gray-100">
                  <div className="w-12 h-12 rounded-full bg-blue-100 text-[var(--color-primary)] flex items-center justify-center mx-auto mb-3">
                    <step.icon className="w-6 h-6" />
                  </div>
                  <h3 className="font-semibold mb-1">{step.title}</h3>
                  <p className="text-xs text-[var(--color-text-secondary)]">{step.desc}</p>
                </div>
              ))}
            </div>
          </section>
        </div>

        {/* Sidebar Action */}
        <div className="lg:col-span-1">
          <Card className="sticky top-24 border-2 border-[var(--color-primary)] shadow-xl">
            <CardContent className="p-6">
              <div className="text-center mb-6">
                <div className="text-[var(--color-text-secondary)] mb-1">Tickets starting at</div>
                <div className="text-3xl font-bold">₦25,000</div>
              </div>

              <div className="space-y-4">
                <Button asChild size="lg" className="w-full text-lg shadow-md animate-pulse">
                  <Link to="/queue/1">On Sale Now - Join Queue</Link>
                </Button>
                
                <div className="flex items-center justify-center gap-2 text-sm text-[var(--color-text-secondary)]">
                  <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                  2,347 people in queue right now
                </div>
              </div>

              <div className="mt-6 pt-6 border-t border-[var(--color-border)]">
                <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)] bg-blue-50 p-3 rounded-lg">
                  <ShieldCheck className="w-5 h-5 text-[var(--color-primary)] shrink-0" />
                  <span>Powered by <span className="font-semibold text-[var(--color-primary)]">FairQueue</span>. Fair allocation guaranteed.</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
