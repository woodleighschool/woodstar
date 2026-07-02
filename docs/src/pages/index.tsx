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
    text: "Bring up Woodstar, Postgres, and the frontend from a checkout with the repo's mise tasks.",
    to: "/docs/getting-started/local-development",
  },
  {
    title: "How it fits together",
    text: "Hosts, labels, the four agents, and where the admin app ends and the agent protocols begin.",
    to: "/docs/concepts/capability-boundaries",
  },
  {
    title: "Agent protocols",
    text: "How Orbit, osquery, Santa, and Munki clients enroll, authenticate, and sync.",
    to: "/docs/agent-protocols/overview",
  },
];

const secondaryLinks: LinkItem[] = [
  {
    title: "Configuration",
    text: "The WOODSTAR_ settings, what's required, and which features switch on once you set them.",
    to: "/docs/configuration/environment",
  },
  {
    title: "API reference",
    text: "Every admin endpoint, generated from the OpenAPI spec the server ships.",
    to: "/docs/api/overview",
  },
  {
    title: "Development",
    text: "Build, test, generate, and format with the mise tasks.",
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
      description="Self-hosted macOS management: Munki, Santa, and osquery."
    >
      <main className={styles.main}>
        <section className={styles.intro}>
          <img src="/img/woodstar.svg" className={styles.logo} alt="Woodstar" />
          <div>
            <p className={styles.kicker}>Woodstar docs</p>
            <h1>Self-hosted macOS management: Munki, Santa, and osquery.</h1>
            <p className={styles.summary}>
              Woodstar runs Munki, Santa, and osquery for a fleet of Macs: managed software,
              execution policy, and inventory. It covers the macOS gaps left by our move from Jamf
              to Intune.
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
