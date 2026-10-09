import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";
import z from "zod";
import { envSchema } from "./src/env.schema.ts";

const port = z.coerce.number().int().min(1).max(65535);

const toolEnvSchema = z.object({
  INGESTPOC_SERVER_PORT: port.default(8080),
  INGESTPOC_FRONTEND_PORT: port.default(4040),
});

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // ブラウザ向けの値は、起動・ビルドの時点で検証して止める
  envSchema.parse(loadEnv(mode, process.cwd(), "VITE_"));
  const toolEnv = toolEnvSchema.parse(process.env);

  return {
    plugins: [react()],

    resolve: {
      tsconfigPaths: true,
    },

    server: {
      port: toolEnv.INGESTPOC_FRONTEND_PORT,
      // `make dev-backend` で起動した API へ転送する
      proxy: {
        "/api": `http://localhost:${toolEnv.INGESTPOC_SERVER_PORT}`,
      },
    },
  };
});
