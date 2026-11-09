"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Bot,
  Check,
  CircleHelp,
  Cpu,
  LoaderCircle,
  Sparkles,
  Terminal,
} from "lucide-react";
import { api, type AIProviderInfo, type Idea, type PRD, type Project } from "@/lib/api";
import { Button, Card, FieldError, Input, Label, Select, Textarea } from "@/components/ui";
import { useToast } from "@/components/toast";

const schema = z.object({
  productName: z.string().trim().min(2, "Nama produk minimal 2 karakter.").max(120),
  description: z.string().trim().min(20, "Jelaskan ide setidaknya 20 karakter."),
  problem: z.string().trim().min(10, "Ceritakan masalah yang ingin diselesaikan."),
  targetUsers: z.string().trim().min(3, "Sebutkan pengguna yang ingin dibantu."),
  businessGoals: z.string().optional(),
  platform: z.string().min(1),
  productType: z.string().min(1),
  featuresText: z.string().optional(),
  technology: z.string().optional(),
  constraints: z.string().optional(),
  language: z.string().min(1),
  detailLevel: z.string().min(1),
});

type Form = z.infer<typeof schema>;
type Question = { id: string; question: string; required: boolean; options?: string[] };
type Session = { id: string; questions: Question[]; answers: Record<string, string>; ready: boolean };

const defaults: Form = {
  productName: "",
  description: "",
  problem: "",
  targetUsers: "",
  businessGoals: "",
  platform: "Web",
  productType: "SaaS",
  featuresText: "",
  technology: "",
  constraints: "",
  language: "Bahasa Indonesia",
  detailLevel: "balanced",
};

const stepCopy = ["Ceritakan ide", "Perjelas arah", "Susun PRD"];

