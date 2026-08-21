import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import axios from "axios";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
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
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import RefreshIcon from "@mui/icons-material/Refresh";
import SearchIcon from "@mui/icons-material/Search";

import { getStoredUser } from "../../api/auth";
import {
  createUser,
  deleteUser,
  getUsers,
  updateUser,
  type User,
} from "../../api/users";
import { getAgents, type Agent } from "../../api/distribution";

interface UserForm {
  name: string;
  username: string;
  email: string;
  password: string;
  role: "admin" | "agent" | "user" | "superadmin";
  agent_id: number | "";
  active: boolean;
}
const initialForm = (): UserForm => ({
  name: "",
  username: "",
  email: "",
  password: "",
  role: "admin",
  active: true,
  agent_id: "",
});
const errorMessage = (error: unknown, fallback: string) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error || fallback;
  }
  return fallback;
};

function Users() {
  const signedInUser = getStoredUser();
  const isSuperadmin = signedInUser?.role === "superadmin";
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [search, setSearch] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState<UserForm>(initialForm);
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);
  const [agents, setAgents] = useState<Agent[]>([]);

  const loadUsers = async () => {
    try {
      setLoading(true);
      setError("");
      const data = await getUsers();
      setUsers(data.users);
    } catch (err: unknown) {
      setError(errorMessage(err, "Failed to load users."));
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadUsers();
  }, []);
  useEffect(() => {
    const load = async () => {
      try {
        setAgents(await getAgents());
      } catch {
        setAgents([]);
      }
    };
    void load();
  }, []);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    return query
      ? users.filter((item) =>
          [
            item.name,
            item.username,
            item.email,
            item.role,
            item.active ? "active" : "disabled",
          ]
            .join(" ")
            .toLowerCase()
            .includes(query),
        )
      : users;
  }, [users, search]);

  const openCreate = () => {
    setEditing(null);
    setForm(initialForm());
    setError("");
    setSuccess("");
    setOpen(true);
  };
  const openEdit = (item: User) => {
    setEditing(item);
    setForm({
      name: item.name,
      username: item.username,
      email: item.email,
      password: "",
      role: item.role as UserForm["role"],
      active: item.active,
      agent_id: item.agent_id ?? "",
    });
    setError("");
    setSuccess("");
    setOpen(true);
  };
  const change = <K extends keyof UserForm>(key: K, value: UserForm[K]) =>
    setForm((current) => ({ ...current, [key]: value }));

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!editing && form.password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    if (editing && form.password && form.password.length < 8) {
      setError("New password must be at least 8 characters.");
      return;
    }
    if (form.role === "agent" && !form.agent_id) {
      setError("Please select a linked agent.");
      return;
    }
    try {
      setBusy(true);
      setError("");
      setSuccess("");
      if (editing) {
        await updateUser(editing.id, {
          name: form.name.trim(),
          username: form.username.trim(),
          email: form.email.trim(),
          ...(isSuperadmin ? { active: form.active } : {}),
          ...(isSuperadmin && editing.id !== signedInUser?.id
            ? { role: form.role }
            : {}),
          ...(isSuperadmin && form.role === "agent"
            ? { agent_id: Number(form.agent_id) }
            : {}),
          ...(form.password ? { password: form.password } : {}),
        });
      } else {
        await createUser({
          name: form.name.trim(),
          username: form.username.trim(),
          email: form.email.trim(),
          password: form.password,
          role: form.role,
          ...(form.role === "agent" ? { agent_id: Number(form.agent_id) } : {}),
        });
      }
      setOpen(false);
      setSuccess(
        editing ? "User updated successfully." : "User created successfully.",
      );
      await loadUsers();
    } catch (err: unknown) {
      setError(errorMessage(err, "Failed to save user."));
    } finally {
      setBusy(false);
    }
  };
  const remove = async () => {
    if (!deleteTarget) return;
    try {
      setBusy(true);
      setError("");
      setSuccess("");
      await deleteUser(deleteTarget.id);
      setDeleteTarget(null);
      setSuccess("User deleted successfully.");
      await loadUsers();
    } catch (err: unknown) {
      setError(errorMessage(err, "Failed to delete user."));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Box>
      <Box
        sx={{
          display: "flex",
          flexDirection: { xs: "column", sm: "row" },
          justifyContent: "space-between",
          gap: 2,
          mb: 3,
        }}
      >
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Users ({users.length})
          </Typography>
          <Typography color="text.secondary">
            Manage administrator and staff accounts.
          </Typography>
        </Box>
        {isSuperadmin && (
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={openCreate}
          >
            Add User
          </Button>
        )}
      </Box>
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError("")}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert severity="success" sx={{ mb: 3 }} onClose={() => setSuccess("")}>
          {success}
        </Alert>
      )}
      <Card>
        <CardContent>
          <Box
            sx={{
              display: "flex",
              gap: 2,
              justifyContent: "space-between",
              mb: 2,
            }}
          >
            <TextField
              size="small"
              placeholder="Search users..."
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              sx={{ maxWidth: 400, width: "100%" }}
              slotProps={{
                input: { startAdornment: <SearchIcon sx={{ mr: 1 }} /> },
              }}
            />
            <IconButton onClick={() => void loadUsers()} disabled={loading}>
              <RefreshIcon />
            </IconButton>
          </Box>
          {loading ? (
            <Box sx={{ py: 8, textAlign: "center" }}>
              <CircularProgress />
            </Box>
          ) : filtered.length === 0 ? (
            <Typography
              sx={{ py: 8, textAlign: "center" }}
              color="text.secondary"
            >
              No users found
            </Typography>
          ) : (
            <TableContainer>
              <Table sx={{ minWidth: 840 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>#</TableCell>
                    <TableCell>Name</TableCell>
                    <TableCell>Username</TableCell>
                    <TableCell>Email</TableCell>
                    <TableCell>Role</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filtered.map((item, index) => (
                    <TableRow key={item.id} hover>
                      <TableCell>{index + 1}</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>
                        {item.name}
                      </TableCell>
                      <TableCell>{item.username}</TableCell>
                      <TableCell>{item.email}</TableCell>
                      <TableCell sx={{ textTransform: "capitalize" }}>
                        {item.role}
                      </TableCell>
                      <TableCell
                        sx={{
                          color: item.active ? "success.main" : "warning.main",
                          fontWeight: 600,
                        }}
                      >
                        {item.active ? "ACTIVE" : "DISABLED"}
                      </TableCell>
                      <TableCell align="right" sx={{ whiteSpace: "nowrap" }}>
                        <IconButton
                          color="primary"
                          onClick={() => openEdit(item)}
                        >
                          <EditIcon />
                        </IconButton>
                        {isSuperadmin &&
                          item.id !== signedInUser?.id &&
                          item.role !== "superadmin" && (
                            <IconButton
                              color="error"
                              onClick={() => setDeleteTarget(item)}
                            >
                              <DeleteIcon />
                            </IconButton>
                          )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={open}
        onClose={() => !busy && setOpen(false)}
        fullWidth
        maxWidth="sm"
      >
        <Box component="form" onSubmit={submit}>
          <DialogTitle>{editing ? "Edit User" : "Add User"}</DialogTitle>
          <DialogContent dividers>
            <Grid container spacing={2} sx={{ pt: 1 }}>
              <Grid size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  required
                  label="Name"
                  value={form.name}
                  onChange={(event) => change("name", event.target.value)}
                />
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  required
                  label="Username"
                  value={form.username}
                  onChange={(event) => change("username", event.target.value)}
                  autoComplete="username"
                />
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  required
                  type="email"
                  label="Email"
                  value={form.email}
                  onChange={(event) => change("email", event.target.value)}
                />
              </Grid>
              <Grid size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  required={!editing}
                  type="password"
                  label={editing ? "New Password" : "Password"}
                  value={form.password}
                  onChange={(event) => change("password", event.target.value)}
                  helperText={
                    editing
                      ? "Leave blank to keep the current password; minimum 8 characters"
                      : "Minimum 8 characters"
                  }
                  autoComplete="new-password"
                />
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  fullWidth
                  select
                  label="Role"
                  value={form.role}
                  disabled={!isSuperadmin || editing?.id === signedInUser?.id}
                  onChange={(event) => {
                    const role = event.target.value as UserForm["role"];

                    setForm((current) => ({
                      ...current,
                      role,
                      agent_id: role === "agent" ? current.agent_id : "",
                    }));
                  }}
                >
                  <MenuItem value="superadmin">Superadmin</MenuItem>
                  <MenuItem value="admin">Admin</MenuItem>
                  <MenuItem value="agent">Agent</MenuItem>
                  <MenuItem value="user">User</MenuItem>
                </TextField>
              </Grid>
              {form.role === "agent" && (
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    required
                    select
                    label="Linked Agent"
                    value={form.agent_id}
                    onChange={(event) =>
                      change("agent_id", Number(event.target.value))
                    }
                  >
                    {agents.map((agent) => (
                      <MenuItem key={agent.id} value={agent.id}>
                        {agent.code} — {agent.name}
                      </MenuItem>
                    ))}
                  </TextField>
                </Grid>
              )}

              {editing && (
                <Grid size={{ xs: 12, md: 6 }}>
                  <TextField
                    fullWidth
                    select
                    label="Status"
                    value={form.active ? "active" : "disabled"}
                    disabled={!isSuperadmin || editing.id === signedInUser?.id}
                    onChange={(event) =>
                      change("active", event.target.value === "active")
                    }
                  >
                    <MenuItem value="active">ACTIVE</MenuItem>
                    <MenuItem value="disabled">DISABLED</MenuItem>
                  </TextField>
                </Grid>
              )}
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="contained"
              disabled={
                busy ||
                !form.name.trim() ||
                !form.username.trim() ||
                !form.email.trim()
              }
            >
              {busy ? "Saving..." : editing ? "Update User" : "Create User"}
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog
        open={Boolean(deleteTarget)}
        onClose={() => !busy && setDeleteTarget(null)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>Delete User</DialogTitle>
        <DialogContent>
          <Typography>
            Delete <strong>{deleteTarget?.username}</strong>? This action cannot
            be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)} disabled={busy}>
            Cancel
          </Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => void remove()}
            disabled={busy}
          >
            {busy ? "Deleting..." : "Delete"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export default Users;
