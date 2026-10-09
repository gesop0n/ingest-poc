import z from "zod";

// ブラウザに埋め込む値 (VITE_*) だけを定義する
// INGESTPOC_* は vite.config.ts の toolEnvSchema で扱う
export const envSchema = z.object({
  // VITE_APP_VERSION: z.string().min(1),
});
