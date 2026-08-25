import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Grid,
  IconButton,
  MenuItem,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  TextField,
  Typography,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import PlayCircleIcon from "@mui/icons-material/PlayCircle";
import VisibilityIcon from "@mui/icons-material/Visibility";
import {
  createNetworkDevice,
  deleteNetworkDevice,
  getNetworkDevices,
  getNetworkDeviceONUs,
  getNetworkDevicePorts,
  testNetworkDeviceConnection,
  updateNetworkDevice,
  type NetworkDevice,
  type NetworkDeviceInput,
  type NetworkDeviceONU,
  type NetworkDevicePort,
} from "../../api/networkDevices";
import { getPOPs, type POP } from "../../api/distribution";
import { getNetworkRouters, type NetworkRouter } from "../../api/networkRouters";
import { getAPIErrorMessage } from "../../api/errors";
import { getStoredUser } from "../../api/auth";

const catalogs: Record<string, string[]> = {
  MIKROTIK: ["CCR2116-12G-4S+", "Other / Custom Model"],
  FOCUSCOM: ["8-Port Switch", "Other / Custom Model"],
  ZIBBIX: ["4-Port 10G EPON OLT", "Other / Custom Model"],
  BDCOM: ["Other / Custom Model"],
  ECOM: ["G/EPON OLT", "Other / Custom Model"],
  VSOL: ["10G G/EPON OLT", "Other / Custom Model"],
  HSGQ: ["OLT", "Other / Custom Model"],
  "SOLITINE / TBS": ["OLT", "Other / Custom Model"],
  PHYHOME: ["OLT", "Other / Custom Model"],
  OTHER: ["Other / Custom Model"],
};
const blank: NetworkDeviceInput = {
  code: "",
  name: "",
  device_type: "OLT",
  vendor: "VSOL",
  model: "10G G/EPON OLT",
  olt_type: "EPON",
  management_ip: "",
  management_port: 9001,
  router_ids: [],
  monitoring_protocol: "SNMP",
  snmp_version: "V2C",
  snmp_port: 161,
  snmp_username: "",
  snmp_secret: "",
  polling_interval_seconds: 300,
  monitoring_enabled: true,
  remarks: "",
};

type ONUSortKey =
  | "onu"
  | "mac"
  | "description"
  | "status"
  | "rx_power"
  | "rx_mbps"
  | "tx_mbps"
  | "last_down";

