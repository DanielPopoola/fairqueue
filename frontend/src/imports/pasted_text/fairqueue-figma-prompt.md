# Figma UI Design Prompt for FairQueue

## Context for Designer

You are designing the UI/UX for **FairQueue**, a SaaS platform that helps event organizers sell tickets and limited inventory fairly during high-demand scenarios. The system prevents website crashes and ensures fair allocation through a virtual queue system.

**Target users:**
- **Organizers:** Small-to-medium event promoters in Nigeria (concerts, conferences, product drops)
- **Customers:** Concert-goers, product buyers who are tired of crashed websites and unfair bot-dominated sales

**Core value proposition:**
- Organizers: "Never crash during a sale again. Sell out fairly and transparently."
- Customers: "Know your place in line. Buy tickets without the chaos."

---

## Design Principles

1. **Trust & Transparency:** Users must see exactly where they are in the queue and feel the system is fair
2. **Calm Under Pressure:** Interface stays calm even when 10,000 people are waiting (no panic-inducing elements)
3. **Mobile-First:** 80% of Nigerian users will access via mobile phones
4. **Low-Bandwidth Friendly:** Minimal images, fast load times, works on 3G
5. **Clear Status Communication:** At any moment, user knows: "What's happening? What do I do next?"

---

## User Flows to Design

### Flow 1: Customer - Joining Queue & Purchasing

**Pages needed:**

#### 1. Event Landing Page
- Hero image of event (Burna Boy concert photo)
- Event details: Date, venue, ticket price
- Ticket availability status:
  - "On Sale Now - Join Queue" (green button, prominent)
  - "Coming Soon - Sale starts June 1 at 10:00 AM" (countdown timer)
  - "Sold Out" (greyed out)
- Brief explanation: "How the queue works" (expandable section)

**Design notes:**
- CTA button should be unmissable (bright color, large, centered)
- Trust indicators: "Powered by FairQueue - Fair allocation guaranteed"
- Show social proof: "2,347 people in queue right now"

---

#### 2. Queue Waiting Screen
User has joined the queue and is waiting to be admitted.

**Key elements:**
- **Large queue position number:** "You are #1,547 in line"
- **Estimated wait time:** "Estimated wait: 12 minutes"
- **Progress indicator:** Visual representation (progress bar showing position relative to front of queue)
- **Live updates:** "Queue moving... You're now #1,423" (updates every 5 seconds)
- **What's happening:** Small text explaining "We're admitting users at a controlled rate to prevent the system from crashing"
- **Stay on this page warning:** "Keep this page open. If you leave, you'll lose your spot."
- **Option to get SMS when admitted:** "Enter your phone number to get notified when it's your turn"

**Design notes:**
- Calming color palette (blues, not aggressive reds)
- Subtle animation on the progress bar (shows system is working)
- NO countdown timers that create panic ("Only 3 seconds left!")
- Clear "What happens next" section

**States to design:**
- Waiting (position > 100)
- Almost there (position < 20) - slightly more excited UI
- Admitted - "It's your turn! You have 5 minutes to complete your purchase"

---

#### 3. Inventory Selection Screen
User has been admitted from the queue.

**Key elements:**
- **Timer at top:** "You have 4:32 remaining to complete purchase" (countdown, but not panic-inducing)
- **Available tickets grid:** Show available ticket types with prices
  - VIP Section - ₦50,000 - "23 left"
  - Regular - ₦25,000 - "347 left"
  - Early Bird (if applicable) - SOLD OUT (greyed)
- **Select quantity:** Dropdown (max 4 tickets per person)
- **Total price calculator:** Updates as user selects
- **"Proceed to Payment" button:** Clear, prominent
- **What if I don't finish:** "If time runs out, your spot goes to the next person. No items are reserved until payment completes."

