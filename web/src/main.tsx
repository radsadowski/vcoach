import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import "./index.css";
import App from "./App";
import { DashboardView } from "./features/dashboard/DashboardView";
import { ChatView } from "./features/chat/ChatView";
import { PlanView } from "./features/plan/PlanView";
import { GoalsView } from "./features/goals/GoalsView";
import { ProfileView } from "./features/profile/ProfileView";
import { ClimbingLogView } from "./features/climbing/ClimbingLogView";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000 },
  },
});

const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    children: [
      { index: true, element: <DashboardView /> },
      { path: "chat", element: <ChatView /> },
      { path: "chat/:sessionId", element: <ChatView /> },
      { path: "plan", element: <PlanView /> },
      { path: "goals", element: <GoalsView /> },
      { path: "climbing", element: <ClimbingLogView /> },
      { path: "profile", element: <ProfileView /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
);
