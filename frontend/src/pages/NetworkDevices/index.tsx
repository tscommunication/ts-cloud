import { useEffect, useMemo, useState } from "react";
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
  TextField,
  Typography,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import PlayCircleIcon from "@mui/icons-material/PlayCircle";
import {
  createNetworkDevice,
  deleteNetworkDevice,
  getNetworkDevices,
  testNetworkDeviceConnection,
  updateNetworkDevice,
  type NetworkDevice,
  type NetworkDeviceInput,
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

export default function NetworkDevices() {
  const [rows, setRows] = useState<NetworkDevice[]>([]);
  const [pops, setPops] = useState<POP[]>([]);
  const [routers, setRouters] = useState<NetworkRouter[]>([]);
  const [form, setForm] = useState<NetworkDeviceInput>(blank);
  const [editing, setEditing] = useState<NetworkDevice | null>(null);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [customModel, setCustomModel] = useState("");
  const isSuper = getStoredUser()?.role === "superadmin";
  const models = useMemo(
    () => catalogs[form.vendor] ?? catalogs.OTHER,
    [form.vendor],
  );
  const safeRows = Array.isArray(rows) ? rows : [];
  const load = async () => {
    try {
      setError("");
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
  };
  useEffect(() => {
    const initialLoad = window.setTimeout(() => {
      void load();
    }, 0);

    return () => window.clearTimeout(initialLoad);
  }, []);
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
                <IconButton color="primary" disabled={busy || r.monitoring_protocol !== "SNMP"} title="Test SNMP connection" onClick={() => void testConnection(r)}>
                  <PlayCircleIcon />
                </IconButton>
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
                {safeRows.map((r) => (
                  <TableRow key={r.id}>
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
                        color="primary"
                        disabled={busy || r.monitoring_protocol !== "SNMP"}
                        title="Test SNMP connection"
                        onClick={() => void testConnection(r)}
                      >
                        <PlayCircleIcon />
                      </IconButton>
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
