import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import DeviceHubIcon from "@mui/icons-material/DeviceHub";
import FiberManualRecordIcon from "@mui/icons-material/FiberManualRecord";
import RouterIcon from "@mui/icons-material/Router";
import SensorsOffIcon from "@mui/icons-material/SensorsOff";
import WarningAmberIcon from "@mui/icons-material/WarningAmber";

import { getOLTDashboard } from "../../api/oltDashboard";
import { getAPIErrorMessage } from "../../api/errors";

const asNumber = (value: number | undefined | null) => value ?? 0;

const statusChip = (status: string) => {
  const online = status.trim().toUpperCase() === "ONLINE";
  return <Chip size="small" color={online ? "success" : "default"} label={online ? "ONLINE" : status || "OFFLINE"} />;
};

const formatDate = (value?: string | null) => value ? new Date(value).toLocaleString() : "Never";

export default function OLTDashboard() {
  const navigate = useNavigate();
  const dashboard = useQuery({
    queryKey: ["network-olt-dashboard"],
    queryFn: getOLTDashboard,
    refetchInterval: 30000,
  });

  if (dashboard.isLoading) {
    return <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}><CircularProgress aria-label="Loading OLT dashboard" /></Box>;
  }

  if (dashboard.isError) {
    return <Alert severity="error">{getAPIErrorMessage(dashboard.error, "Failed to load OLT dashboard.")}</Alert>;
  }

  const data = dashboard.data;
  if (!data) return null;

  const summaryCards = [
    { label: "Total OLTs", value: data.summary.total_olts, icon: <RouterIcon />, color: "primary.main" },
    { label: "OLTs Online", value: data.summary.online_olts, icon: <FiberManualRecordIcon />, color: "success.main" },
    { label: "OLTs Offline", value: data.summary.offline_olts, icon: <SensorsOffIcon />, color: "error.main" },
    { label: "Total ONUs", value: data.summary.total_onus, icon: <DeviceHubIcon />, color: "info.main" },
    { label: "ONUs Online", value: data.summary.online_onus, icon: <FiberManualRecordIcon />, color: "success.main" },
    { label: "ONUs Offline", value: data.summary.offline_onus, icon: <SensorsOffIcon />, color: "error.main" },
    { label: "Optical Reading Missing", value: data.summary.optical_missing, icon: <WarningAmberIcon />, color: "warning.main" },
  ];

  return <Box>
    <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: { sm: "center" }, flexDirection: { xs: "column", sm: "row" }, gap: 2, mb: 3 }}>
      <Box>
        <Typography variant="h4" sx={{ fontWeight: 700 }}>OLT Dashboard</Typography>
        <Typography color="text.secondary">OLT health and ONU status for your authorized network scope. Refreshes every 30 seconds.</Typography>
      </Box>
      <Button variant="outlined" onClick={() => navigate("/network/devices?type=OLT")}>Open OLT Monitoring</Button>
    </Box>

    <Grid container spacing={2} sx={{ mb: 3 }}>
      {summaryCards.map((card) => <Grid key={card.label} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
        <Card sx={{ height: "100%" }}><CardContent><Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
          <Box><Typography color="text.secondary">{card.label}</Typography><Typography variant="h5" sx={{ fontWeight: 700 }}>{asNumber(card.value).toLocaleString()}</Typography></Box>
          <Box sx={{ color: card.color }}>{card.icon}</Box>
        </Box></CardContent></Card>
      </Grid>)}
    </Grid>

    <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>OLT Health</Typography>
    <Card sx={{ mb: 3 }}><CardContent><TableContainer><Table size="small" sx={{ minWidth: 1150 }}><TableHead><TableRow>
      <TableCell>OLT</TableCell><TableCell>Vendor / Type</TableCell><TableCell>POP</TableCell><TableCell>Status</TableCell><TableCell align="right">ONUs</TableCell><TableCell align="right">Online</TableCell><TableCell align="right">Offline</TableCell><TableCell align="right">Optical Missing</TableCell><TableCell>Last Polled</TableCell><TableCell>Action</TableCell>
    </TableRow></TableHead><TableBody>
      {data.olts.map((olt) => <TableRow key={olt.id} hover>
        <TableCell><Typography sx={{ fontWeight: 700 }}>{olt.code}</Typography><Typography variant="body2" color="text.secondary">{olt.name}</Typography></TableCell>
        <TableCell>{olt.vendor || "—"}<br /><Typography variant="caption" color="text.secondary">{olt.olt_type || "—"}</Typography></TableCell>
        <TableCell>{olt.pop_name || "Unassigned"}</TableCell><TableCell>{statusChip(olt.monitoring_status)}</TableCell>
        <TableCell align="right">{asNumber(olt.total_onus).toLocaleString()}</TableCell><TableCell align="right">{asNumber(olt.online_onus).toLocaleString()}</TableCell><TableCell align="right">{asNumber(olt.offline_onus).toLocaleString()}</TableCell>
        <TableCell align="right">{asNumber(olt.optical_missing).toLocaleString()}</TableCell><TableCell>{formatDate(olt.last_polled_at)}{olt.last_error && <Typography variant="caption" color="error" sx={{ display: "block", maxWidth: 200 }}>{olt.last_error}</Typography>}</TableCell>
        <TableCell><Button size="small" onClick={() => navigate("/network/devices?type=OLT")}>View device</Button></TableCell>
      </TableRow>)}
      {data.olts.length === 0 && <TableRow><TableCell colSpan={10} align="center">No OLTs are available in your network scope.</TableCell></TableRow>}
    </TableBody></Table></TableContainer></CardContent></Card>

    <Grid container spacing={3}>
      <Grid size={{ xs: 12, lg: 6 }}><Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>Vendor Summary</Typography><Card><CardContent><TableContainer><Table size="small"><TableHead><TableRow><TableCell>Vendor</TableCell><TableCell align="right">OLTs</TableCell><TableCell align="right">Online</TableCell><TableCell align="right">ONUs</TableCell><TableCell align="right">ONU Online</TableCell><TableCell align="right">ONU Offline</TableCell></TableRow></TableHead><TableBody>
        {data.vendors.map((vendor) => <TableRow key={vendor.vendor}><TableCell>{vendor.vendor || "Unspecified"}</TableCell><TableCell align="right">{asNumber(vendor.olt_count)}</TableCell><TableCell align="right">{asNumber(vendor.online_olts)}</TableCell><TableCell align="right">{asNumber(vendor.total_onus)}</TableCell><TableCell align="right">{asNumber(vendor.online_onus)}</TableCell><TableCell align="right">{asNumber(vendor.offline_onus)}</TableCell></TableRow>)}
        {data.vendors.length === 0 && <TableRow><TableCell colSpan={6} align="center">No vendor data available.</TableCell></TableRow>}
      </TableBody></Table></TableContainer></CardContent></Card></Grid>
      <Grid size={{ xs: 12, lg: 6 }}><Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>POP Summary</Typography><Card><CardContent><TableContainer><Table size="small"><TableHead><TableRow><TableCell>POP</TableCell><TableCell align="right">OLTs</TableCell><TableCell align="right">Online</TableCell><TableCell align="right">ONUs</TableCell><TableCell align="right">ONU Online</TableCell><TableCell align="right">ONU Offline</TableCell></TableRow></TableHead><TableBody>
        {data.pops.map((pop) => <TableRow key={pop.pop_id ?? "unassigned"}><TableCell>{pop.pop_name || "Unassigned"}</TableCell><TableCell align="right">{asNumber(pop.olt_count)}</TableCell><TableCell align="right">{asNumber(pop.online_olts)}</TableCell><TableCell align="right">{asNumber(pop.total_onus)}</TableCell><TableCell align="right">{asNumber(pop.online_onus)}</TableCell><TableCell align="right">{asNumber(pop.offline_onus)}</TableCell></TableRow>)}
        {data.pops.length === 0 && <TableRow><TableCell colSpan={6} align="center">No POP data available.</TableCell></TableRow>}
      </TableBody></Table></TableContainer></CardContent></Card></Grid>
    </Grid>
  </Box>;
}
