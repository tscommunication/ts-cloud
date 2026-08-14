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

export default function CustomerImport() {
  const routers = useQuery({
    queryKey: ["network-routers"],
    queryFn: getNetworkRouters,
  });
  const [file, setFile] = useState<File | null>(null),
    [preview, setPreview] = useState<CustomerImportPreview | null>(null),
    [routerID, setRouterID] = useState(0),
    [confirmed, setConfirmed] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [result, setResult] = useState<CustomerImportBatch | null>(null);
  const inspect = async () => {
    if (!file) return;
    try {
      setBusy(true);
      setError("");
      setResult(null);
      setPreview(await previewCustomerImport(file));
    } catch (e) {
      setError(getAPIErrorMessage(e, "CSV preview failed."));
    } finally {
      setBusy(false);
    }
  };
  const execute = async () => {
    if (!file || !routerID || !confirmed) return;
    try {
      setBusy(true);
      setError("");
      setResult(await importCustomers(file, routerID));
      setPreview(null);
    } catch (e) {
      setError(
        getAPIErrorMessage(e, "CSV import failed; no partial rows were saved."),
      );
    } finally {
      setBusy(false);
    }
  };
  return (
    <Box>
      <Typography variant="h4" sx={{ fontWeight: 700 }}>
        Customer CSV Import
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Preview and safely import customers, POPs, packages and subscriptions
        from the supported CSV export.
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
      <Card>
        <CardContent>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 8 }}>
              <Button
                component="label"
                variant="outlined"
                startIcon={<UploadFileIcon />}
              >
                {file?.name || "Select CSV file"}
                <input
                  hidden
                  type="file"
                  accept=".csv,text/csv"
                  onChange={(e) => {
                    setFile(e.target.files?.[0] ?? null);
                    setPreview(null);
                    setConfirmed(false);
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
                {busy ? "Checking..." : "Preview CSV"}
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
        </CardContent>
      </Card>
    </Box>
  );
}
