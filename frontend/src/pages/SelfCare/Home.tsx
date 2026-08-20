import {
  Box,
  Button,
  Card,
  CardContent,
  Container,
  Typography,
} from "@mui/material";
import { useNavigate } from "react-router-dom";

import { getStoredUser, logout } from "../../api/auth";

export default function SelfCareHome() {
  const navigate = useNavigate();
  const user = getStoredUser();

  function handleLogout() {
    logout();
    navigate("/selfcare/login", { replace: true });
  }

  return (
    <Box
      sx={{
        minHeight: "100vh",
        backgroundColor: "#f5f7fb",
        py: 4,
      }}
    >
      <Container maxWidth="md">
        <Card sx={{ borderRadius: 3 }}>
          <CardContent sx={{ p: 4 }}>
            <Typography variant="h4" sx={{ fontWeight: 700 }} gutterBottom>
              TS-Cloud SelfCare
            </Typography>

            <Typography color="text.secondary">
              Customer portal is connected successfully.
            </Typography>

            <Typography sx={{ mt: 3 }}>
              Username: {user?.username || "—"}
            </Typography>

            <Typography>Customer ID: {user?.customer_id || "—"}</Typography>

            <Button variant="outlined" sx={{ mt: 3 }} onClick={handleLogout}>
              Sign Out
            </Button>
          </CardContent>
        </Card>
      </Container>
    </Box>
  );
}
