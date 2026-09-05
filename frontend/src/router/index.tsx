import { createBrowserRouter, Navigate } from "react-router-dom";

import App from "../App";
import AdminLayout from "../layouts/AdminLayout";
import ProtectedRoute from "./ProtectedRoute";
import CustomerRoute from "./CustomerRoute";
import SelfCareLayout from "../layouts/SelfCareLayout";
import RoleRoute from "./RoleRoute";
import RouteError from "./RouteError";

import {
  Customers,
  CustomerDetails,
  CustomerProvisionRequests,
  CustomerChangeRequests,
  Dashboard,
  FTP,
  Invoices,
  LazyRoute,
  Login,
  SelfCareLogin,
  SelfCareHome,
  SelfCarePortalPage,
  Packages,
  Payments,
  Organization,
  AgentPackagePermissions,
  CodeManagement,
  AgentCollections,
  NetworkRouters,
  NetworkDevices,
  OLTDashboard,
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
            element: <SelfCareLayout />,
            children: [
              { path: "selfcare", element: <LazyRoute element={<SelfCareHome />} /> },
              { path: "selfcare/live-traffic", element: <LazyRoute element={<SelfCarePortalPage page="traffic" />} /> },
              { path: "selfcare/connection", element: <LazyRoute element={<SelfCarePortalPage page="connection" />} /> },
              { path: "selfcare/packages", element: <LazyRoute element={<SelfCarePortalPage page="packages" />} /> },
              { path: "selfcare/billing", element: <LazyRoute element={<SelfCarePortalPage page="billing" />} /> },
              { path: "selfcare/payments", element: <LazyRoute element={<SelfCarePortalPage page="payments" />} /> },
              { path: "selfcare/profile", element: <LazyRoute element={<SelfCarePortalPage page="profile" />} /> },
              { path: "selfcare/services", element: <LazyRoute element={<SelfCarePortalPage page="services" />} /> },
            ],
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
                element: <RoleRoute roles={["superadmin", "admin", "noc", "agent"]} />,
                children: [
                  {
                    path: "dashboard",
                    element: <LazyRoute element={<Dashboard />} />,
                  },
                  {
                    path: "network/devices",
                    element: <LazyRoute element={<NetworkDevices />} />,
                  },
                  {
                    path: "network/olt-dashboard",
                    element: <LazyRoute element={<OLTDashboard />} />,
                  },
                ],
              },
              {
                element: <RoleRoute roles={["superadmin", "admin", "agent"]} />,
                children: [
                  {
                    path: "customer-provision-requests",
                    element: (
                      <LazyRoute element={<CustomerProvisionRequests />} />
                    ),
                  },
                  { path: "customer-change-requests", element: <LazyRoute element={<CustomerChangeRequests />} /> },
                  {
                    path: "customers",
                    element: <LazyRoute element={<Customers />} />,
                  },
                  {
                    path: "customers/:id",
                    element: <LazyRoute element={<CustomerDetails />} />,
                  },
                  {
                    path: "agent-collections",
                    element: <LazyRoute element={<AgentCollections />} />,
                  },
                  {
                    path: "payments",
                    element: <LazyRoute element={<Payments />} />,
                  },
                  {
                    path: "network/pppoe-sessions",
                    element: <LazyRoute element={<PPPoESessions />} />,
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
                    path: "network/routers",
                    element: <LazyRoute element={<NetworkRouters />} />,
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
