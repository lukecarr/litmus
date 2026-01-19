// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
  integrations: [
    starlight({
      title: "Litmus",
      description:
        "Specification testing for structured LLM outputs. Compare accuracy, latency, and throughput across models.",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/lukecarr/litmus",
        },
      ],
      sidebar: [
        {
          label: "Getting Started",
          autogenerate: { directory: "getting-started" },
        },
        {
          label: "Usage",
          autogenerate: { directory: "usage" },
        },
        {
          label: "Output",
          autogenerate: { directory: "output" },
        },
      ],
    }),
  ],
});
