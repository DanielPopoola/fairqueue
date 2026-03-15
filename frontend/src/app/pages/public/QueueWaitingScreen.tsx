import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router';
import { motion, AnimatePresence } from 'motion/react';
import { Info, Bell, Users, Clock, AlertTriangle } from 'lucide-react';
import { Card, CardContent } from '../../components/ui/Card';
import { Progress } from '../../components/ui/Progress';
import { Button } from '../../components/ui/Button';

export function QueueWaitingScreen() {
  const { eventId } = useParams();
  const navigate = useNavigate();
  const [position, setPosition] = useState(1547);
  const [initialPosition] = useState(1547);
  
  useEffect(() => {
    // Simulate queue moving
    const interval = setInterval(() => {
      setPosition((prev) => {
        const newPos = prev - Math.floor(Math.random() * 5 + 1);
        if (newPos <= 0) {
          clearInterval(interval);
          setTimeout(() => navigate(`/buy/${eventId}`), 3000); // Redirect to buy after 3s
          return 0;
        }
        return newPos;
      });
    }, 5000);
    return () => clearInterval(interval);
  }, [eventId, navigate]);

  const progressValue = ((initialPosition - position) / initialPosition) * 100;
  const estimatedWaitTime = Math.ceil(position / 120); // rough estimate

  return (
    <div className="max-w-xl mx-auto space-y-8 min-h-[70vh] flex flex-col justify-center py-12">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold">You are in line</h1>
        <p className="text-[var(--color-text-secondary)]">Burna Boy Live in Lagos</p>
      </div>

      <AnimatePresence mode="wait">
        {position > 0 ? (
          <motion.div
            key="waiting"
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 1.05 }}
            className="space-y-6"
          >
            <Card className="border-[var(--color-primary)] shadow-lg overflow-hidden relative">
              <div className="absolute top-0 left-0 w-full h-1 bg-[var(--color-primary)] animate-pulse" />
              <CardContent className="p-8 sm:p-12 text-center space-y-8">
                <div>
                  <div className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)] mb-2">Your Position</div>
                  <motion.div 
                    key={position}
                    initial={{ y: -10, opacity: 0 }}
                    animate={{ y: 0, opacity: 1 }}
                    className="text-6xl sm:text-7xl font-bold text-[var(--color-primary)] tracking-tighter"
                  >
                    #{position.toLocaleString()}
                  </motion.div>
                </div>

                <div className="space-y-4">
                  <Progress value={progressValue} className="h-3" />
                  <div className="flex justify-between items-center text-sm font-medium">
                    <span className="flex items-center gap-1.5 text-[var(--color-text-secondary)]">
                      <Clock className="w-4 h-4" />
                      Wait: ~{estimatedWaitTime} min
                    </span>
                    <span className="text-[var(--color-primary)] animate-pulse">
                      Queue moving...
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 flex items-start gap-3 text-amber-800">
              <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5" />
              <div className="text-sm leading-relaxed">
                <span className="font-semibold">Keep this page open.</span> If you leave or refresh, you'll lose your spot. We're admitting users at a controlled rate to prevent the system from crashing.
              </div>
            </div>

            <Card>
              <CardContent className="p-6 flex flex-col sm:flex-row items-center gap-4">
                <div className="bg-blue-100 p-3 rounded-full text-[var(--color-primary)] shrink-0">
                  <Bell className="w-6 h-6" />
                </div>
                <div className="flex-1 text-center sm:text-left">
                  <h3 className="font-semibold">Get notified</h3>
                  <p className="text-sm text-[var(--color-text-secondary)]">We can SMS you when it's your turn.</p>
                </div>
                <Button variant="secondary" className="w-full sm:w-auto shrink-0">
                  Notify Me
                </Button>
              </CardContent>
            </Card>
          </motion.div>
        ) : (
          <motion.div
            key="admitted"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            className="text-center space-y-6"
          >
            <div className="w-24 h-24 bg-green-100 rounded-full flex items-center justify-center mx-auto text-green-600 mb-6 shadow-inner">
              <Users className="w-12 h-12 animate-bounce" />
            </div>
            <h2 className="text-4xl font-bold text-[var(--color-text-primary)]">It's your turn!</h2>
            <p className="text-xl text-[var(--color-text-secondary)] max-w-sm mx-auto">
              Redirecting you to the ticket selection page...
            </p>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