export default function NewProjectPage() {
  const router = useRouter();
  const toast = useToast();

  const [step, setStep] = useState(1);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [idea, setIdea] = useState<Idea | null>(null);
  const [project, setProject] = useState<Project | null>(null);
  const [session, setSession] = useState<Session | null>(null);
  const [answers, setAnswers] = useState<Record<string, string>>({});

  const [streamingStep, setStreamingStep] = useState<string>("init");
  const [streamingMessage, setStreamingMessage] = useState<string>("Mempersiapkan generator AI...");
  const [streamChunks, setStreamChunks] = useState<string>("");
  const [generatedPRD, setGeneratedPRD] = useState<PRD | null>(null);

  // Dynamic AI Provider & Model Selection
  const [providers, setProviders] = useState<AIProviderInfo[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<string>("openai");
  const [selectedModel, setSelectedModel] = useState<string>("gpt-4o-mini");

  const generationKey = useRef<string | null>(null);
  const streamEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    Promise.all([api.getAIProviders(), api.getAISettings()])
      .then(([provs, settings]) => {
        setProviders(provs);
        if (settings) {
          if (settings.provider) setSelectedProvider(settings.provider);
          if (settings.model) setSelectedModel(settings.model);
        }
      })
      .catch(() => {
        // Fallback silently if settings fail to load
      });
  }, []);

  useEffect(() => {
    if (streamEndRef.current) {
      streamEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [streamChunks]);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: defaults,
  });

  async function begin(values: Form) {
    setBusy(true);
    setError("");

    const nextIdea: Idea = {
      productName: values.productName,
      description: values.description,
      problem: values.problem,
      targetUsers: values.targetUsers,
      businessGoals: values.businessGoals || "",
      platform: values.platform,
      productType: values.productType,
      features: (values.featuresText || "")
        .split(/[,\n]/)
        .map((x) => x.trim())
        .filter(Boolean),
      technology: values.technology || "",
      constraints: values.constraints || "",
      language: values.language,
      detailLevel: values.detailLevel,
    };

    try {
      const created = project
        ? await api.patch<Project>(`/projects/${project.id}`, {
            name: nextIdea.productName,
            description: nextIdea.description,
          })
        : await api.post<Project>("/projects", {
            name: nextIdea.productName,
            description: nextIdea.description,
          });

      setProject(created);
      setIdea(nextIdea);

      const result = await api.post<Session>("/ai/clarify", { idea: nextIdea });
      setSession(result);
      setAnswers(result.answers || {});
      setStep(2);
      toast.info("Ide tersimpan. Silakan jawab pertanyaan klarifikasi jika relevan.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Gagal memulai proyek.");
      toast.error("Gagal memulai proyek.");
    } finally {
      setBusy(false);
    }
  }

  async function proceed(skip = false) {
    if (!project || !idea || !session) return;
    setBusy(true);
    setError("");

    try {
      const updated = await api.post<Session>("/ai/clarify", {
        sessionId: session.id,
        answers,
        skip,
      });
      setSession(updated);

      if (!updated.ready && !skip) {
        setAnswers(updated.answers || answers);
        toast.info("AI memerlukan sedikit klarifikasi tambahan.");
        setBusy(false);
        return;
      }

      generationKey.current ||= crypto.randomUUID();
      setStep(3);
      setStreamingStep("init");
      setStreamingMessage("Menghubungkan ke layanan AI generator...");
      setStreamChunks("");

      const prd = await api.streamPRD(
        {
          projectId: project.id,
          sessionId: updated.id,
          idempotencyKey: generationKey.current,
          provider: selectedProvider || undefined,
          model: selectedModel || undefined,
        },
        (event) => {
          if (event.type === "progress") {
            setStreamingStep(event.step);
            setStreamingMessage(event.message);
          } else if (event.type === "chunk") {
            setStreamChunks((prev) => prev + event.chunk);
          } else if (event.type === "done") {
            setStreamingStep("done");
            setStreamingMessage("Dokumen PRD berhasil dibuat dan disimpan!");
            setGeneratedPRD(event.prd);
          } else if (event.type === "error") {
            setError(event.message);
          }
        }
      );

      generationKey.current = null;
      setGeneratedPRD(prd);
      toast.success("PRD berhasil dibuat! Mengalihkan ke editor...");
      setTimeout(() => {
        router.push(`/prd/${prd.id}`);
      }, 1500);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Gagal menghasilkan PRD.";
      setError(msg);
      toast.error(msg);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl pb-12">
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-2 text-xs font-semibold text-[#809080] hover:text-[#315642] transition"
      >
        <ArrowLeft size={14} /> Kembali ke workspace
      </Link>

      <div className="mt-6 flex items-end justify-between gap-4">
        <div>
          <div className="inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-[.17em] text-[#789176]">
            <Sparkles size={13} /> WORKSPACE IDE
          </div>
          <h1 className="mt-3 text-3xl font-semibold tracking-[-.045em] text-[#24372a]">
            {step === 1 ? "Mulai dari ide." : step === 2 ? "Sedikit lebih jelas." : "Menyusun PRD."}
          </h1>
          <p className="mt-2 text-sm text-[#859084]">
            {step === 1
              ? "Ceritakan gambaran awal ide produkmu. AI akan membantu merumuskan kebutuhan teknisnya."
              : step === 2
              ? "AI telah meninjau idemu. Jawab pertanyaan berikut untuk hasil yang lebih presisi."
              : "AI sedang menganalisis ide dan merumuskan spesifikasi produk lengkap secara real-time."}
          </p>
        </div>
        <span className="hidden rounded-full bg-[#eaf0e5] px-3.5 py-1.5 text-[11px] font-semibold text-[#547353] sm:inline-flex">
          Langkah {step === 1 ? "01" : step === 2 ? "02" : "03"} / 03
        </span>
      </div>

      {/* Progress stepper */}
      <div className="mt-8 grid grid-cols-3 gap-2">
        {stepCopy.map((copy, index) => (
          <div key={copy}>
            <div
              className={`h-1.5 rounded-full transition-all ${
                index < step ? "bg-[#4f7556]" : "bg-[#e5e9e1]"
              }`}
            />
            <p
              className={`mt-2 text-[10px] font-semibold ${
                index + 1 === step ? "text-[#4a694d]" : "text-[#a0a99f]"
              }`}
            >
              {copy}
            </p>
          </div>
        ))}
      </div>

      {step === 1 ? (
        <form onSubmit={handleSubmit(begin)} className="mt-6">
          <Card className="p-5 md:p-8">
            <div className="flex items-start gap-3 border-b border-[#edf0ea] pb-5">
              <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-[#eaf0e5] text-[#628066]">
                <CircleHelp size={17} />
              </span>
              <div>
                <h2 className="font-semibold text-[#25372a]">Gambaran Produk</h2>
                <p className="mt-0.5 text-xs text-[#8d988d]">
                  Mulai dari apa yang paling Anda ketahui saat ini.
                </p>
              </div>
            </div>

            {/* AI Engine & Model Selector */}
            {providers.length > 0 && (
              <div className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[#dfe7dc] bg-[#f8faf6] p-3 text-xs">
                <div className="flex items-center gap-2">
                  <span className="grid size-6 place-items-center rounded-lg bg-[#315642] text-white">
                    <Cpu size={13} />
                  </span>
                  <div>
                    <span className="font-semibold text-[#25372a]">Engine & Model AI:</span>
                    <span className="ml-1.5 text-[11px] text-[#6e8071]">
                      (Dapat disesuaikan khusus untuk PRD ini)
                    </span>
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <Select
                    aria-label="Pilih Provider"
                    value={selectedProvider}
                    onChange={(e) => {
                      const prov = e.target.value;
                      setSelectedProvider(prov);
                      const p = providers.find((x) => x.id === prov);
                      if (p) setSelectedModel(p.defaultModel);
                    }}
                    className="h-8 py-0 px-2.5 text-xs font-medium"
                  >
                    {providers.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name}
                      </option>
                    ))}
                  </Select>

                  <Select
                    aria-label="Pilih Model"
                    value={selectedModel}
                    onChange={(e) => setSelectedModel(e.target.value)}
                    className="h-8 py-0 px-2.5 text-xs font-mono font-semibold"
                  >
                    {providers.find((p) => p.id === selectedProvider)?.models.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.name}
                      </option>
                    ))}
                  </Select>

                  <Link
                    href="/settings"
                    className="text-[11px] text-[#6d8270] hover:text-[#2d4233] underline ml-1"
                    title="Ubah default permanen di Pengaturan"
                  >
                    Ubah Default
                  </Link>
                </div>
              </div>
            )}

            <div className="mt-6 grid gap-5 md:grid-cols-2">
              <div className="md:col-span-2">
                <Label htmlFor="productName">
                  Nama Produk <span className="text-[#688a64]">*</span>
                </Label>
                <Input
                  id="productName"
                  placeholder="Contoh: Nara Workspace, PayFlow, RuangKelas"
                  {...register("productName")}
                />
                <FieldError>{errors.productName?.message}</FieldError>
              </div>

              <div className="md:col-span-2">
                <Label htmlFor="description">
                  Jelaskan Ide Produk <span className="text-[#688a64]">*</span>
                </Label>
                <Textarea
                  id="description"
                  placeholder="Apa yang ingin Anda buat? Gambarkan cara kerja dan nilai utamanya..."
                  {...register("description")}
                />
                <FieldError>{errors.description?.message}</FieldError>
              </div>

              <div>
                <Label htmlFor="problem">
                  Masalah yang Ingin Diselesaikan <span className="text-[#688a64]">*</span>
                </Label>
                <Textarea
                  id="problem"
                  className="min-h-24"
                  placeholder="Apa tantangan atau masalah utama yang dihadapi pengguna saat ini?"
                  {...register("problem")}
                />
                <FieldError>{errors.problem?.message}</FieldError>
              </div>

              <div>
                <Label htmlFor="targetUsers">
                  Target Pengguna / Persona <span className="text-[#688a64]">*</span>
                </Label>
                <Textarea
                  id="targetUsers"
                  className="min-h-24"
                  placeholder="Siapa pengguna utamanya? (misal: Software Engineer, Founder, Mahasiswa)..."
                  {...register("targetUsers")}
                />
                <FieldError>{errors.targetUsers?.message}</FieldError>
              </div>

              <div>
                <Label htmlFor="businessGoals">Tujuan Bisnis / Success Metrics</Label>
                <Textarea
                  id="businessGoals"
                  className="min-h-24"
                  placeholder="Contoh: Meningkatkan retensi pengguna 20%, membuka revenue stream baru..."
                  {...register("businessGoals")}
                />
              </div>

              <div>
                <Label htmlFor="featuresText">Fitur-fitur Utama yang Dibayangkan</Label>
                <Textarea
                  id="featuresText"
                  className="min-h-24"
                  placeholder="Pisahkan fitur dengan koma atau baris baru..."
                  {...register("featuresText")}
                />
              </div>

              <div>
                <Label htmlFor="platform">Platform Utama</Label>
                <Select id="platform" {...register("platform")}>
                  <option>Web</option>
                  <option>Mobile (iOS & Android)</option>
                  <option>API / Backend Service</option>
                  <option>Multi-platform</option>
                </Select>
              </div>

              <div>
                <Label htmlFor="productType">Kategori Produk</Label>
                <Select id="productType" {...register("productType")}>
                  <option>SaaS B2B</option>
                  <option>Consumer App (B2C)</option>
                  <option>Marketplace</option>
                  <option>Internal Tool</option>
                  <option>Developer Tool / API</option>
                  <option>Fintech</option>
                  <option>Other</option>
                </Select>
              </div>

              <div>
                <Label htmlFor="technology">Preferensi Teknologi (Opsional)</Label>
                <Input
                  id="technology"
                  placeholder="Contoh: Next.js, Go, PostgreSQL, AWS"
                  {...register("technology")}
                />
              </div>

              <div>
                <Label htmlFor="constraints">Batasan Waktu atau Regulasi</Label>
                <Input
                  id="constraints"
                  placeholder="Contoh: Peluncuran MVP dalam 3 bulan, kepatuhan GDPR"
                  {...register("constraints")}
                />
              </div>

              <div>
                <Label htmlFor="language">Bahasa Dokumen PRD</Label>
                <Select id="language" {...register("language")}>
                  <option>Bahasa Indonesia</option>
                  <option>English</option>
                </Select>
              </div>

              <div>
                <Label htmlFor="detailLevel">Tingkat Kedalaman Dokumen</Label>
                <Select id="detailLevel" {...register("detailLevel")}>
                  <option value="concise">Ringkas (Fokus MVP)</option>
                  <option value="balanced">Seimbang (Standar Industri)</option>
                  <option value="comprehensive">Komprehensif (Spesifikasi Lengkap & NFR)</option>
                </Select>
              </div>
            </div>

            {error && (
              <div role="alert" className="mt-5 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700">
                {error}
              </div>
            )}

            <div className="mt-7 flex items-center justify-between border-t border-[#edf0ea] pt-5">
              <p className="hidden text-xs text-[#99a399] sm:block">
                Semua bagian dapat disunting kembali nanti di editor.
              </p>
              <Button type="submit" disabled={busy} className="ml-auto">
                {busy ? (
                  <>
                    <LoaderCircle className="animate-spin" size={15} /> Memproses Ide…
                  </>
                ) : (
                  <>
                    Lanjutkan ke Klarifikasi <ArrowRight size={15} />
                  </>
                )}
              </Button>
            </div>
          </Card>
        </form>
      ) : step === 2 ? (
        <Card className="mt-6 p-5 md:p-8">
          <div className="flex items-start gap-3">
            <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-[#eaf0e5] text-[#628066]">
              <Sparkles size={17} />
            </span>
            <div>
              <h2 className="font-semibold text-[#25372a]">
                Pertanyaan Klarifikasi untuk {idea?.productName}
              </h2>
              <p className="mt-1 text-xs leading-5 text-[#8d988d]">
                Jawaban Anda akan membantu AI menghasilkan acceptance criteria dan cakupan teknis yang
                lebih tajam.
              </p>
            </div>
          </div>

          {session?.questions.length ? (
            <div className="mt-6 space-y-5">
              {session.questions.map((question, index) => (
                <div key={question.id} className="rounded-xl bg-[#fbfcf9] border border-[#edf0ea] p-4">
                  <Label htmlFor={`answer-${question.id}`} className="text-xs font-semibold text-[#2f4333]">
                    {index + 1}. {question.question}{" "}
                    {!question.required && (
                      <span className="ml-1 text-[10px] font-normal text-[#a0aaa0]">(opsional)</span>
                    )}
                  </Label>
                  {question.options?.length ? (
                    <Select
                      id={`answer-${question.id}`}
                      value={answers[question.id] || ""}
                      onChange={(e) =>
                        setAnswers((prev) => ({ ...prev, [question.id]: e.target.value }))
                      }
                      className="mt-2 text-xs"
                    >
                      <option value="">Pilih opsi jawaban yang sesuai…</option>
                      {question.options.map((option) => (
                        <option key={option} value={option}>
                          {option}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <Textarea
                      id={`answer-${question.id}`}
                      className="mt-2 min-h-20 text-xs"
                      placeholder="Tuliskan jawaban atau preferensi Anda..."
                      value={answers[question.id] || ""}
                      onChange={(e) =>
                        setAnswers((prev) => ({ ...prev, [question.id]: e.target.value }))
                      }
                    />
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="mt-6 rounded-xl bg-[#f6f8f3] p-4 text-sm text-[#546e55]">
              AI telah memiliki informasi yang cukup dari ide awal Anda untuk merumuskan PRD lengkap.
            </div>
          )}

          {error && (
            <div role="alert" className="mt-5 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          )}

          <div className="mt-7 flex flex-col-reverse justify-between gap-3 border-t border-[#edf0ea] pt-5 sm:flex-row">
            <Button type="button" variant="secondary" disabled={busy} onClick={() => void proceed(true)}>
              Lewati Pertanyaan & Gunakan Asumsi Wajar
            </Button>
            <Button type="button" disabled={busy} onClick={() => void proceed(false)}>
              {busy ? (
                <>
                  <LoaderCircle className="animate-spin" size={15} /> Menyusun PRD…
                </>
              ) : (
                <>
                  Buat Dokumen PRD <ArrowRight size={15} />
                </>
              )}
            </Button>
          </div>
        </Card>
      ) : (
        <Card className="mt-6 p-5 md:p-8">
          <div className="flex items-start justify-between gap-4 border-b border-[#edf0ea] pb-6">
            <div className="flex items-start gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-[#eaf0e5] text-[#315642]">
                <Sparkles size={20} className="animate-pulse" />
              </span>
              <div>
                <div className="flex items-center gap-2">
                  <h2 className="text-base font-semibold text-[#25372a]">
                    Menyusun PRD: {idea?.productName}
                  </h2>
                  <span className="rounded-full bg-[#eaf0e5] px-2.5 py-0.5 text-[10px] font-semibold text-[#486b51]">
                    Live Streaming Pipeline
                  </span>
                </div>
                <p className="mt-1 text-xs text-[#7e8c80]">
                  {streamingMessage}
                </p>
              </div>
            </div>
            {streamingStep === "done" && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-[#dcf4e3] px-3 py-1 text-xs font-semibold text-[#2b7244]">
                <Check size={14} /> Dokumen Siap
              </span>
            )}
          </div>

          {/* Stepper Pipeline Indicators */}
          <div className="mt-6 grid grid-cols-2 gap-2 sm:grid-cols-5">
            {[
              { id: "init", label: "Inisialisasi" },
              { id: "generating", label: "Penulisan PRD" },
              { id: "validating", label: "Validasi Struktur" },
              { id: "saving", label: "Penyimpanan DB" },
              { id: "done", label: "Selesai" },
            ].map((st, i) => {
              const order = ["init", "generating", "validating", "saving", "done"];
              const currentIndex = order.indexOf(streamingStep);
              const targetIndex = order.indexOf(st.id);
              const isPast = streamingStep === "done" || targetIndex < currentIndex;
              const isActive = streamingStep === st.id && streamingStep !== "done";

              return (
                <div
                  key={st.id}
                  className={`flex flex-col items-center justify-center rounded-xl border p-2.5 text-center transition ${
                    isPast
                      ? "border-[#b8d8be] bg-[#f1f8f3] text-[#2f633d]"
                      : isActive
                      ? "border-[#4f7556] bg-[#eaf0e5] text-[#24372a] shadow-xs"
                      : "border-[#edf0ea] bg-[#fafbf9] text-[#9ba49b]"
                  }`}
                >
                  <div className="flex size-5 items-center justify-center rounded-full text-xs">
                    {isPast ? (
                      <Check size={13} className="text-[#315642]" />
                    ) : isActive ? (
                      <LoaderCircle size={13} className="animate-spin text-[#4f7556]" />
                    ) : (
                      <span className="text-[10px] font-medium text-[#abb4ab]">{i + 1}</span>
                    )}
                  </div>
                  <span className="mt-1 text-[11px] font-semibold">{st.label}</span>
                </div>
              );
            })}
          </div>

          {/* Typewriter Stream Terminal */}
          <div className="mt-6 overflow-hidden rounded-2xl border border-[#2b3d30] bg-[#17231a] shadow-sm">
            <div className="flex items-center justify-between border-b border-[#25392b] bg-[#1e2f23] px-4 py-2 text-xs text-[#a4bda9]">
              <div className="flex items-center gap-2">
                <Terminal size={14} className="text-[#64cb85]" />
                <span className="font-mono text-[11px] font-semibold">Live AI Token Stream</span>
              </div>
              <div className="flex items-center gap-3 font-mono text-[11px]">
                <span className="flex items-center gap-1.5">
                  <span className="size-2 rounded-full bg-[#52d37c] animate-pulse" />
                  Streaming
                </span>
                <span className="text-[#849a88]">{streamChunks.length.toLocaleString()} byte diterima</span>
              </div>
            </div>

            <div className="max-h-80 overflow-y-auto p-4 font-mono text-xs leading-relaxed text-[#cde0d2] select-text">
              {streamChunks ? (
                <pre className="whitespace-pre-wrap break-all font-mono">
                  {streamChunks}
                  {streamingStep !== "done" && (
                    <span className="inline-block size-2.5 animate-ping bg-[#64cb85] ml-1 align-middle" />
                  )}
                </pre>
              ) : (
                <div className="flex min-h-[140px] items-center justify-center gap-2 text-xs text-[#708a76]">
                  <LoaderCircle size={15} className="animate-spin text-[#589c6d]" />
                  Menunggu respons token pertama dari AI engine...
                </div>
              )}
              <div ref={streamEndRef} />
            </div>
          </div>

          {error && (
            <div role="alert" className="mt-5 rounded-xl bg-red-50 p-4 text-xs text-red-700">
              <div className="flex items-center gap-2 font-semibold">
                <AlertTriangle size={15} /> Pembuatan PRD Mengalami Kendala
              </div>
              <p className="mt-1.5">{error}</p>
              <div className="mt-3 flex gap-2">
                <Button size="sm" variant="danger" onClick={() => void proceed(false)}>
                  Coba Ulangi
                </Button>
                <Button size="sm" variant="secondary" onClick={() => setStep(2)}>
                  Kembali ke Klarifikasi
                </Button>
              </div>
            </div>
          )}

          {generatedPRD && (
            <div className="mt-6 flex flex-col items-center justify-between gap-3 rounded-2xl bg-[#eaf0e5] p-4 sm:flex-row">
              <div className="flex items-center gap-2.5 text-xs text-[#284533]">
                <span className="grid size-6 place-items-center rounded-full bg-[#315642] text-white">
                  <Check size={14} />
                </span>
                <span className="font-semibold">
                  PRD &ldquo;{generatedPRD.title}&rdquo; siap! Mengalihkan ke editor dokumen...
                </span>
              </div>
              <Button
                size="sm"
                onClick={() => router.push(`/prd/${generatedPRD.id}`)}
                className="w-full sm:w-auto"
              >
                Buka Dokumen Sekarang <ArrowRight size={13} />
              </Button>
            </div>
          )}
        </Card>
      )}

      <div className="mt-6 flex items-center justify-center gap-2 text-[10px] text-[#a0a99f]">
        <Check size={12} className="text-[#648866]" /> Ide dan dokumen dienkripsi serta hanya dapat diakses
        oleh akun Anda
      </div>
    </div>
  );
}
