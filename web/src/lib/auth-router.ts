import { redirect } from "@tanstack/react-router";

import { apiClient } from "@/lib/api";

export async function requireSetup() {
  const result = await apiClient.GET("/api/setup/status");
  if (result.error || !result.data?.complete) {
    throw redirect({ to: "/setup" });
  }
}

export async function redirectForEntry() {
  const setup = await apiClient.GET("/api/setup/status");
  if (setup.error || !setup.data?.complete) {
    throw redirect({ to: "/setup" });
  }

  const me = await apiClient.GET("/api/auth/me");
  if (me.response.status === 401) {
    throw redirect({ to: "/login" });
  }
  throw redirect({ to: "/hosts" });
}

export async function requireUser() {
  await requireSetup();

  const me = await apiClient.GET("/api/auth/me");
  if (me.response.status === 401) {
    throw redirect({ to: "/login" });
  }
  if (me.error || !me.data) {
    throw redirect({ to: "/login" });
  }
}

// Viewers are bounced to /hosts; non-authenticated users to /login (via requireUser).
export async function requireAdmin() {
  await requireSetup();

  const me = await apiClient.GET("/api/auth/me");
  if (me.response.status === 401 || me.error || !me.data) {
    throw redirect({ to: "/login" });
  }
  if (me.data.role !== "admin") {
    throw redirect({ to: "/hosts" });
  }
}

export async function redirectAuthenticatedFromLogin() {
  const setup = await apiClient.GET("/api/setup/status");
  if (setup.error || !setup.data?.complete) {
    throw redirect({ to: "/setup" });
  }

  const me = await apiClient.GET("/api/auth/me");
  if (me.data) {
    throw redirect({ to: "/hosts" });
  }
}

export async function redirectCompletedSetup() {
  const setup = await apiClient.GET("/api/setup/status");
  if (!setup.data?.complete) {
    return;
  }

  const me = await apiClient.GET("/api/auth/me");
  if (me.data) {
    throw redirect({ to: "/hosts" });
  }
  throw redirect({ to: "/login" });
}
