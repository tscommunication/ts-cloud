import { Navigate, Outlet, useLocation } from "react-router-dom";

import { getStoredUser } from "../api/auth";

export default function CustomerRoute() {
  const location = useLocation();
  const token = localStorage.getItem("access_token");
  const user = getStoredUser();

  if (!token) {
    return (
      <Navigate
        to="/selfcare/login"
        replace
        state={{ from: location.pathname }}
      />
    );
  }

  if (user?.role !== "customer" || !user.customer_id) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
