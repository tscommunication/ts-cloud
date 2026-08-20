import { Navigate, Outlet, useLocation } from "react-router-dom";
import { getStoredUser } from "../api/auth";

export default function ProtectedRoute() {
  const location = useLocation();
  const token = localStorage.getItem("access_token");
  const user = getStoredUser();

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  if (user?.role === "customer") {
    return <Navigate to="/selfcare" replace />;
  }

  return <Outlet />;
}
