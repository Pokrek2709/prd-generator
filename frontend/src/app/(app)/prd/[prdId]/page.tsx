"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import ReactMarkdown from "react-markdown";
import { MermaidChart } from "@/components/mermaid";
import {
  ArrowLeft,
  Check,
  ChevronDown,
  Clock3,
  Copy,
  Download,
  FileText,
  LoaderCircle,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Share2,
  Sparkles,
  WandSparkles,
  AlertTriangle,
} from "lucide-react";
import { api, type PRD, type PRDContent } from "@/lib/api";
import { Badge, Button, Card, Input, Label, Select, Textarea } from "@/components/ui";
import { useToast } from "@/components/toast";
import { formatDate } from "@/lib/utils";

type Version = {
  id: string;
  version: number;
  title: string;
  status: string;
  content: PRDContent;
  createdAt: string;
};

type Field = {
  key: keyof PRDContent;
  label: string;
  kind: "text" | "list" | "requirements";
};

type Requirement = {
  id: string;
  description: string;
  priority: string;
  acceptanceCriteria: string[];
};

const fields: Field[] = [
  { key: "executiveSummary", label: "Ringkasan eksekutif", kind: "text" },
  { key: "productOverview", label: "Gambaran produk", kind: "text" },
  { key: "problemStatement", label: "Pernyataan masalah", kind: "text" },
  { key: "visionAndGoals", label: "Visi dan tujuan", kind: "list" },
  { key: "targetAudience", label: "Target pengguna", kind: "list" },
  { key: "useCases", label: "Use cases", kind: "list" },
  { key: "userStories", label: "User stories", kind: "requirements" },
  { key: "functionalRequirements", label: "Kebutuhan fungsional", kind: "requirements" },
  { key: "nonFunctionalRequirements", label: "Kebutuhan non-fungsional", kind: "requirements" },
  { key: "featureSpecifications", label: "Spesifikasi fitur", kind: "list" },
  { key: "userJourney", label: "Perjalanan pengguna", kind: "list" },
  { key: "scope", label: "Cakupan", kind: "list" },
  { key: "outOfScope", label: "Di luar cakupan", kind: "list" },
  { key: "businessRules", label: "Aturan bisnis", kind: "list" },
  { key: "dataRequirements", label: "Kebutuhan data", kind: "list" },
  { key: "apiRequirements", label: "Kebutuhan API", kind: "list" },
  { key: "architecture", label: "Arsitektur tingkat tinggi", kind: "text" },
  { key: "securityAndPrivacy", label: "Keamanan dan privasi", kind: "list" },
  { key: "successMetrics", label: "Metrik keberhasilan", kind: "list" },
  { key: "dependenciesAndConstraints", label: "Dependensi dan batasan", kind: "list" },
  { key: "risks", label: "Risiko dan mitigasi", kind: "list" },
  { key: "milestones", label: "Milestone dan roadmap", kind: "list" },
  { key: "openQuestions", label: "Pertanyaan terbuka", kind: "list" },
  { key: "assumptions", label: "Asumsi", kind: "list" },
];

const actions = [
  ["improve_prd", "Perbaiki PRD"],
  ["find_missing_requirements", "Cari kebutuhan yang hilang"],
  ["detect_ambiguity", "Deteksi ambiguitas"],
  ["generate_user_stories", "Buat user stories"],
  ["generate_acceptance_criteria", "Buat acceptance criteria"],
  ["identify_edge_cases", "Identifikasi edge cases"],
  ["review_security", "Tinjau keamanan"],
  ["generate_tasks", "Buat development tasks"],
  ["create_mvp_scope", "Buat scope MVP"],
  ["estimate_complexity", "Estimasi kompleksitas"],
  ["draft_api_contract", "Buat draft kontrak API"],
  ["draft_database_schema", "Buat draft skema database"],
  ["simplify_language", "Sederhanakan bahasa"],
  ["translate_document", "Terjemahkan dokumen"],
] as const;

function valueText(value: unknown, kind: Field["kind"]): string {
  if (Array.isArray(value)) {
    return value
      .map((item) =>
        kind === "requirements" && item && typeof item === "object"
          ? (item as { description?: string }).description || ""
          : String(item)
      )
      .filter(Boolean)
      .join("\n");
  }
  return typeof value === "string" ? value : "";
}

function contentMarkdown(title: string, content: PRDContent) {
  const core = fields
    .map((field) => {
      const value = content[field.key];
      if (!value || (Array.isArray(value) && value.length === 0)) return "";
      const text = Array.isArray(value)
        ? value
            .map((item) => {
              if (field.kind === "requirements" && item && typeof item === "object") {
                const req = item as Requirement;
                const criteria = req.acceptanceCriteria?.length
                  ? `\n${req.acceptanceCriteria.map((c) => `  - [ ] ${c}`).join("\n")}`
                  : "";
                return `- **${req.id}** [${req.priority || "P1"}]: ${req.description}${criteria}`;
              }
              return `- ${String(item)}`;
            })
            .join("\n")
        : String(value);
      return `## ${field.label}\n\n${text}`;
    })
    .filter(Boolean)
    .join("\n\n");

  const extra = Array.isArray(content.customSections)
    ? (content.customSections as Array<{ title: string; body: string }>)
        .map((section) => `## ${section.title}\n\n${section.body || ""}`)
        .join("\n\n")
    : "";

  return `# ${title}\n\n${[core, extra].filter(Boolean).join("\n\n")}`;
}

