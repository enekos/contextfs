import { GoogleGenAI } from "@google/genai";
import { config } from "../core/config";

const MAX_RETRIES = 3;
const RETRY_DELAY_MS = 1000;

export function getLLM(): GoogleGenAI | null {
  return config.geminiApiKey ? new GoogleGenAI({ apiKey: config.geminiApiKey }) : null;
}

export async function llmGenerate(prompt: string, attempt = 1): Promise<string> {
  const ai = getLLM();
  if (!ai) throw new Error("Gemini API key not configured");
  try {
    const response = await ai.models.generateContent({ model: config.llmModel, contents: prompt });
    return response.text?.trim() || "";
  } catch (error: unknown) {
    const status = (error as { status?: number })?.status;
    const msg = (error as { message?: string })?.message;
    if (attempt < MAX_RETRIES && (status === 429 || (status ?? 0) >= 500 || msg?.includes("fetch failed"))) {
      const delay = RETRY_DELAY_MS * Math.pow(2, attempt - 1);
      console.warn(`[llm] API error (${msg}), retrying in ${delay}ms (${attempt + 1}/${MAX_RETRIES})`);
      await new Promise((r) => setTimeout(r, delay));
      return llmGenerate(prompt, attempt + 1);
    }
    throw error;
  }
}
