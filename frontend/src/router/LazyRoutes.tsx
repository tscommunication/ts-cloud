import { lazy, Suspense, type ReactNode } from "react";
import { Box, CircularProgress } from "@mui/material";

export const Login = lazy(() => import("../pages/Auth/Login"));
export const SelfCareLogin = lazy(() => import("../pages/SelfCare/Login"));
export const SelfCareHome = lazy(() => import("../pages/SelfCare/Home"));
export const SelfCarePortalPage = lazy(() => import("../pages/SelfCare/PortalPage"));
export const Customers = lazy(() => import("../pages/Customers"));
export const CustomerDetails = lazy(() => import("../pages/CustomerDetails"));
export const CustomerProvisionRequests = lazy(
  () => import("../pages/CustomerProvisionRequests"),
);
export const Dashboard = lazy(() => import("../pages/Dashboard"));
export const FTP = lazy(() => import("../pages/FTP"));
export const Invoices = lazy(() => import("../pages/Invoices"));
export const Packages = lazy(() => import("../pages/Packages"));
export const Payments = lazy(() => import("../pages/Payments"));
export const Settings = lazy(() => import("../pages/Settings"));
export const Subscriptions = lazy(() => import("../pages/Subscriptions"));
export const Users = lazy(() => import("../pages/Users"));
export const Organization = lazy(() => import("../pages/Organization"));
export const AgentPackagePermissions = lazy(() => import("../pages/AgentPackagePermissions"));
export const CodeManagement = lazy(() => import("../pages/CodeManagement"));
export const AgentCollections = lazy(() => import("../pages/AgentCollections"));
export const NetworkRouters = lazy(() => import("../pages/NetworkRouters"));
export const NetworkDevices = lazy(() => import("../pages/NetworkDevices"));
export const PPPoESessions = lazy(() => import("../pages/PPPoESessions"));
export const CustomerImport = lazy(() => import("../pages/CustomerImport"));
export const CustomerChangeRequests = lazy(() => import("../pages/CustomerChangeRequests"));
export const ServiceEntitlements = lazy(() => import("../pages/ServiceEntitlements"));

function RouteFallback() {
  return (
    <Box
      sx={{
        minHeight: "40vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <CircularProgress aria-label="Loading page" />
    </Box>
  );
}

export function LazyRoute({ element }: { element: ReactNode }) {
  return <Suspense fallback={<RouteFallback />}>{element}</Suspense>;
}