function RequirementEditor({
  items,
  onChange,
}: {
  items: Requirement[];
  onChange: (items: Requirement[]) => void;
}) {
  return (
    <div className="space-y-3">
      {items.map((item, index) => (
        <div key={`${item.id}-${index}`} className="rounded-xl border border-[#edf0ea] bg-[#fbfcf9] p-3.5">
          <div className="flex flex-wrap items-center gap-2">
            <Input
              aria-label="ID requirement"
              value={item.id}
              onChange={(e) =>
                onChange(items.map((req, i) => (i === index ? { ...req, id: e.target.value } : req)))
              }
              className="h-9 w-28 text-xs font-mono font-medium"
            />
            <Select
              aria-label="Prioritas"
              value={item.priority}
              onChange={(e) =>
                onChange(items.map((req, i) => (i === index ? { ...req, priority: e.target.value } : req)))
              }
              className="h-9 w-24 text-xs font-medium"
            >
              <option value="P0">P0 (MVP)</option>
              <option value="P1">P1 (Tinggi)</option>
              <option value="P2">P2 (Rendah)</option>
            </Select>
            <button
              onClick={() => onChange(items.filter((_, i) => i !== index))}
              className="ml-auto px-2 py-1 text-xs text-red-600 hover:underline"
            >
              Hapus
            </button>
          </div>
          <Textarea
            aria-label="Deskripsi requirement"
            value={item.description}
            onChange={(e) =>
              onChange(items.map((req, i) => (i === index ? { ...req, description: e.target.value } : req)))
            }
            placeholder="Deskripsi kebutuhan (misal: Pengguna dapat login via Google OAuth)..."
            className="mt-2 min-h-16 text-xs"
          />
          <Label className="mb-1 mt-3 text-[11px] font-semibold text-[#6d7b6e]">
            Acceptance criteria · satu per baris (Given / When / Then)
          </Label>
          <Textarea
            aria-label="Acceptance criteria"
            value={item.acceptanceCriteria?.join("\n") || ""}
            onChange={(e) =>
              onChange(
                items.map((req, i) =>
                  i === index
                    ? {
                        ...req,
                        acceptanceCriteria: e.target.value
                          .split("\n")
                          .map((line) => line.trim())
                          .filter(Boolean),
                      }
                    : req
                )
              )
            }
            placeholder="Given pengguna di halaman login&#10;When klik tombol Google&#10;Then dialihkan ke consent screen"
            className="min-h-16 text-xs font-mono"
          />
        </div>
      ))}
      <Button
        type="button"
        variant="secondary"
        size="sm"
        onClick={() =>
          onChange([
            ...items,
            {
              id: `REQ-${String(items.length + 1).padStart(3, "0")}`,
              description: "",
              priority: "P1",
              acceptanceCriteria: [],
            },
          ])
        }
      >
        <Plus size={13} /> Tambah requirement
      </Button>
    </div>
  );
}

