import { createBrowserRouter } from "react-router";
import { Layout } from "./components/Layout";
import { Home } from "./pages/Home";
import { Queue } from "./pages/Queue";
import { Selection } from "./pages/Selection";
import { Payment } from "./pages/Payment";
import { Confirmation } from "./pages/Confirmation";
import { Dashboard } from "./pages/organizer/Dashboard";
import { CreateEvent } from "./pages/organizer/CreateEvent";
import { LiveDashboard } from "./pages/organizer/LiveDashboard";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: Layout,
    children: [
      {
        index: true,
        Component: Home,
      },
      {
        path: "queue",
        Component: Queue,
      },
      {
        path: "select-tickets",
        Component: Selection,
      },
      {
        path: "payment",
        Component: Payment,
      },
      {
        path: "confirmation",
        Component: Confirmation,
      },
      {
        path: "organizer",
        children: [
          {
            index: true,
            Component: Dashboard,
          },
          {
            path: "create",
            Component: CreateEvent,
          },
          {
            path: "live/:id",
            Component: LiveDashboard,
          },
        ],
      },
    ],
  },
]);
