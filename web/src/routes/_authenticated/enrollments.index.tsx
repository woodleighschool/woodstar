/* eslint-disable @typescript-eslint/only-throw-error -- tanstack/react-router uses thrown redirect() as control-flow */
import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/enrollments/")({
  beforeLoad: () => {
    throw redirect({ to: "/enrollments/orbit" });
  },
});