export default function NetworkDevices() {
  const [rows, setRows] = useState<NetworkDevice[]>([]);
  const [selectedDeviceIDs, setSelectedDeviceIDs] =
    useState<Set<number>>(new Set());
  const [pops, setPops] = useState<POP[]>([]);
  const [routers, setRouters] = useState<NetworkRouter[]>([]);
  const [form, setForm] = useState<NetworkDeviceInput>(blank);
  const [editing, setEditing] = useState<NetworkDevice | null>(null);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [customModel, setCustomModel] = useState("");
  const [detailDevice, setDetailDevice] =
    useState<NetworkDevice | null>(null);
  const [detailPorts, setDetailPorts] =
    useState<NetworkDevicePort[]>([]);
  const [detailONUs, setDetailONUs] =
    useState<NetworkDeviceONU[]>([]);
  const [loadingPorts, setLoadingPorts] =
    useState(false);
  const [ponONUFilters, setPonONUFilters] =
    useState<Record<number, "ALL" | "ONLINE" | "OFFLINE">>({});
  const [onuSortBy, setONUSortBy] =
    useState<ONUSortKey>("onu");
  const [onuSortDirection, setONUSortDirection] =
    useState<"asc" | "desc">("asc");
  const [onuSearch, setONUSearch] = useState("");
  const [onuPONFilter, setONUPONFilter] =
    useState("ALL");
  const [onuStatusFilter, setONUStatusFilter] =
    useState<"ALL" | "ONLINE" | "OFFLINE">("ALL");
  const [onuOpticalFilter, setONUOpticalFilter] =
    useState<
      "ALL" | "NORMAL" | "WEAK" | "TOO_RISK" | "UNAVAILABLE"
    >("ALL");
  const [onuNumberFilter, setONUNumberFilter] =
    useState("");
  const [searchParams] = useSearchParams();
  const currentRole = getStoredUser()?.role;
  const isSuper = currentRole === "superadmin";
  const isAgent = currentRole === "agent";

  const requestedType = (
    searchParams.get("type") ?? ""
  ).trim().toUpperCase();

  const requestedStatus = (
    searchParams.get("status") ?? ""
  ).trim().toUpperCase();

  const models = useMemo(
    () => catalogs[form.vendor] ?? catalogs.OTHER,
    [form.vendor],
  );

  const safeRows = useMemo(() => {
    const source = Array.isArray(rows) ? rows : [];

    return source.filter((row) => {
      if (
        requestedType &&
        row.device_type !== requestedType
      ) {
        return false;
      }

      if (requestedStatus) {
        if (
          requestedStatus === "OFFLINE" &&
          !row.monitoring_enabled
        ) {
          return false;
        }

        if (
          row.monitoring_status !== requestedStatus
        ) {
          return false;
        }
      }

      return true;
    });
  }, [
    rows,
    requestedType,
    requestedStatus,
  ]);
  const selectedRows = rows.filter((row) =>
    selectedDeviceIDs.has(row.id),
  );

  const canEnableSelected = selectedRows.some(
    (row) => !row.monitoring_enabled,
  );

  const canDisableSelected = selectedRows.some(
    (row) => row.monitoring_enabled,
  );

  const allVisibleSelected =
    safeRows.length > 0 &&
    safeRows.every((row) => selectedDeviceIDs.has(row.id));

  const someVisibleSelected =
    safeRows.some((row) => selectedDeviceIDs.has(row.id)) &&
    !allVisibleSelected;

  const toggleVisibleDevices = (checked: boolean) => {
    setSelectedDeviceIDs((current) => {
      const next = new Set(current);

      for (const row of safeRows) {
        if (checked) {
          next.add(row.id);
        } else {
          next.delete(row.id);
        }
      }

      return next;
    });
  };

  const toggleDeviceSelection = (id: number) => {
    setSelectedDeviceIDs((current) => {
      const next = new Set(current);

      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }

      return next;
    });
  };

  const load = useCallback(async () => {
    try {
      setError("");

      if (isAgent) {
        const d = await getNetworkDevices();
        setRows(Array.isArray(d) ? d : []);
        setPops([]);
        setRouters([]);
        return;
      }

      const [d, p, r] = await Promise.all([
        getNetworkDevices(),
        getPOPs(),
        getNetworkRouters(),
      ]);
      setRows(Array.isArray(d) ? d : []);
      setPops(Array.isArray(p) ? p : []);
      setRouters(Array.isArray(r) ? r : []);
    } catch (e) {
      setError(getAPIErrorMessage(e, "Unable to load network devices."));
    }
  }, [isAgent]);
  const updateSelectedMonitoring = async (enabled: boolean) => {
    const targets = selectedRows.filter(
      (row) => row.monitoring_enabled !== enabled,
    );

    if (targets.length === 0) {
      return;
    }

    if (
      !enabled &&
      !window.confirm(
        `Disable monitoring for ${targets.length} selected device(s)?`,
      )
    ) {
      return;
    }

    const failedIDs = new Set<number>();
    const failedNames: string[] = [];

    try {
      setBusy(true);
      setError("");

      for (const row of targets) {
        const payload: NetworkDeviceInput = {
          code: row.code,
          name: row.name,
          device_type: row.device_type,
          vendor: row.vendor,
          model: row.model,
          olt_type: row.olt_type,
          pop_id: row.pop_id,
          management_ip: row.management_ip,
          management_port: row.management_port,
          router_ids: row.router_ids ?? [],
          monitoring_protocol: row.monitoring_protocol,
          snmp_version: row.snmp_version,
          snmp_port: row.snmp_port,
          snmp_username: row.snmp_username,
          snmp_secret: "",
          polling_interval_seconds: row.polling_interval_seconds,
          monitoring_enabled: enabled,
          remarks: row.remarks,
        };

        try {
          await updateNetworkDevice(row.id, payload);
        } catch {
          failedIDs.add(row.id);
          failedNames.push(row.code);
        }
      }

      await load();

      if (failedIDs.size > 0) {
        setSelectedDeviceIDs(failedIDs);
        setError(
          `Monitoring update failed for: ${failedNames.join(", ")}`,
        );
      } else {
        setSelectedDeviceIDs(new Set());
      }
    } catch (e) {
      setError(
        getAPIErrorMessage(
          e,
          "Unable to update selected network devices.",
        ),
      );
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    const initialLoad = window.setTimeout(() => {
      void load();
    }, 0);

    return () => window.clearTimeout(initialLoad);
  }, [load]);
  const show = (row?: NetworkDevice) => {
    setEditing(row ?? null);
    setCustomModel("");
    setForm(
      row
        ? {
            code: row.code,
            name: row.name,
            device_type: row.device_type,
            vendor: row.vendor,
            model: row.model,
            olt_type: row.olt_type,
            pop_id: row.pop_id,
            management_ip: row.management_ip,
            management_port: row.management_port,
            router_ids: row.router_ids ?? [],
            monitoring_protocol: row.monitoring_protocol,
            snmp_version: row.snmp_version,
            snmp_port: row.snmp_port,
            snmp_username: row.snmp_username,
            snmp_secret: "",
            polling_interval_seconds: row.polling_interval_seconds,
            monitoring_enabled: row.monitoring_enabled,
            remarks: row.remarks,
          }
        : { ...blank },
    );
    setOpen(true);
  };
  const save = async () => {
    try {
      setBusy(true);
      setError("");
      const payload = {
        ...form,
        model:
          form.model === "Other / Custom Model"
            ? customModel.trim()
            : form.model,
      };
      if (editing) {
        await updateNetworkDevice(editing.id, payload);
      } else {
        await createNetworkDevice(payload);
      }
      setOpen(false);
      await load();
    } catch (e) {
      setError(getAPIErrorMessage(e, "Unable to save network device."));
    } finally {
      setBusy(false);
    }
  };
  const remove = async (row: NetworkDevice) => {
    if (!window.confirm(`Delete ${row.name}?`)) return;
    try {
      await deleteNetworkDevice(row.id);
      await load();
    } catch (e) {
      setError(getAPIErrorMessage(e, "Unable to delete device."));
    }
  };
  const testConnection = async (row: NetworkDevice) => {
    try {
      setBusy(true);
      setError("");
      await testNetworkDeviceConnection(row.id);
      await load();
    } catch (e) {
      setError(getAPIErrorMessage(e, "Unable to test SNMP connection."));
      await load();
    } finally {
      setBusy(false);
    }
  };

  const openDetails = async (row: NetworkDevice) => {
    try {
      setDetailDevice(row);
      setDetailPorts([]);
      setDetailONUs([]);
      setLoadingPorts(true);
      setError("");

      const [ports, onus] = await Promise.all([
        getNetworkDevicePorts(row.id),
        getNetworkDeviceONUs(row.id),
      ]);

      setDetailPorts(
        Array.isArray(ports) ? ports : [],
      );
      setDetailONUs(
        Array.isArray(onus) ? onus : [],
      );
    } catch (e) {
      setDetailPorts([]);
      setDetailONUs([]);
      setError(
        getAPIErrorMessage(
          e,
          "Unable to load network device telemetry.",
        ),
      );
    } finally {
      setLoadingPorts(false);
    }
  };

  const handleONUSort = (key: ONUSortKey) => {
    if (onuSortBy === key) {
      setONUSortDirection((current) =>
        current === "asc" ? "desc" : "asc",
      );
      return;
    }

    setONUSortBy(key);
    setONUSortDirection("asc");
  };

  const formatMbps = (value?: number) => {
    if (
      value === undefined ||
      value === null ||
      !Number.isFinite(value)
    ) {
      return "—";
    }

    return `${value.toFixed(2)} Mbps`;
  };

  const formatSpeed = (value: number) => {
    if (!value || value <= 0) {
      return "Unknown";
    }

    if (value >= 1000) {
      const gbps = value / 1000;

      return `${Number.isInteger(gbps) ? gbps : gbps.toFixed(1)} Gbps`;
    }

    return `${value} Mbps`;
  };  const oltEthernetPorts = useMemo(
    () =>
      detailPorts.filter((port) =>
        /^GE0\/\d+(?:\s|$)/i.test(
          (port.name ?? "").trim(),
        ),
      ),
    [detailPorts],
  );

  const oltPONPorts = useMemo(
    () =>
      detailPorts.filter((port) =>
        /^EPON0\/\d+(?:\s|$)/i.test(
          (port.name ?? "").trim(),
        ),
      ),
    [detailPorts],
  );
  const oltOtherInterfaces = useMemo(
    () =>
      detailPorts.filter((port) => {
        const name = (port.name ?? "").trim();

        return (
          !/^GE0\/\d+(?:\s|$)/i.test(name) &&
          !/^EPON0\/\d+(?:\s|$)/i.test(name) &&
          !/^EPON\d{2}ONU\d+(?:\s|$)/i.test(name) &&
            !/^EPON0\/\d+:\d+(?:\s|$)/i.test(name)
        );
      }),
    [detailPorts],
  );
  return (
    <Box>
      <Box sx={{ display: "flex", alignItems: "center", flexWrap: "wrap", gap: 2, mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Network Integration
          </Typography>
          <Typography color="text.secondary">
            OLT, switch and MikroTik monitoring inventory.
          </Typography>
        </Box>
        <Box sx={{ flexGrow: 1 }} />
        {isSuper && (
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => show()}
          >
            Add Network Device
          </Button>
        )}
      </Box>
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      {isSuper && selectedRows.length > 0 && (
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            flexWrap: "wrap",
            gap: 1,
            mb: 2,
          }}
        >
          <Chip
            label={`Selected: ${selectedRows.length}`}
            color="primary"
            variant="outlined"
          />
          <Button
            variant="outlined"
            disabled={busy || !canEnableSelected}
            onClick={() => void updateSelectedMonitoring(true)}
          >
            Enable Monitoring
          </Button>
          <Button
            variant="outlined"
            color="warning"
            disabled={busy || !canDisableSelected}
            onClick={() => void updateSelectedMonitoring(false)}
          >
            Disable Monitoring
          </Button>
          <Button
            variant="text"
            disabled={busy}
            onClick={() => setSelectedDeviceIDs(new Set())}
          >
            Clear Selection
          </Button>
        </Box>
      )}
      <Box sx={{ display: { xs: "grid", md: "none" }, gap: 2 }}>
        {safeRows.map((r) => (
          <Card key={r.id} variant="outlined">
            <CardContent>
              <Box sx={{ display: "flex", alignItems: "flex-start", gap: 1 }}>
                <Box sx={{ minWidth: 0, flex: 1 }}>
                  <Typography variant="h6" sx={{ fontWeight: 700, overflowWrap: "anywhere" }}>
                    {r.code} — {r.name}
                  </Typography>
                  <Typography color="text.secondary">
                    {r.device_type} · {r.vendor} · {r.model}
                    {r.olt_type ? ` · ${r.olt_type}` : ""}
                  </Typography>
                </Box>
                <Chip
                  size="small"
                  label={r.monitoring_enabled ? r.monitoring_status : "DISABLED"}
                  color={r.monitoring_status === "ONLINE" ? "success" : r.monitoring_status === "OFFLINE" ? "error" : "default"}
                />
              </Box>
              <Box sx={{ mt: 2, display: "grid", gridTemplateColumns: "110px minmax(0, 1fr)", gap: 0.75 }}>
                <Typography color="text.secondary">POP</Typography>
                <Typography>{r.pop_name || "—"}</Typography>
                <Typography color="text.secondary">Management</Typography>
                <Typography sx={{ overflowWrap: "anywhere" }}>
                  {r.management_ip}{r.management_port > 0 ? `:${r.management_port}` : ""}
                </Typography>
                <Typography color="text.secondary">SNMP</Typography>
                <Typography>{r.snmp_version} · UDP {r.snmp_port}</Typography>
                <Typography color="text.secondary">Community</Typography>
                <Typography color="text.secondary">{r.credential_configured ? "Credential set" : "Not configured"}</Typography>
                <Typography color="text.secondary">Polling</Typography>
                <Typography>{r.polling_interval_seconds} seconds</Typography>
                {(r.router_names ?? []).length > 0 && <>
                  <Typography color="text.secondary">MikroTik</Typography>
                  <Typography sx={{ overflowWrap: "anywhere" }}>{r.router_names.join(", ")}</Typography>
                </>}
                {r.last_error && <>
                  <Typography color="error">Last Error</Typography>
                  <Typography color="error" sx={{ overflowWrap: "anywhere" }}>{r.last_error}</Typography>
                </>}
              </Box>
              <Box sx={{ display: "flex", justifyContent: "flex-end", mt: 1 }}>
                <IconButton
                  title="View device details"
                  onClick={() => void openDetails(r)}
                >
                  <VisibilityIcon />
                </IconButton>
                {!isAgent && (
                  <IconButton color="primary" disabled={busy || r.monitoring_protocol !== "SNMP"} title="Test SNMP connection" onClick={() => void testConnection(r)}>
                    <PlayCircleIcon />
                  </IconButton>
                )}
                {isSuper && <>
                  <IconButton title="Edit device" onClick={() => show(r)}><EditIcon /></IconButton>
                  <IconButton title="Delete device" color="error" onClick={() => void remove(r)}><DeleteIcon /></IconButton>
                </>}
              </Box>
            </CardContent>
          </Card>
        ))}
        {safeRows.length === 0 && <Card variant="outlined"><CardContent><Typography align="center">No OLT or switch has been added.</Typography></CardContent></Card>}
      </Box>
      <Card sx={{ display: { xs: "none", md: "block" } }}>
        <CardContent>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Serial</TableCell>
                  <TableCell padding="checkbox">
                    <Checkbox
                      checked={allVisibleSelected}
                      indeterminate={someVisibleSelected}
                      onChange={(event) =>
                        toggleVisibleDevices(event.target.checked)
                      }
                      slotProps={{
                        input: {
                          "aria-label": "Select all visible network devices",
                        },
                      }}
                    />
                  </TableCell>
                  <TableCell>Device</TableCell>
                  <TableCell>Type / Vendor</TableCell>
                  <TableCell>Model</TableCell>
                  <TableCell>POP</TableCell>
                  <TableCell>Management</TableCell>
                  <TableCell>Monitoring</TableCell>
                  <TableCell align="right">Action</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {safeRows.map((r, index) => (
                  <TableRow key={r.id} selected={selectedDeviceIDs.has(r.id)}>
                    <TableCell>{index + 1}</TableCell>
                    <TableCell padding="checkbox">
                      <Checkbox
                        checked={selectedDeviceIDs.has(r.id)}
                        onChange={() => toggleDeviceSelection(r.id)}
                        slotProps={{
                          input: {
                            "aria-label": `Select ${r.code}`,
                          },
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <b>{r.code}</b>
                      <br />
                      {r.name}
                    </TableCell>
                    <TableCell>
                      {r.device_type}
                      <br />
                      {r.vendor}
                    </TableCell>
                    <TableCell>{r.model}{r.olt_type ? ` · ${r.olt_type}` : ""}</TableCell>
                    <TableCell>{r.pop_name || "—"}</TableCell>
                    <TableCell>
                      {r.management_ip}{r.management_port > 0 ? `:${r.management_port}` : ""}
                      <br />
                      {r.monitoring_protocol === "SNMP"
                        ? `${r.snmp_version} · UDP ${r.snmp_port}`
                        : "MikroTik API"}
                    </TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={
                          r.monitoring_enabled
                            ? r.monitoring_status
                            : "DISABLED"
                        }
                        color={
                          r.monitoring_status === "ONLINE"
                            ? "success"
                            : r.monitoring_status === "OFFLINE"
                              ? "error"
                              : "default"
                        }
                      />
                      <br />
                      {r.polling_interval_seconds}s ·{" "}
                      {r.credential_configured
                        ? "Credential set"
                        : "No credential"}

                      {(r.router_names ?? []).length > 0 && <><br />MikroTik: {r.router_names.join(", ")}</>}
                    </TableCell>
                    <TableCell align="right">
                      <IconButton
                        title="View device details"
                        onClick={() => void openDetails(r)}
                      >
                        <VisibilityIcon />
                      </IconButton>
                      {!isAgent && (
                        <IconButton
                          color="primary"
                          disabled={busy || r.monitoring_protocol !== "SNMP"}
                          title="Test SNMP connection"
                          onClick={() => void testConnection(r)}
                        >
                          <PlayCircleIcon />
                        </IconButton>
                      )}
                      {isSuper && (
                        <>
                          <IconButton onClick={() => show(r)}>
                            <EditIcon />
                          </IconButton>
                          <IconButton
                            color="error"
                            onClick={() => void remove(r)}
                          >
                            <DeleteIcon />
                          </IconButton>
                        </>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {safeRows.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} align="center">
                      No OLT or switch has been added.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>
      <Dialog
        open={detailDevice !== null}
        onClose={() => {
          if (!loadingPorts) {
            setDetailDevice(null);
            setDetailPorts([]);
          }
        }}
        fullWidth
        maxWidth="xl"
      >
        <DialogTitle>
          {detailDevice
            ? `${detailDevice.code} — ${detailDevice.name}`
            : "Network Device Details"}
        </DialogTitle>

        <DialogContent>
          {detailDevice && (
            <>
              <Grid
                container
                spacing={2}
                sx={{ mb: 3 }}
              >
                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <Typography color="text.secondary">
                    Type
                  </Typography>
                  <Typography>
                    {detailDevice.device_type}
                  </Typography>
                </Grid>

                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <Typography color="text.secondary">
                    Vendor / Model
                  </Typography>
                  <Typography>
                    {detailDevice.vendor} · {detailDevice.model}
                  </Typography>
                </Grid>

                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <Typography color="text.secondary">
                    Management
                  </Typography>
                  <Typography>
                    {detailDevice.management_ip}
                    {detailDevice.management_port > 0
                      ? `:${detailDevice.management_port}`
                      : ""}
                  </Typography>
                </Grid>

                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <Typography color="text.secondary">
                    Monitoring
                  </Typography>
                  <Chip
                    size="small"
                    label={
                      detailDevice.monitoring_enabled
                        ? detailDevice.monitoring_status
                        : "DISABLED"
                    }
                    color={
                      detailDevice.monitoring_status === "ONLINE"
                        ? "success"
                        : detailDevice.monitoring_status === "OFFLINE"
                          ? "error"
                          : "default"
                    }
                  />
                </Grid>
              </Grid>

              <Typography
                variant="h6"
                sx={{ fontWeight: 700, mb: 1 }}
              >
                {detailDevice.device_type === "OLT"
                  ? `Ethernet / Uplink Ports (${oltEthernetPorts.length})`
                  : `Ports (${detailPorts.length})`}
              </Typography>

              <TableContainer>
                <Table
                  size="small"
                  sx={{ minWidth: 1250 }}
                >
                  <TableHead>
                    <TableRow>
                      <TableCell>Port</TableCell>
                      <TableCell>Type</TableCell>
                      <TableCell>Admin</TableCell>
                      <TableCell>Oper</TableCell>
                      <TableCell>Speed</TableCell>
                      <TableCell>RX</TableCell>
                      <TableCell>TX</TableCell>
                      <TableCell>Errors</TableCell>
                      <TableCell>Discards</TableCell>
                      <TableCell>MAC</TableCell>
                      <TableCell>Last Seen</TableCell>
                    </TableRow>
                  </TableHead>

                  <TableBody>
                    {(detailDevice.device_type === "OLT"
                      ? oltEthernetPorts
                      : detailPorts
                    ).map((port) => (
                      <TableRow key={port.id}>
                        <TableCell>
                          <b>
                            {port.name ||
                              port.description ||
                              port.port_key}
                          </b>
                          <br />
                          <Typography
                            variant="caption"
                            color="text.secondary"
                          >
                            ifIndex{" "}
                            {port.if_index ?? "—"}
                            {port.description &&
                            port.description !== port.name
                              ? ` · ${port.description}`
                              : ""}
                          </Typography>
                        </TableCell>

                        <TableCell>
                          {port.port_type || "—"}
                        </TableCell>

                        <TableCell>
                          <Chip
                            size="small"
                            label={port.admin_status || "UNKNOWN"}
                            color={
                              port.admin_status === "UP"
                                ? "success"
                                : "default"
                            }
                          />
                        </TableCell>

                        <TableCell>
                          <Chip
                            size="small"
                            label={port.oper_status || "UNKNOWN"}
                            color={
                              port.oper_status === "UP"
                                ? "success"
                                : port.oper_status === "DOWN"
                                  ? "error"
                                  : "default"
                            }
                          />
                        </TableCell>

                        <TableCell>
                          {formatSpeed(port.speed_mbps)}
                        </TableCell>

                        <TableCell>
                          {formatMbps(
                            port.latest_sample?.in_mbps,
                          )}
                        </TableCell>

                        <TableCell>
                          {formatMbps(
                            port.latest_sample?.out_mbps,
                          )}
                        </TableCell>

                        <TableCell>
                          {port.latest_sample
                            ? `${port.latest_sample.in_errors} / ${port.latest_sample.out_errors}`
                            : "—"}
                        </TableCell>

                        <TableCell>
                          {port.latest_sample
                            ? `${port.latest_sample.in_discards} / ${port.latest_sample.out_discards}`
                            : "—"}
                        </TableCell>

                        <TableCell>
                          {port.mac_address || "—"}
                        </TableCell>

                        <TableCell>
                          {port.last_seen_at
                            ? new Date(
                                port.last_seen_at,
                              ).toLocaleString()
                            : "—"}
                        </TableCell>
                      </TableRow>
                    ))}

                    {loadingPorts && (
                      <TableRow>
                        <TableCell
                          colSpan={11}
                          align="center"
                        >
                          Loading ports...
                        </TableCell>
                      </TableRow>
                    )}

                    {!loadingPorts &&
                      (detailDevice.device_type === "OLT"
                        ? oltEthernetPorts.length === 0
                        : detailPorts.length === 0) && (
                        <TableRow>
                          <TableCell
                            colSpan={11}
                            align="center"
                          >
                            No port telemetry recorded yet.
                          </TableCell>
                        </TableRow>
                      )}
                  </TableBody>
                </Table>
              </TableContainer>

              {detailDevice.device_type === "OLT" && (
                <>
                  <Typography
                    variant="h6"
                    sx={{ fontWeight: 700, mt: 4, mb: 1 }}
                  >
                    PON Ports ({oltPONPorts.length})
                  </Typography>

                  <Box
                    sx={{
                      display: "grid",
                      gridTemplateColumns: {
                        xs: "1fr",
                        sm: "repeat(2, minmax(0, 1fr))",
                        lg: "repeat(6, minmax(0, 1fr))",
                      },
                      gap: 1.5,
                      mb: 2,
                    }}
                  >
                    <TextField
                      size="small"
                      label="Search ONU"
                      placeholder="ONU ID / MAC / Description"
                      value={onuSearch}
                      onChange={(event) =>
                        setONUSearch(event.target.value)
                      }
                    />

                    <TextField
                      size="small"
                      select
                      label="PON"
                      value={onuPONFilter}
                      onChange={(event) =>
                        setONUPONFilter(
                          event.target.value,
                        )
                      }
                    >
                      <MenuItem value="ALL">
                        All PONs
                      </MenuItem>

                      {oltPONPorts.map((pon) => {
                        const match = (
                          pon.name ?? ""
                        ).match(
                          /^EPON0\/(\d+)/i,
                        );

                        if (!match) {
                          return null;
                        }

                        const ponNo = String(
                          Number(match[1]),
                        );

                        return (
                          <MenuItem
                            key={pon.id}
                            value={ponNo}
                          >
                            {pon.name ||
                              `PON ${ponNo}`}
                          </MenuItem>
                        );
                      })}
                    </TextField>

                    <TextField
                      size="small"
                      select
                      label="Status"
                      value={onuStatusFilter}
                      onChange={(event) =>
                        setONUStatusFilter(
                          event.target.value as
                            | "ALL"
                            | "ONLINE"
                            | "OFFLINE",
                        )
                      }
                    >
                      <MenuItem value="ALL">
                        All
                      </MenuItem>
                      <MenuItem value="ONLINE">
                        Online
                      </MenuItem>
                      <MenuItem value="OFFLINE">
                        Offline
                      </MenuItem>
                    </TextField>

                    <TextField
                      size="small"
                      select
                      label="Optical Status"
                      value={onuOpticalFilter}
                      onChange={(event) =>
                        setONUOpticalFilter(
                          event.target.value as
                            | "ALL"
                            | "NORMAL"
                            | "WEAK"
                            | "TOO_RISK"
                            | "UNAVAILABLE",
                        )
                      }
                    >
                      <MenuItem value="ALL">
                        All
                      </MenuItem>
                      <MenuItem value="NORMAL">
                        Normal
                      </MenuItem>
                      <MenuItem value="WEAK">
                        Weak
                      </MenuItem>
                      <MenuItem value="TOO_RISK">
                        Too Risk
                      </MenuItem>
                      <MenuItem value="UNAVAILABLE">
                        Unavailable
                      </MenuItem>
                    </TextField>

                    <TextField
                      size="small"
                      label="ONU Number"
                      placeholder="e.g. 12"
                      value={onuNumberFilter}
                      onChange={(event) =>
                        setONUNumberFilter(
                          event.target.value.replace(
                            /[^0-9]/g,
                            "",
                          ),
                        )
                      }
                    />

                    <Button
                      variant="outlined"
                      onClick={() => {
                        setONUSearch("");
                        setONUPONFilter("ALL");
                        setONUStatusFilter("ALL");
                        setONUOpticalFilter("ALL");
                        setONUNumberFilter("");
                        setPonONUFilters({});
                      }}
                    >
                      Clear Filters
                    </Button>
                  </Box>


                  <Box
                    sx={{
                      display: "flex",
                      flexDirection: "column",
                      gap: 1.5,
                    }}
                  >
                      {oltPONPorts.map((pon) => {
                        const match = (
                          pon.name ?? ""
                        ).match(/^EPON0\/(\d+)/i);

                        const ponNo = match
                          ? Number(match[1])
                          : null;

                        if (
                          ponNo !== null &&
                          onuPONFilter !== "ALL" &&
                          String(ponNo) !==
                            onuPONFilter
                        ) {
                          return null;
                        }

                        const onus =
                          ponNo === null
                            ? []
                            : detailONUs.filter(
                                (onu) => {
                                  if (
                                    onu.pon_no !== ponNo
                                  ) {
                                    return false;
                                  }

                                  const search =
                                    onuSearch
                                      .trim()
                                      .toLowerCase();

                                  if (search) {
                                    const onuID =
                                      `EPON0/${onu.pon_no}:${onu.onu_no}`;

                                    const haystack = [
                                      onuID,
                                      onu.mac_address,
                                      onu.description,
                                    ]
                                      .filter(Boolean)
                                      .join(" ")
                                      .toLowerCase();

                                    if (
                                      !haystack.includes(
                                        search,
                                      )
                                    ) {
                                      return false;
                                    }
                                  }

                                  if (
                                    onuNumberFilter &&
                                    String(
                                      onu.onu_no,
                                    ) !==
                                      onuNumberFilter
                                  ) {
                                    return false;
                                  }

                                  if (
                                    onuStatusFilter ===
                                      "ONLINE" &&
                                    onu.oper_status !==
                                      "UP"
                                  ) {
                                    return false;
                                  }

                                  if (
                                    onuStatusFilter ===
                                      "OFFLINE" &&
                                    onu.oper_status !==
                                      "DOWN"
                                  ) {
                                    return false;
                                  }

                                  const rxPower =
                                    onu.latest_optical
                                      ?.rx_power_dbm;

                                  if (
                                    onuOpticalFilter ===
                                      "UNAVAILABLE"
                                  ) {
                                    return (
                                      rxPower === null ||
                                      rxPower === undefined
                                    );
                                  }

                                  if (
                                    onuOpticalFilter !==
                                      "ALL"
                                  ) {
                                    if (
                                      rxPower === null ||
                                      rxPower === undefined
                                    ) {
                                      return false;
                                    }

                                    if (
                                      onuOpticalFilter ===
                                        "NORMAL" &&
                                      !(
                                        rxPower >= -25 &&
                                        rxPower <= -7
                                      )
                                    ) {
                                      return false;
                                    }

                                    if (
                                      onuOpticalFilter ===
                                        "WEAK" &&
                                      !(rxPower < -25)
                                    ) {
                                      return false;
                                    }

                                    if (
                                      onuOpticalFilter ===
                                        "TOO_RISK" &&
                                      !(
                                        rxPower > -7 &&
                                        rxPower <= 10
                                      )
                                    ) {
                                      return false;
                                    }
                                  }

                                  return true;
                                },
                              );

                        const onuFilter =
                          ponONUFilters[pon.id] ??
                          "ALL";

                        const online =
                          onus.filter(
                            (onu) =>
                              onu.oper_status === "UP",
                          ).length;

                        const offline =
                          onus.filter(
                            (onu) =>
                              onu.oper_status === "DOWN",
                          ).length;

                        const visibleONUs =
                          onus.filter((onu) => {
                            if (
                              onuFilter === "ONLINE"
                            ) {
                              return (
                                onu.oper_status === "UP"
                              );
                            }

                            if (
                              onuFilter === "OFFLINE"
                            ) {
                              return (
                                onu.oper_status === "DOWN"
                              );
                            }

                            return true;
                          });

                        const getONUSortValue = (
                          onu: NetworkDeviceONU,
                        ): string | number | null => {
                          switch (onuSortBy) {
                            case "onu":
                              return onu.onu_no;

                            case "mac":
                              return (
                                onu.mac_address ?? ""
                              ).toLowerCase();

                            case "description":
                              return (
                                onu.description ?? ""
                              ).toLowerCase();

                            case "status":
                              return (
                                onu.oper_status ?? ""
                              ).toLowerCase();

                            case "rx_power":
                              return (
                                onu.latest_optical
                                  ?.rx_power_dbm ??
                                null
                              );

                            case "rx_mbps":
                              return (
                                onu.latest_sample
                                  ?.in_mbps ??
                                null
                              );

                            case "tx_mbps":
                              return (
                                onu.latest_sample
                                  ?.out_mbps ??
                                null
                              );

                            case "last_down":
                              return onu.last_deregistered_at
                                ? new Date(
                                    onu.last_deregistered_at,
                                  ).getTime()
                                : null;

                            default:
                              return onu.onu_no;
                          }
                        };

                        const sortedONUs = [
                          ...visibleONUs,
                        ].sort((a, b) => {
                          const aValue =
                            getONUSortValue(a);
                          const bValue =
                            getONUSortValue(b);

                          if (
                            aValue === null &&
                            bValue === null
                          ) {
                            return 0;
                          }

                          if (aValue === null) {
                            return 1;
                          }

                          if (bValue === null) {
                            return -1;
                          }

                          let result: number;

                          if (
                            typeof aValue ===
                              "number" &&
                            typeof bValue ===
                              "number"
                          ) {
                            result =
                              aValue - bValue;
                          } else {
                            result = String(
                              aValue,
                            ).localeCompare(
                              String(bValue),
                            );
                          }

                          return onuSortDirection ===
                            "asc"
                            ? result
                            : -result;
                        });

                        const setONUFilter = (
                          filter:
                            | "ALL"
                            | "ONLINE"
                            | "OFFLINE",
                        ) => {
                          setPonONUFilters(
                            (current) => ({
                              ...current,
                              [pon.id]: filter,
                            }),
                          );
                        };

                        const formatTelemetryNumber = (
                          value:
                            | number
                            | null
                            | undefined,
                          digits = 2,
                        ) => {
                          if (
                            value === null ||
                            value === undefined ||
                            !Number.isFinite(value)
                          ) {
                            return "—";
                          }

                          return value.toFixed(digits);
                        };


                        return (
                          <Box
                            component="details"
                            key={pon.id}
                            sx={{
                              border: 1,
                              borderColor: "divider",
                              borderRadius: 1,
                              overflow: "hidden",
                            }}
                          >
                            <Box
                              component="summary"
                              sx={{
                                cursor: "pointer",
                                px: 2,
                                py: 1.5,
                                bgcolor: "action.hover",
                              }}
                            >
                              <Box
                                sx={{
                                  display: "flex",
                                  alignItems: "center",
                                  flexWrap: "wrap",
                                  gap: 1,
                                }}
                              >
                                <Typography
                                  sx={{ fontWeight: 700 }}
                                >
                                  {pon.name ||
                                    pon.port_key}
                                </Typography>

                                <Chip
                                  size="small"
                                  label={
                                    pon.oper_status ||
                                    "UNKNOWN"
                                  }
                                  color={
                                    pon.oper_status ===
                                    "UP"
                                      ? "success"
                                      : pon.oper_status ===
                                          "DOWN"
                                        ? "error"
                                        : "default"
                                  }
                                />

                                <Chip
                                  size="small"
                                  variant={
                                    onuFilter === "ALL"
                                      ? "filled"
                                      : "outlined"
                                  }
                                  label={`${onus.length} ONU`}
                                  onClick={(event) => {
                                    event.preventDefault();
                                    event.stopPropagation();
                                    setONUFilter("ALL");
                                  }}
                                  sx={{
                                    cursor: "pointer",
                                  }}
                                />

                                <Chip
                                  size="small"
                                  variant={
                                    onuFilter ===
                                    "ONLINE"
                                      ? "filled"
                                      : "outlined"
                                  }
                                  color="success"
                                  label={`${online} Online`}
                                  onClick={(event) => {
                                    event.preventDefault();
                                    event.stopPropagation();
                                    setONUFilter(
                                      "ONLINE",
                                    );
                                  }}
                                  sx={{
                                    cursor: "pointer",
                                  }}
                                />

                                <Chip
                                  size="small"
                                  variant={
                                    onuFilter ===
                                    "OFFLINE"
                                      ? "filled"
                                      : "outlined"
                                  }
                                  color={
                                    offline > 0
                                      ? "error"
                                      : "default"
                                  }
                                  label={`${offline} Offline`}
                                  onClick={(event) => {
                                    event.preventDefault();
                                    event.stopPropagation();
                                    setONUFilter(
                                      "OFFLINE",
                                    );
                                  }}
                                  sx={{
                                    cursor: "pointer",
                                  }}
                                />

                                <Typography
                                  variant="body2"
                                  color="text.secondary"
                                >
                                  RX{" "}
                                  {formatMbps(
                                    pon.latest_sample
                                      ?.in_mbps,
                                  )}
                                  {" · TX "}
                                  {formatMbps(
                                    pon.latest_sample
                                      ?.out_mbps,
                                  )}
                                </Typography>
                              </Box>
                            </Box>

                            <TableContainer>
                              <Table
                                size="small"
                                sx={{ minWidth: 1500 }}
                              >
                                <TableHead>
                                  <TableRow>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "onu"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "onu"
                                        }
                                        direction={
                                          onuSortBy === "onu"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "onu",
                                          )
                                        }
                                      >
                                        ONU ID
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "mac"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "mac"
                                        }
                                        direction={
                                          onuSortBy === "mac"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "mac",
                                          )
                                        }
                                      >
                                        MAC Address
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "description"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "description"
                                        }
                                        direction={
                                          onuSortBy === "description"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "description",
                                          )
                                        }
                                      >
                                        Description
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "status"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "status"
                                        }
                                        direction={
                                          onuSortBy === "status"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "status",
                                          )
                                        }
                                      >
                                        Status
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell>
                                      VLAN
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "rx_power"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "rx_power"
                                        }
                                        direction={
                                          onuSortBy === "rx_power"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "rx_power",
                                          )
                                        }
                                      >
                                        ONU Laser
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "rx_mbps"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "rx_mbps"
                                        }
                                        direction={
                                          onuSortBy === "rx_mbps"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "rx_mbps",
                                          )
                                        }
                                      >
                                        RX Mbps
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "tx_mbps"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "tx_mbps"
                                        }
                                        direction={
                                          onuSortBy === "tx_mbps"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "tx_mbps",
                                          )
                                        }
                                      >
                                        TX Mbps
                                      </TableSortLabel>
                                    </TableCell>
                                    <TableCell
                                      sortDirection={
                                        onuSortBy === "last_down"
                                          ? onuSortDirection
                                          : false
                                      }
                                    >
                                      <TableSortLabel
                                        active={
                                          onuSortBy === "last_down"
                                        }
                                        direction={
                                          onuSortBy === "last_down"
                                            ? onuSortDirection
                                            : "asc"
                                        }
                                        onClick={() =>
                                          handleONUSort(
                                            "last_down",
                                          )
                                        }
                                      >
                                        Last Down Time
                                      </TableSortLabel>
                                    </TableCell>
                                  </TableRow>
                                </TableHead>

                                <TableBody>
                                  {sortedONUs.map(
                                    (onu) => (
                                      <TableRow
                                        key={onu.id}
                                        hover
                                      >
                                        <TableCell>
                                          <b>
                                            {`EPON0/${onu.pon_no}:${onu.onu_no}`}
                                          </b>
                                        </TableCell>

                                        <TableCell>
                                          {onu.mac_address ||
                                            "—"}
                                        </TableCell>

                                        <TableCell>
                                          {onu.description ||
                                            "—"}
                                        </TableCell>

                                        <TableCell>
                                          <Chip
                                            size="small"
                                            label={
                                              onu.oper_status ||
                                              "UNKNOWN"
                                            }
                                            color={
                                              onu.oper_status ===
                                              "UP"
                                                ? "success"
                                                : onu.oper_status ===
                                                    "DOWN"
                                                  ? "error"
                                                  : "default"
                                            }
                                          />
                                        </TableCell>

                                        <TableCell>
                                          —
                                        </TableCell>




                                        <TableCell>
                                          {onu.latest_optical
                                            ?.rx_power_dbm ==
                                          null
                                            ? "—"
                                            : `${formatTelemetryNumber(
                                                onu.latest_optical
                                                  .rx_power_dbm,
                                                2,
                                              )} dBm`}
                                        </TableCell>

                                        <TableCell>
                                          {formatMbps(
                                            onu.latest_sample
                                              ?.in_mbps,
                                          )}
                                        </TableCell>

                                        <TableCell>
                                          {formatMbps(
                                            onu.latest_sample
                                              ?.out_mbps,
                                          )}
                                        </TableCell>
                                        <TableCell>
                                          {onu.last_deregistered_at
                                            ? new Date(
                                                onu.last_deregistered_at,
                                              ).toLocaleString()
                                            : "—"}
                                        </TableCell>
                                      </TableRow>
                                    ),
                                  )}

                                  {visibleONUs.length ===
                                    0 && (
                                    <TableRow>
                                      <TableCell
                                        colSpan={9}
                                        align="center"
                                      >
                                        {onus.length === 0
                                          ? "No canonical ONU records detected on this PON."
                                          : onuFilter ===
                                              "ONLINE"
                                            ? "No online ONU on this PON."
                                            : "No offline ONU on this PON."}
                                      </TableCell>
                                    </TableRow>
                                  )}
                                </TableBody>
                              </Table>
                            </TableContainer>
                          </Box>
                        );
                      })}
                  </Box>

                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ mt: 2 }}
                  >
                    Canonical ONUs:{" "}
                      <b>
                        {detailONUs.length}
                      </b>
                    {" · "}
                    Other interfaces:{" "}
                    <b>
                      {oltOtherInterfaces.length}
                    </b>
                  </Typography>
                </>
              )}
            </>
          )}
        </DialogContent>

        <DialogActions>
          <Button
            onClick={() => {
              setDetailDevice(null);
              setDetailPorts([]);
            }}
            disabled={loadingPorts}
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>

      {open && <Dialog
        open={open}
        onClose={() => !busy && setOpen(false)}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>{editing ? "Edit" : "Add"} Network Device</DialogTitle>
        <DialogContent>
          <Alert severity="info" sx={{ mb: 2 }}>
            Credentials are encrypted. Existing credentials remain unchanged
            when the secret field is blank.
          </Alert>
          <Grid container spacing={2} sx={{ mt: 0 }}>
            {(
              [
                ["code", "Device Code"],
                ["name", "Device Name"],
                ["management_ip", "Management IP / Host"],
              ] as const
            ).map(([k, l]) => (
              <Grid size={{ xs: 12, md: 4 }} key={k}>
                <TextField
                  fullWidth
                  required
                  label={l}
                  value={form[k]}
                  onChange={(e) => setForm({ ...form, [k]: e.target.value })}
                />
              </Grid>
            ))}
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                select
                fullWidth
                label="Device Type"
                value={form.device_type}
                onChange={(e) => {
                  const deviceType = e.target.value;
                  const defaults = deviceType === "SWITCH"
                    ? { vendor: "FOCUSCOM", model: "8-Port Switch", olt_type: "", management_port: 80 }
                    : deviceType === "MIKROTIK"
                      ? { vendor: "MIKROTIK", model: "CCR2116-12G-4S+", olt_type: "", management_port: 8729 }
                      : { vendor: "VSOL", model: "10G G/EPON OLT", olt_type: "EPON", management_port: 9001 };
                  setForm({ ...form, device_type: deviceType, ...defaults });
                  setCustomModel("");
                }}
              >
                {["OLT", "SWITCH", "MIKROTIK"].map((x) => (
                  <MenuItem key={x} value={x}>
                    {x}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                select
                fullWidth
                label="Vendor"
                value={form.vendor}
                onChange={(e) =>
                  setForm({
                    ...form,
                    vendor: e.target.value,
                    model: (catalogs[e.target.value] ?? catalogs.OTHER)[0],
                  })
                }
              >
                {Object.keys(catalogs).map((x) => (
                  <MenuItem key={x} value={x}>
                    {x}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            {form.device_type === "OLT" && (
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  select
                  required
                  fullWidth
                  label="OLT Type"
                  value={form.olt_type}
                  onChange={(e) => setForm({ ...form, olt_type: e.target.value })}
                >
                  {["EPON", "GPON", "G/EPON", "XG-PON", "XGS-PON"].map((type) => (
                    <MenuItem key={type} value={type}>{type}</MenuItem>
                  ))}
                </TextField>
              </Grid>
            )}
            {(form.device_type === "OLT" || form.device_type === "SWITCH") && (
              <>
                <Grid size={{ xs: 12, md: 4 }}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Management Port"
                    helperText="Web, Telnet or vendor management port; SNMP remains separate."
                    value={form.management_port}
                    slotProps={{ htmlInput: { min: 1, max: 65535 } }}
                    onChange={(e) => setForm({ ...form, management_port: Number(e.target.value) })}
                  />
                </Grid>
                <Grid size={{ xs: 12, md: 4 }}>
                  <TextField
                    select
                    fullWidth
                    label="Assigned MikroTik Routers"
                    value={form.router_ids}
                    slotProps={{ select: { multiple: true } }}
                    onChange={(e) => setForm({
                      ...form,
                      router_ids: (e.target.value as unknown as number[]).map(Number),
                    })}
                  >
                    {routers.map((router) => (
                      <MenuItem key={router.id} value={router.id}>
                        <Checkbox checked={form.router_ids.includes(router.id)} />
                        {router.code} — {router.name}
                      </MenuItem>
                    ))}
                  </TextField>
                </Grid>
              </>
            )}
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                select
                fullWidth
                label="Model"
                value={form.model}
                onChange={(e) => setForm({ ...form, model: e.target.value })}
              >
                {(models.includes(form.model)
                  ? models
                  : models.concat(form.model)
                ).map((x) => (
                  <MenuItem key={x} value={x}>
                    {x}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            {form.model === "Other / Custom Model" && (
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  required
                  fullWidth
                  label="Custom Model"
                  value={customModel}
                  onChange={(e) => setCustomModel(e.target.value)}
                />
              </Grid>
            )}
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                select
                fullWidth
                label="POP"
                value={form.pop_id ?? ""}
                onChange={(e) =>
                  setForm({
                    ...form,
                    pop_id: e.target.value ? Number(e.target.value) : undefined,
                  })
                }
              >
                <MenuItem value="">No POP</MenuItem>
                {pops.map((p) => (
                  <MenuItem key={p.id} value={p.id}>
                    {p.code} — {p.name}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                select
                fullWidth
                label="Protocol"
                value={form.monitoring_protocol}
                onChange={(e) =>
                  setForm({ ...form, monitoring_protocol: e.target.value })
                }
              >
                <MenuItem value="SNMP">SNMP</MenuItem>
                <MenuItem value="MIKROTIK_API">MikroTik API</MenuItem>
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                select
                fullWidth
                label="SNMP Version"
                disabled={form.monitoring_protocol !== "SNMP"}
                value={form.snmp_version}
                onChange={(e) =>
                  setForm({ ...form, snmp_version: e.target.value })
                }
              >
                <MenuItem value="V2C">v2c</MenuItem>
                <MenuItem value="V3">v3</MenuItem>
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                fullWidth
                type="number"
                label="SNMP Port"
                value={form.snmp_port}
                onChange={(e) =>
                  setForm({ ...form, snmp_port: Number(e.target.value) })
                }
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                fullWidth
                label="SNMP v3 Username"
                value={form.snmp_username}
                onChange={(e) =>
                  setForm({ ...form, snmp_username: e.target.value })
                }
              />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField
                fullWidth
                type="password"
                label={
                  form.snmp_version === "V2C"
                    ? "Community"
                    : "Authentication Secret"
                }
                value={form.snmp_secret}
                onChange={(e) =>
                  setForm({ ...form, snmp_secret: e.target.value })
                }
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                select
                fullWidth
                label="Polling Interval"
                value={form.polling_interval_seconds}
                onChange={(e) =>
                  setForm({
                    ...form,
                    polling_interval_seconds: Number(e.target.value),
                  })
                }
              >
                {[30, 60, 300, 1800, 3600, 86400].map((x) => (
                  <MenuItem key={x} value={x}>
                    {x < 60
                      ? `${x} seconds`
                      : x < 3600
                        ? `${x / 60} minutes`
                        : x < 86400
                          ? `${x / 3600} hour`
                          : "24 hours"}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={form.monitoring_enabled}
                    onChange={(e) =>
                      setForm({ ...form, monitoring_enabled: e.target.checked })
                    }
                  />
                }
                label="Monitoring enabled"
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField
                fullWidth
                multiline
                rows={2}
                label="Remarks"
                value={form.remarks}
                onChange={(e) => setForm({ ...form, remarks: e.target.value })}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            disabled={busy}
            onClick={() => void save()}
          >
            {busy ? "Saving..." : "Save Device"}
          </Button>
        </DialogActions>
      </Dialog>}
    </Box>
  );
}
