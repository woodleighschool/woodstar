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
    title: "Run Woodstar",
    text: "Start Woodstar and PostgreSQL with Docker Compose.",
    to: "/docs/getting-started/docker-compose",
  },
  {
    title: "How Woodstar fits together",
    text: "Learn how hosts, labels, the web app, and agent protocols relate.",
    to: "/docs/concepts/capability-boundaries",
  },
  {
    title: "Agent protocols",
    text: "Configure Orbit, osquery, Santa, and Munki clients.",
    to: "/docs/agent-protocols/overview",
  },
];

const secondaryLinks: LinkItem[] = [
  {
    title: "Configuration",
    text: "Find every WOODSTAR_ setting and its default.",
    to: "/docs/configuration/environment",
  },
  {
    title: "API reference",
    text: "Browse the API generated from Woodstar's OpenAPI schema.",
    to: "/docs/api/overview",
  },
  {
    title: "Development",
    text: "Set up a checkout, then build and test the repository.",
    to: "/docs/development/setup",
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
      description="Documentation for the Woodstar macOS management server."
    >
      <main className={styles.main}>
        <section className={styles.intro}>
          <img src="/img/woodstar.svg" className={styles.logo} alt="Woodstar" />
          <div>
            <p className={styles.kicker}>Documentation</p>
            <h1>Woodstar</h1>
            <p className={styles.summary}>
              Self-hosted macOS management for the gaps Intune leaves, with Munki, Santa, Orbit, and
              osquery.
            </p>
            <div className={styles.actions}>
              <Link className={styles.primaryAction} to="/docs/intro">
                Read the introduction
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
