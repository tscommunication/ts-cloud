import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Container,
  Divider,
  Grid,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useNavigate } from "react-router-dom";

import { logout } from "../../api/auth";
import { getAPIErrorMessage } from "../../api/errors";
import {
  getCustomerPortalInvoices,
  getCustomerPortalMe,
  getCustomerPortalPayments,
  getCustomerPortalSubscriptions,
  type CustomerPortalInvoice,
  type CustomerPortalMe,
  type CustomerPortalPayment,
  type CustomerPortalSubscription,
} from "../../api/customerPortal";

function money(value: number) {
  return `\u09F3${Number(value || 0).toLocaleString("en-BD", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
}

function displayValue(value?: string | number | null) {
  if (value === undefined || value === null || value === "") {
    return "—";
  }

  return String(value);
}

function statusColor(
  status?: string,
): "default" | "success" | "warning" | "error" | "info" {
  switch ((status || "").toUpperCase()) {
    case "ACTIVE":
    case "PAID":
    case "COMPLETED":
      return "success";
    case "PARTIAL":
    case "PENDING":
      return "warning";
    case "EXPIRED":
    case "DISCONNECTED":
    case "VOID":
    case "FAILED":
      return "error";
    default:
      return "default";
  }
}

export default function SelfCareHome() {
  const navigate = useNavigate();

  const [me, setMe] = useState<CustomerPortalMe | null>(null);
  const [subscriptions, setSubscriptions] = useState<
    CustomerPortalSubscription[]
  >([]);
  const [invoices, setInvoices] = useState<CustomerPortalInvoice[]>([]);
  const [payments, setPayments] = useState<CustomerPortalPayment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    async function loadPortal() {
      setLoading(true);
      setError("");

      try {
        const [meData, subscriptionData, invoiceData, paymentData] =
          await Promise.all([
            getCustomerPortalMe(),
            getCustomerPortalSubscriptions(),
            getCustomerPortalInvoices(),
            getCustomerPortalPayments(),
          ]);

        if (!active) {
          return;
        }

        setMe(meData);
        setSubscriptions(subscriptionData);
        setInvoices(invoiceData);
        setPayments(paymentData);
      } catch (error: unknown) {
        if (!active) {
          return;
        }

        setError(
          getAPIErrorMessage(
            error,
            "Unable to load your SelfCare information.",
          ),
        );
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void loadPortal();

    return () => {
      active = false;
    };
  }, []);

  const totalDue = useMemo(
    () =>
      invoices.reduce(
        (sum, invoice) => sum + Number(invoice.due_amount || 0),
        0,
      ),
    [invoices],
  );

  const totalPaid = useMemo(
    () =>
      payments
        .filter((payment) => payment.status.toUpperCase() !== "VOID")
        .reduce((sum, payment) => sum + Number(payment.amount || 0), 0),
    [payments],
  );

  const primarySubscription = subscriptions[0] || null;

  function handleLogout() {
    logout();
    navigate("/selfcare/login", {
      replace: true,
    });
  }

  if (loading) {
    return (
      <Box
        sx={{
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: "#f5f7fb",
        }}
      >
        <Stack spacing={2} sx={{ alignItems: "center" }}>
          <CircularProgress />
          <Typography color="text.secondary">Loading SelfCare...</Typography>
        </Stack>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        minHeight: "100vh",
        backgroundColor: "#f5f7fb",
        py: 4,
      }}
    >
      <Container maxWidth="xl">
        <Stack spacing={3}>
          <Card sx={{ borderRadius: 3 }}>
            <CardContent sx={{ p: 3 }}>
              <Stack
                direction={{
                  xs: "column",
                  sm: "row",
                }}
                spacing={2}
                sx={{
                  justifyContent: "space-between",
                  alignItems: {
                    xs: "flex-start",
                    sm: "center",
                  },
                }}
              >
                <Box>
                  <Typography variant="h4" sx={{ fontWeight: 700 }}>
                    TS-Cloud SelfCare
                  </Typography>

                  <Typography color="text.secondary" sx={{ mt: 0.5 }}>
                    Welcome, {me?.full_name || "Customer"}
                  </Typography>
                </Box>

                <Button variant="outlined" onClick={handleLogout}>
                  Sign Out
                </Button>
              </Stack>
            </CardContent>
          </Card>

          {error && <Alert severity="error">{error}</Alert>}

          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ height: "100%" }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary">
                    Account Status
                  </Typography>

                  <Box sx={{ mt: 2 }}>
                    <Chip
                      label={
                        me?.status || primarySubscription?.status || "UNKNOWN"
                      }
                      color={statusColor(
                        me?.status || primarySubscription?.status,
                      )}
                    />
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ height: "100%" }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary">
                    Current Due
                  </Typography>

                  <Typography
                    variant="h5"
                    sx={{
                      mt: 1.5,
                      fontWeight: 700,
                    }}
                  >
                    {money(totalDue)}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ height: "100%" }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary">
                    Total Paid
                  </Typography>

                  <Typography
                    variant="h5"
                    sx={{
                      mt: 1.5,
                      fontWeight: 700,
                    }}
                  >
                    {money(totalPaid)}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ height: "100%" }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary">
                    Next Billing Date
                  </Typography>

                  <Typography
                    variant="h6"
                    sx={{
                      mt: 1.5,
                      fontWeight: 700,
                    }}
                  >
                    {displayValue(primarySubscription?.next_billing_date)}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          <Card sx={{ borderRadius: 3 }}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Customer Profile
              </Typography>

              <Divider sx={{ my: 2 }} />

              <Grid container spacing={2}>
                <Grid size={{ xs: 12, md: 4 }}>
                  <Typography variant="caption" color="text.secondary">
                    Customer Code
                  </Typography>
                  <Typography>{displayValue(me?.customer_code)}</Typography>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Typography variant="caption" color="text.secondary">
                    Full Name
                  </Typography>
                  <Typography>{displayValue(me?.full_name)}</Typography>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Typography variant="caption" color="text.secondary">
                    Mobile
                  </Typography>
                  <Typography>{displayValue(me?.mobile)}</Typography>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Typography variant="caption" color="text.secondary">
                    Email
                  </Typography>
                  <Typography>{displayValue(me?.email)}</Typography>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Typography variant="caption" color="text.secondary">
                    Joining Date
                  </Typography>
                  <Typography>{displayValue(me?.joining_date)}</Typography>
                </Grid>

                <Grid size={{ xs: 12, md: 4 }}>
                  <Typography variant="caption" color="text.secondary">
                    Billing Day
                  </Typography>
                  <Typography>{displayValue(me?.billing_day)}</Typography>
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <Typography variant="caption" color="text.secondary">
                    Present Address
                  </Typography>
                  <Typography>{displayValue(me?.present_address)}</Typography>
                </Grid>
              </Grid>
            </CardContent>
          </Card>

          <Card sx={{ borderRadius: 3 }}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Subscriptions
              </Typography>

              <Divider sx={{ my: 2 }} />

              {subscriptions.length === 0 ? (
                <Typography color="text.secondary">
                  No subscription found.
                </Typography>
              ) : (
                <Grid container spacing={2}>
                  {subscriptions.map((subscription) => (
                    <Grid
                      key={subscription.id}
                      size={{
                        xs: 12,
                        md: 6,
                      }}
                    >
                      <Card variant="outlined" sx={{ height: "100%" }}>
                        <CardContent>
                          <Stack
                            direction="row"
                            spacing={2}
                            sx={{
                              justifyContent: "space-between",
                            }}
                          >
                            <Box>
                              <Typography
                                variant="subtitle1"
                                sx={{
                                  fontWeight: 700,
                                }}
                              >
                                {displayValue(subscription.subscription_code)}
                              </Typography>

                              <Typography
                                variant="body2"
                                color="text.secondary"
                              >
                                PPPoE:{" "}
                                {displayValue(subscription.pppoe_username)}
                              </Typography>
                            </Box>

                            <Chip
                              size="small"
                              label={subscription.status}
                              color={statusColor(subscription.status)}
                            />
                          </Stack>

                          <Divider sx={{ my: 2 }} />

                          <Stack spacing={1}>
                            <Typography variant="body2">
                              Package ID: {subscription.package_id}
                            </Typography>

                            <Typography variant="body2">
                              Activation:{" "}
                              {displayValue(subscription.activation_date)}
                            </Typography>

                            <Typography variant="body2">
                              Next Billing:{" "}
                              {displayValue(subscription.next_billing_date)}
                            </Typography>

                            <Typography variant="body2">
                              Expiry: {displayValue(subscription.expiry_date)}
                            </Typography>

                            <Typography variant="body2">
                              Due: {money(subscription.due_amount)}
                            </Typography>
                          </Stack>
                        </CardContent>
                      </Card>
                    </Grid>
                  ))}
                </Grid>
              )}
            </CardContent>
          </Card>

          <Card sx={{ borderRadius: 3 }}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Invoice History
              </Typography>

              <Divider sx={{ my: 2 }} />

              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Invoice</TableCell>
                      <TableCell>Issue Date</TableCell>
                      <TableCell>Due Date</TableCell>
                      <TableCell align="right">Total</TableCell>
                      <TableCell align="right">Paid</TableCell>
                      <TableCell align="right">Due</TableCell>
                      <TableCell>Status</TableCell>
                    </TableRow>
                  </TableHead>

                  <TableBody>
                    {invoices.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={7} align="center">
                          No invoice found.
                        </TableCell>
                      </TableRow>
                    ) : (
                      invoices.map((invoice) => (
                        <TableRow key={invoice.id}>
                          <TableCell>{invoice.invoice_no}</TableCell>
                          <TableCell>
                            {displayValue(invoice.issue_date)}
                          </TableCell>
                          <TableCell>
                            {displayValue(invoice.due_date)}
                          </TableCell>
                          <TableCell align="right">
                            {money(invoice.total_amount)}
                          </TableCell>
                          <TableCell align="right">
                            {money(invoice.paid_amount)}
                          </TableCell>
                          <TableCell align="right">
                            {money(invoice.due_amount)}
                          </TableCell>
                          <TableCell>
                            <Chip
                              size="small"
                              label={invoice.status}
                              color={statusColor(invoice.status)}
                            />
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>

          <Card sx={{ borderRadius: 3 }}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Payment History
              </Typography>

              <Divider sx={{ my: 2 }} />

              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Receipt</TableCell>
                      <TableCell>Date</TableCell>
                      <TableCell>Method</TableCell>
                      <TableCell>Transaction ID</TableCell>
                      <TableCell align="right">Amount</TableCell>
                      <TableCell>Status</TableCell>
                    </TableRow>
                  </TableHead>

                  <TableBody>
                    {payments.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={6} align="center">
                          No payment found.
                        </TableCell>
                      </TableRow>
                    ) : (
                      payments.map((payment) => (
                        <TableRow key={payment.id}>
                          <TableCell>{payment.receipt_no}</TableCell>
                          <TableCell>
                            {displayValue(payment.payment_date)}
                          </TableCell>
                          <TableCell>{displayValue(payment.method)}</TableCell>
                          <TableCell>
                            {displayValue(payment.transaction_id)}
                          </TableCell>
                          <TableCell align="right">
                            {money(payment.amount)}
                          </TableCell>
                          <TableCell>
                            <Chip
                              size="small"
                              label={payment.status}
                              color={statusColor(payment.status)}
                            />
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Stack>
      </Container>
    </Box>
  );
}
