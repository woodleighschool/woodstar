import Link from "@docusaurus/Link";
import Layout from "@theme/Layout";
import type { ReactNode } from "react";

import styles from "./index.module.css";

type LinkItem = {
  title: string;
  text: string;
  to: string;
};

const primaryLinks: LinkItem[] = [
  {
    title: "Run it locally",
    text: "Use the repo's mise tasks with Postgres and the checked-in compose stack.",
    to: "/docs/getting-started/local-development",
  },
  {
    title: "Read the architecture",
    text: "Capability boundaries, agent protocols, admin API ownership, and runtime wiring.",
    to: "/docs/concepts/capability-boundaries",
  },
  {
    title: "Check the protocol surface",
    text: "Orbit, osquery, Santa, and Munki endpoints are documented from the code that mounts them.",
    to: "/docs/agent-protocols/overview",
  },
];

const secondaryLinks: LinkItem[] = [
  {
    title: "Environment",
    text: "The current `WOODSTAR_` settings, defaults, and capability gates.",
    to: "/docs/configuration/environment",
  },
  {
    title: "Admin API",
    text: "Huma-backed JSON routes under `/api`, plus the generated OpenAPI command.",
    to: "/docs/api/overview",
  },
  {
    title: "Development",
    text: "Build, test, generation, and formatting tasks from the repository contract.",
    to: "/docs/development/commands",
  },
];

function LinkPanel({ title, text, to }: LinkItem) {
  return (
    <Link className={styles.panel} to={to}>
      <h2>{title}</h2>
      <p>{text}</p>
    </Link>
  );
}

export default function Home(): ReactNode {
  return (
    <Layout
      title="Woodstar documentation"
      description="Woodstar macOS observability and admin server documentation"
    >
      <main className={styles.main}>
        <section className={styles.intro}>
          <img src="/img/woodstar.svg" className={styles.logo} />
          <div>
            <p className={styles.kicker}>Woodstar docs</p>
            <h1>macOS fleet admin notes, tied to the code that exists now.</h1>
            <p className={styles.summary}>
              Woodstar is a self-hosted server for Orbit/osquery first, with Santa and
              Munki modules growing alongside the admin UI. These docs avoid release
              promises. They explain what the current tree wires, what is inferred from
              code, and where an operator still needs local knowledge.
            </p>
            <div className={styles.actions}>
              <Link className={styles.primaryAction} to="/docs/intro">
                Start reading
              </Link>
              <Link className={styles.secondaryAction} to="/docs/development/commands">
                Developer commands
              </Link>
            </div>
          </div>
        </section>

        <section className={styles.grid}>
          {primaryLinks.map((item) => (
            <LinkPanel key={item.title} {...item} />
          ))}
        </section>

        <section className={styles.compactGrid}>
          {secondaryLinks.map((item) => (
            <LinkPanel key={item.title} {...item} />
          ))}
        </section>
      </main>
    </Layout>
  );
}