**Design notes:**
- Clear price formatting (use ₦ symbol, thousands separator: ₦50,000 not 50000)
- Disable SOLD OUT options (don't make them clickable)
- Show scarcity without panic ("23 left" is informational, not "ONLY 23 LEFT!!!")

---

#### 4. Payment Screen (Paystack Integration)
User clicks "Proceed to Payment" and is redirected to Paystack.

**Key elements (on YOUR site before redirect):**
- **Summary of purchase:**
  - 2x VIP Tickets - ₦100,000
  - Service fee (3%) - ₦3,000
  - **Total: ₦103,000**
- **"Pay with Paystack" button:** Shows Paystack logo (trust indicator)
- **Security badges:** "Secure payment via Paystack" with lock icon
- **What happens next:** "You'll be redirected to Paystack to complete payment. After payment, you'll receive confirmation via email."

**Design notes:**
- Clear breakdown of costs (no hidden fees surprise)
- Use Paystack branding colors when showing their button (brand consistency)
- Reassure user about security

---

#### 5. Confirmation Screen (After Successful Payment)
User completes payment and is redirected back to your site.

**Key elements:**
- **Success message:** "🎉 Payment Successful!" (use emoji or icon)
- **Purchase summary:**
  - Order ID: #FQ-12345
  - 2x VIP Tickets to Burna Boy Live in Lagos
  - Total paid: ₦103,000
  - Date: June 1, 2024
- **What's next:**
  - "Confirmation email sent to user@email.com"
  - "Your tickets are attached as PDF"
  - "Show QR code at venue entrance"
- **Download buttons:**
  - "Download Tickets (PDF)"
  - "Add to Apple Wallet" / "Add to Google Pay" (if applicable)
- **Share on social:** "I got Burna Boy tickets! 🔥" (pre-filled tweet/post)

**Design notes:**
- Celebration moment (confetti animation on page load?)
- Clear next steps (don't leave user wondering "now what?")
- Make tickets easily accessible (big download button)

---

#### 6. Error/Failure States

**Queue Expired:**
- "Your queue session expired" (sad emoji or icon)
- "You waited too long to join. The event has sold out."
- "Sign up for notifications about similar events"

**Payment Failed:**
- "Payment could not be processed"
- Reason: "Card declined / Insufficient funds / Network error"
- "Your spot is still reserved for 3 more minutes. Try again?"
- Button: "Retry Payment"
- Alternative: "Change Payment Method"

**Sold Out While You Were Paying:**
- "Sorry, tickets sold out while you were checking out"
- "You have been automatically refunded"
- "Join the waitlist in case tickets become available"

**Design notes:**
- Empathetic tone (not robotic "ERROR: Transaction failed")
- Always offer a next step (never dead-end)
- If refund is happening, clearly state it

---

### Flow 2: Organizer - Creating Event & Monitoring Sales

**Pages needed:**

#### 1. Organizer Dashboard (Home)
Landing page after login.

**Key metrics (cards):**
- **Active Events:** 3
- **Total Sales This Month:** ₦2,450,000
- **Tickets Sold:** 1,234
- **Upcoming Sales:** 2 events starting soon

**Recent activity feed:**
- "Burna Boy Concert: 234 tickets sold in last hour"
- "Tech Summit: Payment issue reported by user@email.com"

**Quick actions:**
- "Create New Event" (prominent button)
- "View All Events"
- "Download Sales Report"

---

#### 2. Create Event Form
Organizer fills out event details.

**Form sections:**

**Basic Info:**
- Event Name (text input)
- Event Description (textarea)
- Event Image (file upload with preview)
- Event Date & Time (date/time picker)
- Venue (text input with Google Maps integration?)

**Ticketing:**
- Ticket Types: (can add multiple)
  - Name (e.g., "VIP", "Regular", "Early Bird")
  - Price (₦)
  - Quantity available
  - Description (optional)
  
**Sale Settings:**
- Sale Start Date/Time (when queue opens)
- Sale End Date/Time (or "Until sold out")
- Max tickets per person (default: 4)

**Queue Configuration:**
- Admission rate: (slider) "Admit 50-500 users per minute"
  - Help text: "Higher rate = faster sales but more server load"
- Allocation strategy: (dropdown)
  - First-come, first-served (default)
  - Random lottery (advanced - Phase 2)

**Payment:**
- Paystack account connected (if not, show "Connect Paystack" button)
- Service fee: "3% of ticket price + ₦100 per transaction"
- Estimated earnings calculator: 
  - "If you sell 1,000 tickets at ₦25,000 each"
  - "Gross: ₦25,000,000"
  - "Our fee (3%): ₦750,000"
  - "You receive: ₦24,250,000"

**Preview & Publish:**
- Preview button: Shows what event page looks like
- "Publish Event" button (disabled until all required fields filled)

**Design notes:**
- Use a wizard/stepper UI (Step 1 of 4: Basic Info)
- Inline validation (show errors as user types)
- Autosave drafts (don't lose progress)
- Help tooltips next to complex settings

---

#### 3. Live Event Dashboard
Organizer monitors sales in real-time during the event.

**Key metrics (large numbers, update in real-time):**
- **Tickets Sold:** 1,234 / 5,000 (24.7%)
- **Revenue:** ₦30,850,000
- **People in Queue:** 2,347
- **Admission Rate:** 100 users/min (with slider to adjust live)

**Charts:**
- Sales over time (line chart showing tickets sold per minute)
- Ticket type breakdown (pie chart: VIP 40%, Regular 60%)

**Live Activity Feed (scrolling list):**
- "10:15:32 - User purchased 2x VIP tickets - ₦100,000"
- "10:15:28 - 50 users admitted from queue"
- "10:15:15 - Payment failed for Order #12345 (card declined)"

**Queue Management:**
- Current queue size: 2,347
- Average wait time: 18 minutes
- Button: "Pause Admissions" (emergency brake if system overloaded)
- Button: "Increase Admission Rate" (if things are going smoothly)

**Alerts Section:**
- "⚠️ Payment failure rate higher than usual (8% vs typical 3%)"
- "✅ System healthy - no issues"

**Quick Actions:**
- "Download Current Sales Report (CSV)"
- "Send Update to All Waiting Users" (broadcast message)
- "End Sale Early" (if sold out or emergency)

**Design notes:**
- Use WebSocket for live updates (no page refresh)
- Green/yellow/red color coding for system health
- Big fonts for key numbers (should be readable from across the room)
- Mobile-responsive (organizer might be at venue checking on phone)

---

#### 4. Event Analytics (Post-Event)
After event ends, detailed breakdown.

**Summary Cards:**
- Total Revenue
- Total Tickets Sold
- Average Order Value
- Peak Concurrent Users

**Charts & Graphs:**
- Sales timeline (when did tickets sell?)
- Traffic sources (where did buyers come from?)
- Conversion funnel:
  - Visited event page: 10,000
  - Joined queue: 8,000
  - Admitted: 6,000
  - Started checkout: 5,500
  - Completed purchase: 5,000
  - Drop-off at each stage highlighted

**Customer Insights:**
- Most popular ticket type
- Average time in queue
- Payment method breakdown (card vs bank transfer)

**Export Options:**
- "Download Full Report (PDF)"
- "Export Customer List (CSV)" (for follow-up emails)
- "Generate Invoice" (for accounting)

---

## Additional Screens

### Mobile App Considerations
Since 80% of users are on mobile:

**Queue Waiting Screen (Mobile):**
- Larger position number (fills most of screen)
- Push notifications when position changes significantly
- Option to background the tab (send SMS when admitted)

**Payment Screen (Mobile):**
- Paystack's mobile SDK integration (native feel)
- Support for USSD payment (for users without cards)

---

## Design System Specifications

### Colors

**Primary Brand Colors:**
- Primary: `#2563EB` (Blue - trust, calm)
- Secondary: `#10B981` (Green - success, go)
- Accent: `#F59E0B` (Amber - attention, warning)

**UI Colors:**
- Background: `#F9FAFB` (Light grey)
- Card/Surface: `#FFFFFF` (White)
- Text Primary: `#111827` (Near black)
- Text Secondary: `#6B7280` (Grey)
- Border: `#E5E7EB` (Light border)

**Status Colors:**
- Success: `#10B981` (Green)
- Warning: `#F59E0B` (Amber)
- Error: `#EF4444` (Red)
- Info: `#3B82F6` (Blue)

---

### Typography

**Font Family:**
- Primary: `Inter` (clean, modern, great readability)
- Fallback: `system-ui, -apple-system, sans-serif`

**Font Sizes:**
- Hero/Display: `48px` (event titles)
- H1: `36px` (page titles)
- H2: `24px` (section headers)
- H3: `20px` (card titles)
- Body: `16px` (main text)
- Small: `14px` (meta info, help text)
- Tiny: `12px` (timestamps, footnotes)

**Font Weights:**
- Bold: `700` (CTAs, important numbers)
- Semibold: `600` (headings)
- Regular: `400` (body text)

---

### Spacing

Use 8px grid system:
- XXS: `4px`
- XS: `8px`
- S: `16px`
- M: `24px`
- L: `32px`
- XL: `48px`
- XXL: `64px`

---

### Components

**Buttons:**
- Primary: Blue background, white text, rounded corners (8px)
- Secondary: White background, blue border, blue text
- Danger: Red background (for destructive actions)
- Sizes: Small (32px height), Medium (40px), Large (48px)

**Cards:**
- White background
- Border: 1px solid `#E5E7EB`
- Border radius: `12px`
- Shadow: `0 1px 3px rgba(0, 0, 0, 0.1)`
- Padding: `24px`

**Input Fields:**
- Height: `40px`
- Border: 1px solid `#D1D5DB`
- Border radius: `8px`
- Focus state: Blue border `#2563EB`, subtle shadow
- Error state: Red border, error message below

**Progress Bars:**
- Height: `8px`
- Background: `#E5E7EB` (grey)
- Fill: `#2563EB` (blue) with smooth animation
- Border radius: `4px`

**Badges/Tags:**
- Small pill-shaped elements
- Border radius: `12px`
- Padding: `4px 12px`
- Font size: `12px`
- Colors match status (green for active, grey for inactive)

---

### Icons

**Icon Library:** Use Lucide Icons or Heroicons (consistent, modern, free)

**Common icons needed:**
- Queue: `users` icon
- Timer: `clock` icon
- Success: `check-circle` icon
- Error: `x-circle` icon
- Info: `info` icon
- Download: `download` icon
- Edit: `pencil` icon
- Delete: `trash` icon
- Settings: `settings` icon

---

## Responsive Breakpoints

- Mobile: `< 640px`
- Tablet: `640px - 1024px`
- Desktop: `> 1024px`

**Mobile-first approach:**
- Design for mobile first
- Enhance for larger screens
- No horizontal scrolling
- Touch-friendly tap targets (minimum 44px)

---

## Accessibility Requirements

- **Color contrast:** WCAG AA compliant (4.5:1 for text)
- **Focus states:** Visible keyboard focus indicators
- **Screen reader support:** Proper ARIA labels
- **Alt text:** All images have descriptive alt text
- **Form labels:** Every input has a visible label
- **Error messages:** Clear, specific, actionable

---

## Animation & Micro-interactions

**Use sparingly, purposefully:**

**Queue position updates:**
- Number counts up/down smoothly (not instant jump)
- Subtle pulse when big change happens (moved 100+ spots)

**Success states:**
- Confetti animation on purchase confirmation (1-2 seconds, then fades)
- Checkmark animation (draw the check, don't just appear)

**Loading states:**
- Skeleton screens (greyed out placeholders while content loads)
- Spinner for quick actions (< 2 seconds)
- Progress bar for longer operations (payment processing)

**Transitions:**
- Page transitions: `300ms` ease-in-out
- Hover effects: `150ms` ease
- No jarring pop-ins

---

## Mobile-Specific Patterns

**Bottom Sheet Navigation (Mobile):**
- Key actions slide up from bottom
- Large touch targets
- Swipe to dismiss

**Sticky CTA Buttons (Mobile):**
- "Join Queue" button sticks to bottom of screen
- Always visible as user scrolls
- Thumb-friendly position

**Pull-to-Refresh:**
- On queue waiting screen, pull down to manually refresh position
- Visual feedback (loading indicator)

---

## Sample Screens Priority Order

If designer can only do a few screens, prioritize:

1. **Event Landing Page** (customer entry point)
2. **Queue Waiting Screen** (core experience)
3. **Inventory Selection Screen** (purchase flow)
4. **Live Event Dashboard** (organizer monitors sales)
5. **Create Event Form** (organizer onboarding)

---

## Design Deliverables Requested

Please provide:

1. **High-fidelity mockups** (all screens listed above)
2. **Mobile versions** of customer-facing screens
3. **Component library** (buttons, inputs, cards, etc.)
4. **Color palette & typography guide**
5. **Prototype** (clickable flow from event page → queue → purchase)
6. **Design system documentation** (Figma file organized with components)

**File format:** Figma (shareable link with edit access)

---

## Inspiration & References

**Look at these for queue UX:**
- Ticketmaster queue system (but make it calmer, less panic)
- Apple product launches (clean, premium feel)
- Zoom waiting room (clear status communication)

**Look at these for dashboard:**
- Stripe Dashboard (clean metrics, great charts)
- Shopify Analytics (clear data visualization)
- Plausible Analytics (simple, focused)

**Nigerian design context:**
- Bold colors are okay (not overly minimal)
- Show social proof (Nigerians trust what others trust)
- Pricing transparency is critical (no hidden fees)

---

## Questions for Designer to Consider

1. How do we communicate "fairness" visually? (Queue is random, not first-come)
2. How do we keep users engaged during 15-minute wait? (Games? Content?)
3. How do we handle slow internet? (Offline states? Bandwidth indicators?)
4. How do we build trust in a new platform? (Badges? Testimonials? Security indicators?)

---

## Final Note for Designer

**This is a trust-critical product.** Users are entrusting you with:
- Their money (payments)
- Their time (waiting in queue)
- Their fairness expectations (no bots, no crashes)

Every design decision should ask: **"Does this build trust?"**

- Clear communication > Clever animations
- Transparency > Hiding complexity
- Calm, confident tone > Urgency and panic

The UI should feel like a reliable friend guiding you through a stressful process, not adding to the stress.

---

**Ready to design? Let's build something fair. 🎯**