export default function PRDEditorPage() {
  const { prdId } = useParams<{ prdId: string }>();
  const router = useRouter();
  const toast = useToast();

  const [prd, setPrd] = useState<PRD | null>(null);
  const [content, setContent] = useState<PRDContent>({});
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("draft");
  const [mode, setMode] = useState<"edit" | "preview">("edit");
  const [saveState, setSaveState] = useState<"saved" | "saving" | "error">("saved");
  const [error, setError] = useState("");
  const [activeField, setActiveField] = useState<keyof PRDContent>("executiveSummary");
  const [versions, setVersions] = useState<Version[]>([]);
  const [showVersions, setShowVersions] = useState(false);
  const [compareVersion, setCompareVersion] = useState<Version | null>(null);
  const [aiResult, setAiResult] = useState<any>(null);
  const [aiBusy, setAiBusy] = useState(false);
  const [shareUrl, setShareUrl] = useState("");
  const [shareId, setShareId] = useState("");
  const [showActions, setShowActions] = useState(false);
  const [showMore, setShowMore] = useState(false);

  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveQueue = useRef<Promise<void>>(Promise.resolve());
  const latest = useRef<{ title: string; content: PRDContent; status: string; version: number } | null>(null);
  const desired = useRef<{ title: string; content: PRDContent; status: string } | null>(null);
  const initialized = useRef(false);

  desired.current = { title, content, status };

  const markdown = useMemo(() => contentMarkdown(title, content), [title, content]);

  const load = useCallback(async () => {
    setError("");
    try {
      const item = await api.get<PRD>(`/prds/${prdId}`);
      setPrd(item);
      setTitle(item.title);
      setStatus(item.status);
      setContent(item.content || {});
      latest.current = {
        title: item.title,
        content: item.content || {},
        status: item.status,
        version: item.version,
      };
      initialized.current = true;
    } catch (e) {
      setError(e instanceof Error ? e.message : "PRD tidak dapat dimuat.");
    }
  }, [prdId]);

  useEffect(() => {
    void load();
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [load]);

  const save = useCallback(
    (snapshot: { title: string; content: PRDContent; status: string }) => {
      const task = saveQueue.current.catch(() => undefined).then(async () => {
        const persisted = latest.current;
        if (!persisted) return;
        if (
          persisted.title === snapshot.title &&
          persisted.status === snapshot.status &&
          JSON.stringify(persisted.content) === JSON.stringify(snapshot.content)
        ) {
          if (
            desired.current?.title === snapshot.title &&
            desired.current?.status === snapshot.status &&
            JSON.stringify(desired.current?.content) === JSON.stringify(snapshot.content)
          ) setSaveState("saved");
          return;
        }

        setSaveState("saving");
        setError("");
        const updated = await api.patch<PRD>(`/prds/${prdId}`, {
          title: snapshot.title,
          status: snapshot.status,
          content: snapshot.content,
          version: persisted.version,
        });
        setPrd(updated);
        latest.current = {
          title: updated.title,
          content: updated.content,
          status: updated.status,
          version: updated.version,
        };
        const currentDesired = desired.current;
        setSaveState(
          currentDesired && currentDesired.title === updated.title && currentDesired.status === updated.status &&
          JSON.stringify(currentDesired.content) === JSON.stringify(updated.content) ? "saved" : "saving"
        );
      });
      saveQueue.current = task;
      return task.catch((e: unknown) => {
        setSaveState("error");
        setError(e instanceof Error ? e.message : "Perubahan belum tersimpan.");
        throw e;
      });
    },
    [prdId]
  );

  useEffect(() => {
    if (!initialized.current || !latest.current) return;
    if (timer.current) clearTimeout(timer.current);

    timer.current = setTimeout(() => {
      const current = latest.current;
      if (!current) return;
      if (
        current.title === title &&
        JSON.stringify(current.content) === JSON.stringify(content) &&
        current.status === status
      ) {
        return;
      }
      void save({ title, content, status }).catch(() => undefined);
    }, 850);

    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [title, content, status, save]);

  function updateField(field: Field, value: string) {
    setContent((prev) => {
      let updated: unknown;
      if (field.kind === "list") {
        updated = value.split("\n").map((x) => x.trim()).filter(Boolean);
      } else if (field.kind === "requirements") {
        const old = Array.isArray(prev[field.key])
          ? (prev[field.key] as Array<{ id?: string; priority?: string; acceptanceCriteria?: string[] }>)
          : [];
        updated = value
          .split("\n")
          .map((line) => line.trim())
          .filter(Boolean)
          .map((description, index) => ({
            id: old[index]?.id || `REQ-${String(index + 1).padStart(3, "0")}`,
            priority: old[index]?.priority || "P1",
            acceptanceCriteria: old[index]?.acceptanceCriteria || [],
            description,
          }));
      } else {
        updated = value;
      }
      return { ...prev, [field.key]: updated } as PRDContent;
    });
  }

  async function runAction(action: string) {
    if (!prd) return;
    setAiBusy(true);
    setAiResult(null);
    setError("");
    try {
      const result = await api.post<unknown>("/ai/actions", {
        action,
        prdId: prd.id,
        input: { section: activeField },
      });
      setAiResult(result);
      setShowActions(false);
      toast.success("Analisis AI selesai. Tinjau usulan di bawah.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Aksi AI gagal dijalankan.");
      toast.error("Gagal menjalankan aksi AI.");
    } finally {
      setAiBusy(false);
    }
  }

  async function regenerate() {
    if (!prd) return;
    setAiBusy(true);
    setError("");
    try {
      const result = await api.post<unknown>("/ai/regenerate-section", {
        prdId: prd.id,
        input: { section: activeField, currentContent: content[activeField] },
      });
      setAiResult(result);
      toast.success(`Draf baru untuk ${String(activeField)} siap ditinjau.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Section belum dapat dibuat ulang.");
      toast.error("Gagal membuat ulang section.");
    } finally {
      setAiBusy(false);
    }
  }

  function applySuggestion() {
    if (!aiResult) return;

    let payload = aiResult;
    if (typeof aiResult === "object" && aiResult !== null) {
      if ("result" in aiResult) payload = aiResult.result;
      else if ("sectionContent" in aiResult) payload = aiResult.sectionContent;
      else if ("content" in aiResult) payload = aiResult.content;
    }

    const currentFieldDef = fields.find((item) => item.key === activeField);

    if (typeof payload === "string") {
      if (currentFieldDef) updateField(currentFieldDef, payload);
      else setContent((prev) => ({ ...prev, [activeField]: payload }));
    } else if (Array.isArray(payload)) {
      if (currentFieldDef?.kind === "requirements") {
        const formatted: Requirement[] = payload.map((item, idx) => {
          if (typeof item === "object" && item !== null) {
            return {
              id: item.id || `REQ-${String(idx + 1).padStart(3, "0")}`,
              description: item.description || item.title || String(item),
              priority: item.priority || "P1",
              acceptanceCriteria: Array.isArray(item.acceptanceCriteria) ? item.acceptanceCriteria : [],
            };
          }
          return {
            id: `REQ-${String(idx + 1).padStart(3, "0")}`,
            description: String(item),
            priority: "P1",
            acceptanceCriteria: [],
          };
        });
        setContent((prev) => ({ ...prev, [activeField]: formatted }));
      } else if (currentFieldDef?.kind === "list") {
        const stringList = payload.map((x) => (typeof x === "object" ? JSON.stringify(x) : String(x)));
        setContent((prev) => ({ ...prev, [activeField]: stringList }));
      } else {
        const joined = payload.map((x) => (typeof x === "object" ? JSON.stringify(x) : String(x))).join("\n");
        setContent((prev) => ({ ...prev, [activeField]: joined }));
      }
    } else if (typeof payload === "object" && payload !== null) {
      const allowedKeys = new Set(fields.map((f) => f.key as string));
      const cleanPatch: Record<string, unknown> = {};
      for (const [k, v] of Object.entries(payload)) {
        if (allowedKeys.has(k)) {
          cleanPatch[k] = v;
        }
      }
      if (Object.keys(cleanPatch).length > 0) {
        setContent((prev) => ({ ...prev, ...cleanPatch }));
      } else {
        setContent((prev) => ({ ...prev, [activeField]: JSON.stringify(payload, null, 2) }));
      }
    }

    setAiResult(null);
    toast.success("Usulan AI berhasil diterapkan pada dokumen.");
  }

  async function loadVersions() {
    setShowVersions((v) => !v);
    if (versions.length === 0) {
      try {
        const list = await api.get<Version[]>(`/prds/${prdId}/versions`);
        setVersions(list || []);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Riwayat versi belum tersedia.");
      }
    }
  }

  async function restoreVersion(version: Version) {
    if (!confirm(`Pulihkan versi ${version.version}? Konten saat ini akan disimpan sebagai riwayat versi baru.`)) {
      return;
    }
    try {
      const updated = await api.post<PRD>(`/prds/${prdId}/restore/${version.id}`);
      setPrd(updated);
      setTitle(updated.title);
      setStatus(updated.status);
      setContent(updated.content);
      latest.current = {
        title: updated.title,
        content: updated.content,
        status: updated.status,
        version: updated.version,
      };
      setSaveState("saved");
      setShowVersions(false);
      toast.success(`Versi ${version.version} berhasil dipulihkan.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Versi tidak dapat dipulihkan.");
      toast.error("Gagal memulihkan versi.");
    }
  }

  async function share() {
    try {
      const result = await api.post<{ url: string; id: string }>(`/prds/${prdId}/share-links`, {
        expiresInHours: 168,
      });
      const token = result.url.split("/").filter(Boolean).at(-1);
      setShareUrl(`${window.location.origin}/shared/${token}`);
      setShareId(result.id);
      toast.success("Tautan berbagi publik aktif dibuat.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Link berbagi belum dapat dibuat.");
    }
  }

  async function revokeShare() {
    if (!shareId) return;
    try {
      await api.delete(`/prds/${prdId}/share-links/${shareId}`);
      setShareId("");
      setShareUrl("");
      toast.info("Tautan berbagi berhasil dicabut.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Link berbagi belum dapat dicabut.");
    }
  }

  async function duplicatePRD() {
    try {
      const copy = await api.post<PRD>(`/prds/${prdId}/duplicate`);
      toast.success("Dokumen berhasil diduplikasi.");
      router.push(`/prd/${copy.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "PRD tidak dapat diduplikasi.");
    }
  }

  async function deletePRD() {
    if (!prd || !confirm("Hapus dokumen PRD ini? Tindakan ini tidak dapat dibatalkan.")) return;
    try {
      await api.delete(`/prds/${prdId}`);
      toast.info("Dokumen PRD telah dihapus.");
      router.push(`/projects/${prd.projectId}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "PRD tidak dapat dihapus.");
    }
  }

  async function exportAs(format: "markdown" | "pdf" | "docx" | "jira" | "linear") {
    const base = title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "prd";
    try {
      if (timer.current) {
        clearTimeout(timer.current);
        timer.current = null;
      }
      await save({ title, content, status });
      const filename =
        format === "jira"
          ? `${base}-jira.csv`
          : format === "linear"
          ? `${base}-linear.csv`
          : `${base}.${format === "markdown" ? "md" : format}`;
      await api.download(`/prds/${prdId}/export/${format}`, filename);
      toast.success(`Berhasil mengunduh dokumen ${format.toUpperCase()}.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ekspor gagal.");
      toast.error("Gagal mengunduh berkas.");
    }
  }

  if (!prd && !error) {
    return <div className="h-64 animate-pulse rounded-2xl bg-white" />;
  }

  if (!prd) {
    return (
      <Card className="mx-auto max-w-xl p-8 text-center">
        <h1 className="text-lg font-semibold">PRD tidak tersedia</h1>
        <p className="mt-2 text-sm text-red-700">{error}</p>
        <Link href="/dashboard">
          <Button className="mt-5" variant="secondary">
            Kembali ke dashboard
          </Button>
        </Link>
      </Card>
    );
  }

  return (
    <div className="-mx-5 -mt-7 min-h-[calc(100vh-68px)] bg-[#fbfbf8] md:-mx-9 md:-mt-9">
      {/* Top action bar */}
      <div className="sticky top-[68px] z-10 flex flex-wrap items-center justify-between gap-3 border-b border-[#e7ebe4] bg-[#fbfbf8]/95 px-4 py-3 backdrop-blur md:px-7">
        <div className="flex min-w-0 items-center gap-3">
          <Link
            href={`/projects/${prd.projectId}`}
            className="text-[#8b968a] hover:text-[#315642]"
            aria-label="Kembali ke proyek"
          >
            <ArrowLeft size={17} />
          </Link>
          <div className="min-w-0">
            <div className="truncate text-[10px] text-[#9ba49b]">
              Workspace / {title || "Dokumen PRD"}
            </div>
            <div className="mt-0.5 flex items-center gap-1.5 text-[10px] text-[#8d988d]">
              {saveState === "saving" ? (
                <>
                  <LoaderCircle className="animate-spin text-[#315642]" size={11} /> Menyimpan…
                </>
              ) : saveState === "error" ? (
                <span className="flex items-center gap-1 text-red-600">
                  <AlertTriangle size={11} /> Belum tersimpan
                </span>
              ) : (
                <>
                  <Check size={11} className="text-[#628066]" /> Tersimpan
                </>
              )}
              <span>·</span> v{prd.version}
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Select
            aria-label="Status dokumen"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="h-9 w-auto min-w-28 text-xs font-medium"
          >
            <option value="draft">Draft</option>
            <option value="in_progress">In progress</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </Select>

          <div className="flex rounded-xl border border-[#e2e8df] bg-white p-1">
            <button
              onClick={() => setMode("edit")}
              className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
                mode === "edit" ? "bg-[#e9efe6] text-[#385b40]" : "text-[#869187] hover:text-[#2d4734]"
              }`}
            >
              Edit
            </button>
            <button
              onClick={() => setMode("preview")}
              className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
                mode === "preview" ? "bg-[#e9efe6] text-[#385b40]" : "text-[#869187] hover:text-[#2d4734]"
              }`}
            >
              Preview
            </button>
          </div>

          <Button variant="secondary" size="sm" onClick={loadVersions}>
            <Clock3 size={14} /> <span className="hidden sm:inline">Riwayat</span>
          </Button>

          {/* AI Tools dropdown */}
          <div className="relative">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowActions(!showActions)}
              disabled={aiBusy}
            >
              <Sparkles size={14} className="text-[#55785c]" />
              <span className="hidden sm:inline">AI tools</span>
              <ChevronDown size={12} />
            </Button>
            {showActions && (
              <div className="absolute right-0 top-11 z-30 max-h-80 w-64 overflow-y-auto rounded-xl border border-[#e5e9e1] bg-white p-1.5 shadow-xl">
                <div className="px-2 py-1.5 text-[10px] font-bold uppercase tracking-wider text-[#9aa49a]">
                  Aksi Dokumen
                </div>
                {actions.map(([key, label]) => (
                  <button
                    key={key}
                    onClick={() => void runAction(key)}
                    className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#4b5b4e] hover:bg-[#f2f5ef] hover:text-[#2d4e36]"
                  >
                    {label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Share button */}
          <div className="relative">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShareUrl(shareUrl ? "" : "share")}
              aria-label="Bagikan tautan"
            >
              <Share2 size={14} />
            </Button>
            {shareUrl === "share" && (
              <div className="absolute right-0 top-11 z-30 w-72 rounded-xl border border-[#e5e9e1] bg-white p-4 shadow-xl">
                <p className="text-xs font-semibold">Tautan berbagi dokumen</p>
                <p className="mt-1 text-[11px] text-[#8d988d]">
                  Tautan aktif selama 7 hari dan dapat dicabut sewaktu-waktu.
                </p>
                <Button size="sm" className="mt-3 w-full" onClick={() => void share()}>
                  <Plus size={13} /> Buat link berbagi
                </Button>
              </div>
            )}
          </div>

          {/* Export dropdown */}
          <div className="relative group">
            <Button variant="secondary" size="sm">
              <Download size={14} />
              <span className="hidden sm:inline">Ekspor</span>
            </Button>
            <div className="invisible absolute right-0 top-10 z-20 w-44 rounded-xl border border-[#e5e9e1] bg-white p-1.5 opacity-0 shadow-xl transition group-hover:visible group-hover:opacity-100 group-focus-within:visible group-focus-within:opacity-100">
              <button
                onClick={() => void exportAs("markdown")}
                className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
              >
                Markdown (.md)
              </button>
              <button
                onClick={() => void exportAs("pdf")}
                className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
              >
                PDF Document (.pdf)
              </button>
              <button
                onClick={() => void exportAs("docx")}
                className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
              >
                Word Document (.docx)
              </button>
              <button
                onClick={() => void exportAs("jira")}
                className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
              >
                Jira CSV (.csv)
              </button>
              <button
                onClick={() => void exportAs("linear")}
                className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
              >
                Linear CSV (.csv)
              </button>
              <a
                href={`/prds/${prd.id}/print`}
                target="_blank"
                rel="noreferrer"
                className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
              >
                Print view
              </a>
            </div>
          </div>

          {/* More options */}
          <div className="relative">
            <Button
              variant="ghost"
              size="sm"
              aria-label="Aksi lainnya"
              onClick={() => setShowMore(!showMore)}
            >
              <MoreHorizontal size={16} />
            </Button>
            {showMore && (
              <div className="absolute right-0 top-10 z-30 w-40 rounded-xl border border-[#e5e9e1] bg-white p-1 shadow-xl">
                <button
                  onClick={() => {
                    setShowMore(false);
                    void duplicatePRD();
                  }}
                  className="block w-full rounded-lg px-3 py-2 text-left text-xs text-[#566558] hover:bg-[#f2f5ef]"
                >
                  Duplikat PRD
                </button>
                <button
                  onClick={() => {
                    setShowMore(false);
                    void deletePRD();
                  }}
                  className="block w-full rounded-lg px-3 py-2 text-left text-xs text-red-600 hover:bg-red-50"
                >
                  Hapus PRD
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Error alert banner */}
      {error && (
        <div
          role="alert"
          className="mx-5 mt-3 flex items-center justify-between rounded-xl bg-red-50 px-4 py-3 text-xs text-red-700 md:mx-7"
        >
          <span>{error}</span>
          {saveState === "error" && (
            <button onClick={() => void load()} className="ml-3 font-semibold underline">
              Muat ulang server
            </button>
          )}
        </div>
      )}

      {/* Share URL modal / bar */}
      {shareUrl && shareUrl !== "share" && (
        <div className="mx-5 mt-3 flex flex-wrap items-center gap-2 rounded-xl border border-[#dfe9d9] bg-[#f2f6ef] p-3 text-xs md:mx-7">
          <span className="font-semibold text-[#456449]">Tautan publik:</span>
          <Input readOnly value={shareUrl} className="h-8 min-w-0 flex-1 text-xs" />
          <Button
            size="sm"
            variant="secondary"
            onClick={() => {
              navigator.clipboard.writeText(shareUrl);
              toast.success("Tautan disalin ke clipboard!");
            }}
          >
            <Copy size={13} /> Salin
          </Button>
          <Button size="sm" variant="secondary" onClick={() => void revokeShare()}>
            Cabut tautan
          </Button>
          <button className="text-[#98a197] hover:text-[#333]" onClick={() => setShareUrl("")}>
            Tutup
          </button>
        </div>
      )}

      {/* Version history panel */}
      {showVersions && (
        <Card className="mx-5 mt-3 p-4 md:mx-7">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">Riwayat versi tersimpan</h3>
            <button onClick={() => setShowVersions(false)} className="text-xs text-[#8d988d]">
              Tutup
            </button>
          </div>
          {versions.length ? (
            <div className="mt-3 space-y-2">
              {versions.map((version) => (
                <div
                  key={version.id}
                  className="flex items-center justify-between rounded-xl bg-[#f8f9f5] px-3.5 py-2.5"
                >
                  <div>
                    <p className="text-xs font-semibold">
                      v{version.version} · {version.title}
                    </p>
                    <p className="mt-0.5 text-[10px] text-[#97a097]">
                      {formatDate(version.createdAt)}
                    </p>
                  </div>
                  <div className="flex gap-1.5">
                    <Button size="sm" variant="ghost" onClick={() => setCompareVersion(version)}>
                      Bandingkan
                    </Button>
                    <Button size="sm" variant="secondary" onClick={() => void restoreVersion(version)}>
                      Pulihkan
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-3 text-xs text-[#929d92]">Belum ada riwayat versi sebelumnya.</p>
          )}
        </Card>
      )}

      {/* Version diff view */}
      {compareVersion && (
        <Card className="mx-5 mt-3 p-4 md:mx-7">
          <div className="flex items-center justify-between border-b border-[#edf0ea] pb-3">
            <div>
              <h3 className="text-sm font-semibold">
                Perbandingan Versi {compareVersion.version} dan Versi Aktif (v{prd.version})
              </h3>
              <p className="mt-1 text-xs text-[#8d988d]">
                {compareVersion.title} → {title}
              </p>
            </div>
            <button onClick={() => setCompareVersion(null)} className="text-xs text-[#8d988d]">
              Tutup
            </button>
          </div>
          <div className="mt-4 space-y-3 max-h-[60vh] overflow-y-auto">
            {fields
              .filter(
                (field) =>
                  JSON.stringify(compareVersion.content[field.key]) !== JSON.stringify(content[field.key])
              )
              .map((field) => (
                <div key={field.key} className="rounded-xl bg-[#f8f9f5] p-3">
                  <h4 className="text-xs font-semibold text-[#315642]">{field.label}</h4>
                  <div className="mt-2 grid gap-2 md:grid-cols-2">
                    <div className="rounded-lg bg-red-50/70 p-3">
                      <p className="text-[10px] font-bold uppercase tracking-wider text-[#a57670]">
                        v{compareVersion.version}
                      </p>
                      <p className="mt-1 whitespace-pre-wrap text-xs text-[#6e716c]">
                        {valueText(compareVersion.content[field.key], field.kind) || "(kosong)"}
                      </p>
                    </div>
                    <div className="rounded-lg bg-[#eaf2e6] p-3">
                      <p className="text-[10px] font-bold uppercase tracking-wider text-[#6f8a6c]">
                        Versi saat ini
                      </p>
                      <p className="mt-1 whitespace-pre-wrap text-xs text-[#526455]">
                        {valueText(content[field.key], field.kind) || "(kosong)"}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
          </div>
        </Card>
      )}

      {/* Main 3-column layout */}
      <div className="grid min-h-[calc(100vh-150px)] grid-cols-1 xl:grid-cols-[220px_minmax(0,1fr)_300px]">
        {/* Left Sidebar — Section navigation */}
        <aside className="hidden border-r border-[#edf0ea] px-4 py-6 xl:block">
          <p className="px-2 text-[10px] font-bold uppercase tracking-[.15em] text-[#a0aaa0]">
            Daftar Seksi PRD
          </p>
          <nav className="mt-3 space-y-0.5">
            {fields.map((field) => (
              <button
                key={field.key}
                onClick={() => {
                  setActiveField(field.key);
                  document.getElementById(`field-${field.key}`)?.scrollIntoView({
                    behavior: "smooth",
                    block: "center",
                  });
                }}
                className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-left text-[11px] transition ${
                  activeField === field.key
                    ? "bg-[#eaf0e5] font-semibold text-[#456449]"
                    : "text-[#7e8a7e] hover:bg-white hover:text-[#26372d]"
                }`}
              >
                <span className="truncate">{field.label}</span>
                {Boolean(content[field.key]) && (
                  <span className="ml-2 size-1.5 shrink-0 rounded-full bg-[#83a17e]" />
                )}
              </button>
            ))}
          </nav>
          <Button
            className="mt-5 w-full"
            variant="secondary"
            size="sm"
            onClick={() =>
              setContent((prev) => ({
                ...prev,
                customSections: [
                  ...(Array.isArray(prev.customSections)
                    ? (prev.customSections as Array<{ title: string; body: string }>)
                    : []),
                  { title: "Section baru", body: "" },
                ],
              }))
            }
          >
            <Plus size={13} /> Tambah section
          </Button>
        </aside>

        {/* Center Main Editor / Preview */}
        <main className="min-w-0 px-4 py-6 md:px-7 xl:px-10">
          <div className="mx-auto max-w-3xl">
            {mode === "edit" ? (
              <>
                <div className="mb-7">
                  <Input
                    aria-label="Judul PRD"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="h-auto border-0 bg-transparent px-0 py-1 text-3xl font-semibold tracking-[-.045em] shadow-none focus:ring-0 md:text-4xl"
                    placeholder="Judul Produk / Dokumen"
                  />
                  <p className="mt-1 text-xs text-[#94a094]">
                    Dokumen produk · Terakhir diperbarui {formatDate(prd.updatedAt)}
                  </p>
                </div>

                {fields.map((field) => (
                  <section
                    key={field.key}
                    id={`field-${field.key}`}
                    onMouseEnter={() => setActiveField(field.key)}
                    className={`mb-6 scroll-mt-36 rounded-2xl border bg-white p-5 md:p-6 transition ${
                      activeField === field.key ? "border-[#bed3bc] shadow-sm" : "border-[#e8ece5]"
                    }`}
                  >
                    <div className="mb-3 flex items-start justify-between gap-3">
                      <div>
                        <span className="text-[10px] font-bold uppercase tracking-[.14em] text-[#8aa082]">
                          SEKSI {String(fields.indexOf(field) + 1).padStart(2, "0")}
                        </span>
                        <h2 className="mt-0.5 text-sm font-semibold text-[#25372a]">{field.label}</h2>
                      </div>
                      <button
                        aria-label={`Buat ulang ${field.label}`}
                        title="Buat ulang bagian ini dengan AI"
                        onClick={() => {
                          setActiveField(field.key);
                          void regenerate();
                        }}
                        className="rounded-lg p-2 text-[#8c9b8b] hover:bg-[#f1f5ef] hover:text-[#426349] transition"
                      >
                        <RefreshCw size={14} />
                      </button>
                    </div>

                    {field.kind === "requirements" ? (
                      <RequirementEditor
                        items={
                          Array.isArray(content[field.key])
                            ? (content[field.key] as Requirement[])
                            : []
                        }
                        onChange={(items) =>
                          setContent((prev) => ({ ...prev, [field.key]: items }))
                        }
                      />
                    ) : (
                      <Textarea
                        value={valueText(content[field.key], field.kind)}
                        onChange={(e) => updateField(field, e.target.value)}
                        placeholder={
                          field.kind === "text"
                            ? "Tulis ringkasan atau penjelasan seksi ini..."
                            : "Tambahkan satu item per baris..."
                        }
                        className={
                          field.key === "executiveSummary" ||
                          field.key === "productOverview" ||
                          field.key === "problemStatement" ||
                          field.key === "architecture"
                            ? "min-h-32 text-xs"
                            : "min-h-24 text-xs"
                        }
                      />
                    )}
                  </section>
                ))}

                {/* Custom Sections */}
                {Array.isArray(content.customSections) &&
                  (content.customSections as Array<{ title: string; body: string }>).map((section, index) => (
                    <section key={index} className="mb-6 rounded-2xl border border-[#e8ece5] bg-white p-5">
                      <div className="flex items-center gap-2 mb-3">
                        <Input
                          value={section.title}
                          onChange={(e) =>
                            setContent((prev) => ({
                              ...prev,
                              customSections: (
                                prev.customSections as Array<{ title: string; body: string }>
                              ).map((item, i) => (i === index ? { ...item, title: e.target.value } : item)),
                            }))
                          }
                          className="h-9 font-semibold text-sm"
                          placeholder="Judul seksi tambahan"
                        />
                        <button
                          type="button"
                          onClick={() => {
                            if (confirm("Hapus section ini?")) {
                              setContent((prev) => ({
                                ...prev,
                                customSections: (
                                  prev.customSections as Array<{ title: string; body: string }>
                                ).filter((_, i) => i !== index),
                              }));
                            }
                          }}
                          className="px-2 text-xs text-red-600 hover:underline"
                        >
                          Hapus
                        </button>
                      </div>
                      <Textarea
                        value={section.body}
                        onChange={(e) =>
                          setContent((prev) => ({
                            ...prev,
                            customSections: (
                              prev.customSections as Array<{ title: string; body: string }>
                            ).map((item, i) => (i === index ? { ...item, body: e.target.value } : item)),
                          }))
                        }
                        placeholder="Konten seksi..."
                        className="min-h-24 text-xs"
                      />
                    </section>
                  ))}
              </>
            ) : (
              <article className="prose-prd prose prose-sm max-w-none rounded-2xl border border-[#e8ece5] bg-white p-6 md:p-10 shadow-sm">
                <ReactMarkdown
                  components={{
                    code({ className, children, ...props }) {
                      const match = /language-(\w+)/.exec(className || "");
                      const isMermaid = match && match[1] === "mermaid";
                      const codeText = String(children).replace(/\n$/, "");
                      if (
                        isMermaid ||
                        (!match &&
                          (codeText.startsWith("graph ") ||
                            codeText.startsWith("flowchart ") ||
                            codeText.startsWith("sequenceDiagram ") ||
                            codeText.startsWith("classDiagram ") ||
                            codeText.startsWith("stateDiagram")))
                      ) {
                        return <MermaidChart chart={codeText} />;
                      }
                      return (
                        <code className={className} {...props}>
                          {children}
                        </code>
                      );
                    },
                  }}
                >
                  {markdown}
                </ReactMarkdown>
              </article>
            )}
          </div>
        </main>

        {/* Right Sidebar — AI Assistant Panel */}
        <aside className="hidden border-l border-[#edf0ea] bg-[#f8f9f6] p-5 xl:block">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-[10px] font-bold uppercase tracking-[.14em] text-[#9aa49a]">
                Nara Assistant
              </p>
              <h2 className="mt-0.5 text-sm font-semibold">Ruang untuk ide</h2>
            </div>
            <span className="grid size-8 place-items-center rounded-xl bg-[#e7eee3] text-[#658065]">
              <WandSparkles size={16} />
            </span>
          </div>

          <div className="mt-4 rounded-xl border border-[#e6ebe3] bg-white p-3.5">
            <p className="text-xs font-semibold text-[#283d2e]">Seksi aktif: {String(activeField)}</p>
            <p className="mt-1 text-xs leading-5 text-[#899589]">
              Jalankan AI tools di bawah untuk menyempurnakan atau menganalisis seksi yang sedang dipilih.
            </p>
          </div>

          <p className="mt-6 text-[10px] font-bold uppercase tracking-[.14em] text-[#9aa49a]">
            Aksi Cepat
          </p>
          <div className="mt-2 space-y-1.5">
            {[
              ["find_missing_requirements", "Cari kebutuhan yang terlewat"],
              ["detect_ambiguity", "Temukan bagian ambigu"],
              ["review_security", "Tinjau keamanan & privasi"],
              ["generate_tasks", "Buat backlog tasks"],
            ].map(([action, label]) => (
              <button
                key={action}
                disabled={aiBusy}
                onClick={() => void runAction(action)}
                className="flex w-full items-center justify-between rounded-xl border border-[#e8ece5] bg-white px-3 py-2.5 text-left text-[11px] font-medium text-[#5c6e5e] hover:border-[#cddbc9] hover:text-[#315642] transition"
              >
                <span>{label}</span>
                <ArrowLeft size={12} className="rotate-180" />
              </button>
            ))}
          </div>

          <div className="mt-6 rounded-xl bg-[#eaf0e5] p-4">
            <p className="text-[10px] font-bold uppercase tracking-[.13em] text-[#748c70]">
              Status Dokumen
            </p>
            <div className="mt-2.5 flex items-center justify-between text-xs">
              <span className="text-[#738073]">Status</span>
              <Badge tone={status === "completed" ? "green" : status === "in_progress" ? "amber" : "gray"}>
                {status.replace("_", " ")}
              </Badge>
            </div>
            <div className="mt-2 flex items-center justify-between text-xs">
              <span className="text-[#738073]">Versi</span>
              <span className="font-semibold text-[#2f4233]">{prd.version}</span>
            </div>
            <div className="mt-2 flex items-center justify-between text-xs">
              <span className="text-[#738073]">Diperbarui</span>
              <span className="font-semibold text-[#2f4233]">{formatDate(prd.updatedAt)}</span>
            </div>
          </div>
        </aside>
      </div>

      {/* Floating loading toast for AI */}
      {aiBusy && (
        <div className="fixed bottom-5 right-5 z-40 flex items-center gap-2.5 rounded-xl bg-[#315642] px-4 py-3 text-xs font-semibold text-white shadow-xl animate-in fade-in">
          <LoaderCircle size={15} className="animate-spin text-white" /> AI sedang menganalisis dokumen…
        </div>
      )}

      {/* AI Suggestion Review Modal */}
      {aiResult !== null && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-[#1b2b20]/30 p-3 backdrop-blur-[2px] sm:items-center">
          <div className="max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-[#e1e8de] bg-[#fffefa] p-5 shadow-2xl md:p-7">
            <div className="flex items-start justify-between border-b border-[#edf0ea] pb-4">
              <div className="flex items-center gap-3">
                <span className="grid size-9 place-items-center rounded-xl bg-[#eaf0e5] text-[#648066]">
                  <Sparkles size={17} />
                </span>
                <div>
                  <h2 className="text-base font-semibold text-[#25372a]">Hasil Analisis AI</h2>
                  <p className="mt-0.5 text-xs text-[#8c988d]">
                    Tinjau rekomendasi AI untuk seksi:{" "}
                    <span className="font-semibold text-[#315642]">{String(activeField)}</span>
                  </p>
                </div>
              </div>
              <button
                onClick={() => setAiResult(null)}
                className="text-xs text-[#869186] hover:text-[#333]"
              >
                Tutup
              </button>
            </div>

            {/* Explanation callout if available */}
            {typeof aiResult === "object" && aiResult !== null && "explanation" in aiResult && (
              <div className="mt-4 rounded-xl bg-[#f0f4ee] border border-[#dce5da] p-3.5 text-xs leading-5 text-[#3a523f]">
                <p className="font-semibold mb-1 text-[#2d4232]">Penjelasan AI:</p>
                {String(aiResult.explanation)}
              </div>
            )}

            <div className="mt-4">
              <p className="text-xs font-semibold text-[#6d7b6f] mb-1.5">Konten yang Diusulkan:</p>
              <pre className="max-h-[42vh] overflow-auto whitespace-pre-wrap rounded-xl bg-[#f5f7f2] border border-[#e5e9e0] p-4 text-xs leading-5 text-[#384a3a] font-mono">
                {typeof aiResult === "object" && aiResult !== null && "result" in aiResult
                  ? typeof aiResult.result === "string"
                    ? aiResult.result
                    : JSON.stringify(aiResult.result, null, 2)
                  : typeof aiResult === "string"
                  ? aiResult
                  : JSON.stringify(aiResult, null, 2)}
              </pre>
            </div>

            <div className="mt-5 flex justify-end gap-2 border-t border-[#edf0ea] pt-4">
              <Button variant="secondary" onClick={() => setAiResult(null)}>
                Batal
              </Button>
              <Button onClick={applySuggestion}>
                <Check size={14} /> Terapkan ke Dokumen
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
