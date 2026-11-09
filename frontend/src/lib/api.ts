const API_URL = (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").replace(/\/$/, "");

type Envelope<T> = {
  data: T;
  meta?: { total: number; limit: number; page: number };
};

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string
  ) {
    super(message);
    this.name = "APIError";
  }
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  retry = true,
  includeMeta = false
): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_URL}/api/v1${path}`, {
    ...init,
    headers,
    credentials: "include",
    cache: "no-store",
  });

  if (
    response.status === 401 &&
    retry &&
    path !== "/auth/login" &&
    path !== "/auth/register" &&
    path !== "/auth/refresh"
  ) {
    const refreshed = await fetch(`${API_URL}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      cache: "no-store",
    });
    if (refreshed.ok) {
      return request<T>(path, init, false);
    }
  }

  if (!response.ok) {
    let message = "Terjadi kesalahan. Coba lagi.";
    let code: string | undefined;
    try {
      const body = (await response.json()) as { error?: { message?: string; code?: string } };
      message = body.error?.message || message;
      code = body.error?.code;
    } catch {
      /* Non-JSON response */
    }
    throw new APIError(message, response.status, code);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const type = response.headers.get("content-type") || "";
  if (!type.includes("application/json")) {
    return (await response.text()) as T;
  }

  const body = (await response.json()) as Envelope<T>;
  return (includeMeta ? body : body.data) as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  getPaged: <T>(path: string) => request<Envelope<T[]>>(path, {}, true, true),
  post: <T>(path: string, body?: unknown, headers?: HeadersInit) =>
    request<T>(path, {
      method: "POST",
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  patch: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "PATCH", body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  async download(path: string, filename: string) {
    let response = await fetch(`${API_URL}/api/v1${path}`, {
      credentials: "include",
      cache: "no-store",
    });
    if (response.status === 401) {
      const refreshed = await fetch(`${API_URL}/api/v1/auth/refresh`, {
        method: "POST",
        credentials: "include",
        cache: "no-store",
      });
      if (refreshed.ok) {
        response = await fetch(`${API_URL}/api/v1${path}`, {
          credentials: "include",
          cache: "no-store",
        });
      }
    }
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as {
        error?: { message?: string };
      } | null;
      throw new APIError(body?.error?.message || "Ekspor gagal.", response.status);
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
    URL.revokeObjectURL(url);
  },
  async streamPRD(
    payload: {
      projectId: string;
      sessionId?: string;
      idempotencyKey?: string;
      provider?: string;
      model?: string;
      idea?: Idea;
    },
    onEvent: (event: StreamPRDEvent) => void,
    signal?: AbortSignal
  ): Promise<PRD> {
    const headers = new Headers({
      "Content-Type": "application/json",
      Accept: "text/event-stream",
    });
    if (payload.idempotencyKey) {
      headers.set("Idempotency-Key", payload.idempotencyKey);
    }

    let response = await fetch(`${API_URL}/api/v1/ai/generate-prd/stream`, {
      method: "POST",
      headers,
      body: JSON.stringify(payload),
      credentials: "include",
      cache: "no-store",
      signal,
    });

    if (response.status === 401) {
      const refreshed = await fetch(`${API_URL}/api/v1/auth/refresh`, {
        method: "POST",
        credentials: "include",
        cache: "no-store",
      });
      if (refreshed.ok) {
        response = await fetch(`${API_URL}/api/v1/ai/generate-prd/stream`, {
          method: "POST",
          headers,
          body: JSON.stringify(payload),
          credentials: "include",
          cache: "no-store",
          signal,
        });
      }
    }

    if (!response.ok) {
      let message = "Gagal memulai streaming PRD.";
      let code: string | undefined;
      try {
        const body = (await response.json()) as { error?: { message?: string; code?: string } };
        message = body.error?.message || message;
        code = body.error?.code;
      } catch {
        /* Non-JSON response */
      }
      throw new APIError(message, response.status, code);
    }

    if (!response.body) {
      throw new APIError("Response body kosong.", response.status);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let finalPRD: PRD | null = null;

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        const parts = buffer.split("\n\n");
        buffer = parts.pop() || "";

        for (const block of parts) {
          if (!block.trim()) continue;
          let eventType = "message";
          let dataText = "";

          const lines = block.split("\n");
          for (const line of lines) {
            const trimmed = line.trim();
            if (trimmed.startsWith("event:")) {
              eventType = trimmed.slice(6).trim();
            } else if (trimmed.startsWith("data:")) {
              dataText += trimmed.slice(5).trim();
            }
          }

          if (!dataText) continue;

          try {
            const parsed = JSON.parse(dataText);
            if (eventType === "progress") {
              onEvent({ type: "progress", step: parsed.step, message: parsed.message });
            } else if (eventType === "chunk") {
              onEvent({ type: "chunk", chunk: parsed.chunk });
            } else if (eventType === "done") {
              finalPRD = parsed.data as PRD;
              onEvent({ type: "done", prd: finalPRD });
            } else if (eventType === "error") {
              onEvent({ type: "error", message: parsed.message, code: parsed.code });
              throw new APIError(parsed.message || "Gagal menyusun PRD.", 500, parsed.code);
            }
          } catch (e) {
            if (e instanceof APIError) throw e;
            console.error("SSE parse error:", e);
          }
        }
      }
    } finally {
      reader.releaseLock();
    }

    if (!finalPRD) {
      throw new APIError("Sesi streaming selesai tanpa data PRD lengkap.", 500);
    }

    return finalPRD;
  },
  getAIProviders: () => request<AIProviderInfo[]>("/ai/providers"),
  getAISettings: () => request<UserAISettings>("/settings/ai"),
  updateAISettings: (settings: Partial<UserAISettings>) =>
    request<UserAISettings>("/settings/ai", { method: "PUT", body: JSON.stringify(settings) }),
  testAISettings: (settings: Partial<UserAISettings>) =>
    request<{ status: string; message: string }>("/settings/ai/test", {
      method: "POST",
      body: JSON.stringify(settings),
    }),
};

export type StreamPRDEvent =
  | { type: "progress"; step: string; message: string }
  | { type: "chunk"; chunk: string }
  | { type: "done"; prd: PRD }
  | { type: "error"; message: string; code?: string };

export type AIModelInfo = {
  id: string;
  name: string;
  description: string;
  isDefault: boolean;
};

export type AIProviderInfo = {
  id: string;
  name: string;
  description: string;
  defaultBaseUrl: string;
  requiresApiKey: boolean;
  supportsCustom: boolean;
  defaultModel: string;
  models: AIModelInfo[];
};

export type UserAISettings = {
  userId: string;
  provider: string;
  model: string;
  baseUrl?: string;
  apiKey?: string;
  customModel?: string;
  createdAt: string;
  updatedAt: string;
};

export type User = {
  id: string;
  email: string;
  name: string;
};

export type Project = {
  id: string;
  name: string;
  description: string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type Requirement = {
  id: string;
  description: string;
  priority: string;
  acceptanceCriteria: string[];
};

export type PRDContent = Record<string, unknown> & {
  executiveSummary?: string;
  productOverview?: string;
  problemStatement?: string;
  visionAndGoals?: string[];
  targetAudience?: string[];
  useCases?: string[];
  userStories?: Requirement[];
  functionalRequirements?: Requirement[];
  nonFunctionalRequirements?: Requirement[];
  featureSpecifications?: string[];
  userJourney?: string[];
  scope?: string[];
  outOfScope?: string[];
  businessRules?: string[];
  dataRequirements?: string[];
  apiRequirements?: string[];
  architecture?: string;
  securityAndPrivacy?: string[];
  successMetrics?: string[];
  dependenciesAndConstraints?: string[];
  risks?: string[];
  milestones?: string[];
  openQuestions?: string[];
  assumptions?: string[];
  customSections?: Array<{ title: string; body: string }>;
};

export type PRD = {
  id: string;
  projectId: string;
  title: string;
  status: string;
  content: PRDContent;
  version: number;
  createdAt: string;
  updatedAt: string;
};

export type Page<T> = {
  items: T[];
  total: number;
  limit: number;
  page: number;
};

export type Idea = {
  productName: string;
  description: string;
  problem: string;
  targetUsers: string;
  businessGoals: string;
  platform: string;
  productType: string;
  features: string[];
  technology: string;
  constraints: string;
  language: string;
  detailLevel: string;
  clarifications?: Record<string, string>;
};
