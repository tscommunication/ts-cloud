import { createBrowserRouter, Navigate } from "react-router-dom";

import App from "../App";
import AdminLayout from "../layouts/AdminLayout";
import ProtectedRoute from "./ProtectedRoute";
import CustomerRoute from "./CustomerRoute";
import RoleRoute from "./RoleRoute";
import RouteError from "./RouteError";

import {
  Customers,
  CustomerProvisionRequests,
  Dashboard,
  FTP,
  Invoices,
  LazyRoute,
  Login,
  SelfCareLogin,
  SelfCareHome,
  Packages,
  Payments,
  Organization,
  AgentPackagePermissions,
  CodeManagement,
  AgentCollections,
  NetworkRouters,
  NetworkDevices,
  PPPoESessions,
  CustomerImport,
  ServiceEntitlements,
  Settings,
  Subscriptions,
  Users,
} from "./LazyRoutes";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    errorElement: <RouteError />,
    children: [
      {
        path: "login",
        element: <LazyRoute element={<Login />} />,
      },
      {
        path: "selfcare/login",
        element: <LazyRoute element={<SelfCareLogin />} />,
      },
      {
        element: <CustomerRoute />,
        children: [
          {
            path: "selfcare",
            element: <LazyRoute element={<SelfCareHome />} />,
          },
        ],
      },
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <AdminLayout />,
            children: [
              {
                index: true,
                element: <Navigate to="/dashboard" replace />,
              },
              {
                path: "settings",
                element: <LazyRoute element={<Settings />} />,
              },
              {
                element: <RoleRoute roles={["superadmin", "admin", "agent"]} />,
                children: [
                  {
                    path: "dashboard",
                    element: <LazyRoute element={<Dashboard />} />,
                  },
                  {
                    path: "customer-provision-requests",
                    element: (
                      <LazyRoute element={<CustomerProvisionRequests />} />
                    ),
                  },
                  {
                    path: "customers",
                    element: <LazyRoute element={<Customers />} />,
                  },
                  {
                    path: "agent-collections",
                    element: <LazyRoute element={<AgentCollections />} />,
                  },
                  {
                    path: "payments",
                    element: <LazyRoute element={<Payments />} />,
                  },
                ],
              },
              {
                element: <RoleRoute roles={["superadmin", "admin"]} />,
                children: [
                  {
                    path: "organization",
                    element: <Navigate to="/organization/pops" replace />,
                  },
                  {
                    path: "organization/pops",
                    element: <LazyRoute element={<Organization />} />,
                  },
                  {
                    path: "organization/agents",
                    element: <LazyRoute element={<Organization />} />,
                  },
                  {
                    path: "organization/agent-package-permissions",
                    element: <LazyRoute element={<AgentPackagePermissions />} />,
                  },
                  {
                    path: "network/devices",
                    element: <LazyRoute element={<NetworkDevices />} />,
                  },
                  {
                    path: "network/routers",
                    element: <LazyRoute element={<NetworkRouters />} />,
                  },
                  {
                    path: "network/pppoe-sessions",
                    element: <LazyRoute element={<PPPoESessions />} />,
                  },
                  {
                    path: "customers/import",
                    element: <LazyRoute element={<CustomerImport />} />,
                  },
                  {
                    path: "packages",
                    element: <LazyRoute element={<Packages />} />,
                  },
                  {
                    path: "subscriptions",
                    element: <LazyRoute element={<Subscriptions />} />,
                  },
                  {
                    path: "invoices",
                    element: <LazyRoute element={<Invoices />} />,
                  },
                  {
                    path: "ftp",
                    element: <LazyRoute element={<FTP />} />,
                  },
                  {
                    path: "service-entitlements",
                    element: <LazyRoute element={<ServiceEntitlements />} />,
                  },
                  {
                    path: "users",
                    element: <LazyRoute element={<Users />} />,
                  },
                ],
              },
              {
                element: <RoleRoute roles={["superadmin"]} />,
                children: [
                  {
                    path: "organization/code-management",
                    element: <LazyRoute element={<CodeManagement />} />,
                  },
                ],
              },
            ],
          },
        ],
      },
      {
        path: "*",
        loader: () => {
          throw new Response("Not Found", { status: 404 });
        },
      },
    ],
  },
]);
