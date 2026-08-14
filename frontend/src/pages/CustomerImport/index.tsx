import { useState } from "react";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  FormControlLabel,
  Grid,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from "@mui/material";
import UploadFileIcon from "@mui/icons-material/UploadFile";
import { getNetworkRouters } from "../../api/networkRouters";
import {
  importCustomers,
  previewCustomerImport,
  type CustomerImportBatch,
  type CustomerImportPreview,
} from "../../api/customerImports";
import { getAPIErrorMessage } from "../../api/errors";
import { useQuery } from "@tanstack/react-query";
import {
  importAgentUsers,
  previewAgentUserImport,
  type AgentUserImportPreview,
  type AgentUserImportResult,
} from "../../api/agentUserImports";

export default function CustomerImport() {
  const routers = useQuery({
    queryKey: ["network-routers"],
    queryFn: getNetworkRouters,
  });
  const [file, setFile] = useState<File | null>(null),
    [importType, setImportType] = useState<"customers" | "agent-users">("customers"),
    [preview, setPreview] = useState<CustomerImportPreview | null>(null),
    [agentPreview, setAgentPreview] = useState<AgentUserImportPreview | null>(null),
    [routerID, setRouterID] = useState(0),
    [confirmed, setConfirmed] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [result, setResult] = useState<CustomerImportBatch | null>(null),
    [agentResult, setAgentResult] = useState<AgentUserImportResult | null>(null);
  const inspect = async () => {
    if (!file) return;
    try {
      setBusy(true);
      setError("");
      setResult(null);
      setAgentResult(null);
      if (importType === "agent-users") {
        setAgentPreview(await previewAgentUserImport(file));
        setPreview(null);
      } else {
        setPreview(await previewCustomerImport(file));
        setAgentPreview(null);
      }
    } catch (e) {
      setError(getAPIErrorMessage(e, "File preview failed."));
    } finally {
      setBusy(false);
    }
  };
  const execute = async () => {
    if (!file || !confirmed || (importType === "customers" && !routerID)) return;
    try {
      setBusy(true);
      setError("");
      if (importType === "agent-users") {
        setAgentResult(await importAgentUsers(file));
        setAgentPreview(null);
      } else {
        setResult(await importCustomers(file, routerID));
        setPreview(null);
      }
    } catch (e) {
      setError(
        getAPIErrorMessage(e, "Import failed; no partial rows were saved."),
      );
    } finally {
      setBusy(false);
    }
  };
  return (
    <Box>
      <Typography variant="h4" sx={{ fontWeight: 700 }}>
        Data Import &amp; Export
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Preview and safely import customer data or Agent login accounts from
        supported CSV and Excel files.
      </Typography>
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      {result && (
        <Alert severity="success" sx={{ mb: 2 }}>
          Batch #{result.id}: imported {result.imported_rows} customers; created{" "}
          {result.created_packages} packages, {result.created_pops} POPs and{' '}
          {result.created_agents} agents.
        </Alert>
      )}
      {agentResult && (
        <Alert severity="success" sx={{ mb: 2 }}>
          Agent login import completed: created {agentResult.created_rows}, updated{" "}
          {agentResult.updated_rows}, skipped {agentResult.skipped_rows} accounts.
        </Alert>
      )}
      <Card>
        <CardContent>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12 }}>
              <TextField
                select
                fullWidth
                label="Import Type"
                value={importType}
                onChange={(e) => {
                  setImportType(e.target.value as "customers" | "agent-users");
                  setFile(null);
                  setPreview(null);
                  setAgentPreview(null);
                  setConfirmed(false);
                  setError("");
                  setResult(null);
                  setAgentResult(null);
                }}
              >
                <MenuItem value="customers">Customers, Packages, POPs &amp; Subscriptions</MenuItem>
                <MenuItem value="agent-users">Agent Login Users</MenuItem>
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 8 }}>
              <Button
                component="label"
                variant="outlined"
                startIcon={<UploadFileIcon />}
              >
                {file?.name || (importType === "agent-users" ? "Select Agent login Excel file" : "Select Customer CSV or Excel file")}
                <input
                  hidden
                  type="file"
                  accept={importType === "agent-users" ? ".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" : ".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}
                  onChange={(e) => {
                    setFile(e.target.files?.[0] ?? null);
                    setPreview(null);
                    setAgentPreview(null);
                    setConfirmed(false);
                    setResult(null);
                    setAgentResult(null);
                  }}
                />
              </Button>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <Button
                fullWidth
                variant="contained"
                disabled={!file || busy}
                onClick={() => void inspect()}
              >
                {busy ? "Checking..." : "Preview File"}
              </Button>
            </Grid>
          </Grid>
          {preview && (
            <Box sx={{ mt: 3 }}>
              <Grid container spacing={2}>
                {[
                  ["Rows", preview.total_rows],
                  ["Active source users", preview.active_rows],
                  ["Inactive source users", preview.inactive_rows],
                  ["New packages", preview.packages.length],
                  ["New POPs", preview.pops.length],
                ].map(([l, v]) => (
                  <Grid key={String(l)} size={{ xs: 6, md: 2.4 }}>
                    <Card variant="outlined">
                      <CardContent>
                        <Typography color="text.secondary">{l}</Typography>
                        <Typography variant="h6" sx={{ fontWeight: 700 }}>
                          {v}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
              {preview.warnings.map((w) => (
                <Alert key={w} severity="warning" sx={{ mt: 2 }}>
                  {w}
                </Alert>
              ))}
              <TextField
                select
                fullWidth
                label="Target MikroTik Router"
                value={routerID || ""}
                onChange={(e) => setRouterID(Number(e.target.value))}
                sx={{ mt: 2 }}
              >
                {routers.data
                  ?.filter((r) => r.status === "ACTIVE")
                  .map((r) => (
                    <MenuItem key={r.id} value={r.id}>
                      {r.code} — {r.name}
                    </MenuItem>
                  ))}
              </TextField>
              <FormControlLabel
                sx={{ mt: 2 }}
                control={
                  <Checkbox
                    checked={confirmed}
                    onChange={(e) => setConfirmed(e.target.checked)}
                  />
                }
                label="I verified the preview and understand source-active subscriptions will be imported ACTIVE."
              />
              <Button
                fullWidth
                variant="contained"
                color="warning"
                disabled={!routerID || !confirmed || busy}
                onClick={() => void execute()}
                sx={{ mt: 2 }}
              >
                {busy ? "Importing..." : "Confirm Transactional Import"}
              </Button>
            </Box>
          )}
          {agentPreview && (
            <Box sx={{ mt: 3 }}>
              <Grid container spacing={2}>
                {[
                  ["Source rows", agentPreview.total_rows],
                  ["Ready", agentPreview.ready_rows],
                  ["New accounts", agentPreview.create_rows],
                  ["Updates", agentPreview.update_rows],
                  ["Skipped", agentPreview.skipped_rows],
                ].map(([label, value]) => (
                  <Grid key={String(label)} size={{ xs: 6, md: 2.4 }}>
                    <Card variant="outlined"><CardContent><Typography color="text.secondary">{label}</Typography><Typography variant="h6" sx={{ fontWeight: 700 }}>{value}</Typography></CardContent></Card>
                  </Grid>
                ))}
              </Grid>
              {agentPreview.warnings.map((warning) => <Alert key={warning} severity="warning" sx={{ mt: 2 }}>{warning}</Alert>)}
              <TableContainer sx={{ mt: 2 }}>
                <Table size="small" sx={{ minWidth: 760 }}>
                  <TableHead><TableRow><TableCell>#</TableCell><TableCell>Name</TableCell><TableCell>Username</TableCell><TableCell>Role</TableCell><TableCell>Linked Agent</TableCell><TableCell>Status</TableCell></TableRow></TableHead>
                  <TableBody>{agentPreview.rows.map((row, index) => <TableRow key={`${row.row_number}-${row.username}`}><TableCell>{index + 1}</TableCell><TableCell>{row.name}</TableCell><TableCell>{row.username}</TableCell><TableCell sx={{ textTransform: "capitalize" }}>{row.role}</TableCell><TableCell>{row.agent_name || "—"}</TableCell><TableCell>{row.status.replaceAll("_", " ")}</TableCell></TableRow>)}</TableBody>
                </Table>
              </TableContainer>
              <FormControlLabel
                sx={{ mt: 2 }}
                control={<Checkbox checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} />}
                label="I verified this preview and understand existing usernames will have their passwords reset from this workbook."
              />
              <Button fullWidth variant="contained" color="warning" disabled={!confirmed || busy || agentPreview.ready_rows === 0} onClick={() => void execute()} sx={{ mt: 2 }}>
                {busy ? "Importing..." : "Confirm Agent Login Import"}
              </Button>
            </Box>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
