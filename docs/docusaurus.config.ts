import type * as Preset from "@docusaurus/preset-classic";
import type { Config } from "@docusaurus/types";
import type { PrismTheme } from "prism-react-renderer";

const lightCodeTheme: PrismTheme = {
  plain: {
    color: "oklch(0.25 0.006 80)",
    backgroundColor: "oklch(0.925 0.009 82)",
  },
  styles: [
    { types: ["comment", "prolog", "doctype", "cdata"], style: { color: "#6c756e" } },
    { types: ["punctuation"], style: { color: "#59635c" } },
    {
      types: ["property", "tag", "boolean", "number", "constant", "symbol"],
      style: { color: "#23694d" },
    },
    { types: ["selector", "attr-name", "string", "char", "builtin"], style: { color: "#526b2d" } },
    { types: ["operator", "entity", "url"], style: { color: "#59635c" } },
    { types: ["atrule", "attr-value", "keyword"], style: { color: "#715b20" } },
    { types: ["function", "class-name"], style: { color: "#9a3b2f" } },
    { types: ["regex", "important", "variable"], style: { color: "#8a5a1c" } },
  ],
};

const darkCodeTheme: PrismTheme = {
  plain: {
    color: "oklch(0.9 0.006 82)",
    backgroundColor: "oklch(0.295 0.006 80)",
  },
  styles: [
    { types: ["comment", "prolog", "doctype", "cdata"], style: { color: "#8b938d" } },
    { types: ["punctuation"], style: { color: "#b7beb8" } },
    {
      types: ["property", "tag", "boolean", "number", "constant", "symbol"],
      style: { color: "#82caa2" },
    },
    { types: ["selector", "attr-name", "string", "char", "builtin"], style: { color: "#bfd483" } },
    { types: ["operator", "entity", "url"], style: { color: "#b7beb8" } },
    { types: ["atrule", "attr-value", "keyword"], style: { color: "#dfc66b" } },
    { types: ["function", "class-name"], style: { color: "#e99482" } },
    { types: ["regex", "important", "variable"], style: { color: "#e2af63" } },
  ],
};

const config: Config = {
  title: "Woodstar",
  tagline: "Self-hosted macOS management: Munki, Santa, and osquery.",
  favicon: "img/favicon.png",

  future: {
    v4: true,
  },

  url: process.env.DOCS_URL ?? "https://woodstar.docs.localhost",
  baseUrl: process.env.DOCS_BASE_URL ?? "/",

  organizationName: "woodleighschool",
  projectName: "woodstar",

  onBrokenLinks: "throw",

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: "warn",
    },
  },

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  themes: [
    [
      "@easyops-cn/docusaurus-search-local",
      {
        hashed: true,
        indexBlog: false,
        docsRouteBasePath: "/docs",
        docsDir: "content",
        language: "en",
        searchBarShortcutHint: false,
      },
    ],
    "docusaurus-theme-openapi-docs",
  ],

  plugins: [
    [
      "docusaurus-plugin-llms",
      {
        docsDir: "content",
        generateLLMsTxt: true,
        generateLLMsFullTxt: true,
        includeBlog: false,
        excludeImports: true,
        removeDuplicateHeadings: true,
      },
    ],
    [
      "docusaurus-plugin-openapi-docs",
      {
        id: "api",
        docsPluginId: "classic",
        config: {
          woodstar: {
            specPath: "../web/openapi.yaml",
            outputDir: "content/api",
            sidebarOptions: {
              groupPathsBy: "tagGroup",
            },
            hideSendButton: true,
          },
        },
      },
    ],
  ],

  presets: [
    [
      "classic",
      {
        docs: {
          path: "content",
          sidebarPath: "./sidebars.ts",
          routeBasePath: "docs",
          editUrl: "https://github.com/woodleighschool/woodstar/tree/main/docs/",
          docItemComponent: "@theme/ApiItem",
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: "img/apple-touch-icon.png",
    colorMode: {
      defaultMode: "dark",
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: "Woodstar",
      logo: {
        alt: "Woodstar",
        src: "img/woodstar.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "docsSidebar",
          position: "left",
          label: "Docs",
        },
        {
          type: "docSidebar",
          sidebarId: "apiSidebar",
          position: "left",
          label: "API",
        },
        {
          href: "https://github.com/woodleighschool/woodstar",
          position: "right",
          label: "GitHub",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Docs",
          items: [
            { label: "Overview", to: "/docs/intro" },
            { label: "Local development", to: "/docs/getting-started/local-development" },
            { label: "Configuration", to: "/docs/configuration/environment" },
          ],
        },
        {
          title: "Reference",
          items: [
            { label: "Agent protocols", to: "/docs/agent-protocols/overview" },
            { label: "Admin API", to: "/docs/api/overview" },
            { label: "Development", to: "/docs/development/commands" },
          ],
        },
        {
          title: "Project",
          items: [{ label: "Repository", href: "https://github.com/woodleighschool/woodstar" }],
        },
      ],
      copyright: `Copyright ${new Date().getFullYear()} Woodleigh School`,
    },
    prism: {
      theme: lightCodeTheme,
      darkTheme: darkCodeTheme,
      additionalLanguages: ["bash", "go", "json", "sql", "toml", "yaml"],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
