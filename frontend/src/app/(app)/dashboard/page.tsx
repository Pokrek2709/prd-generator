"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import {
  ArrowRight,
  ArrowUpRight,
  Clock3,
  FileText,
  FolderKanban,
  Plus,
  Search,
  Sparkles,
  TrendingUp,
} from "lucide-react";
import { api, type Project } from "@/lib/api";
import { Badge, Button, Card, Input } from "@/components/ui";
import { formatDate } from "@/lib/utils";

type DashboardData = {
  totalProjects: number;
  totalPRDs: number;
  draftPRDs: number;
  recentDocuments: Array<{
    prdId: string;
    title: string;
    status: string;
    updatedAt: string;
    projectId: string;
    projectName: string;
  }>;
};

const empty: DashboardData = {
  totalProjects: 0,
  totalPRDs: 0,
  draftPRDs: 0,
  recentDocuments: [],
};

const tone = (status: string) =>
  status === "completed"
    ? ("green" as const)
    : status === "in_progress"
    ? ("amber" as const)
    : ("gray" as const);

export default function DashboardPage() {
  const params = useSearchParams();
  const view = params.get("view");

  const [summary, setSummary] = useState(empty);
  const [projects, setProjects] = useState<Project[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load(query = "") {
    setError("");
    try {
      const [data, list] = await Promise.all([
        api.get<DashboardData>("/dashboard"),
        api.get<Project[]>(
          `/projects?limit=8&sort=updated_desc${query ? `&q=${encodeURIComponent(query)}` : ""}`
        ),
      ]);
      setSummary(data || empty);
      setProjects(list || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Dashboard belum dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => {
      if (!loading) void load(search);
    }, 260);
    return () => clearTimeout(timer);
  }, [search]);

  const counts = useMemo(
    () => [
      {
        label: "Total proyek",
        value: summary.totalProjects,
        icon: FolderKanban,
        note: "Ruang untuk ide produk",
      },
      {
        label: "Dokumen PRD",
        value: summary.totalPRDs,
        icon: FileText,
        note: "Dokumen tersimpan",
      },
      {
        label: "Draft aktif",
        value: summary.draftPRDs,
        icon: TrendingUp,
        note: "Siap dilanjutkan",
      },
    ],
    [summary]
  );

  return (
    <div className="space-y-8 pb-12">
      {/* Header section */}
      <section className="flex flex-col justify-between gap-5 md:flex-row md:items-end">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-[#e0e8dc] bg-white px-3 py-1.5 text-[11px] font-semibold text-[#6e866c]">
            <Sparkles size={13} /> RUANG UNTUK IDE BESAR ANDA
          </div>
          <h1 className="mt-4 max-w-2xl text-3xl font-semibold leading-tight tracking-[-.045em] md:text-[42px] text-[#24372a]">
            Dari ide, jadi
            <br className="hidden sm:block" /> langkah nyata.
          </h1>
          <p className="mt-3 max-w-lg text-sm leading-6 text-[#849084]">
            Selamat datang kembali. Ide hebat berikutnya dimulai di sini.
          </p>
        </div>
        <Link href="/projects/new">
          <Button size="lg">
            <Plus size={17} /> Buat PRD baru
          </Button>
        </Link>
      </section>

      {/* KPI Cards */}
      <section className="grid gap-4 sm:grid-cols-3">
        {counts.map(({ label, value, icon: Icon, note }, index) => (
          <Card key={label} className="p-5 md:p-6">
            <div className="flex items-start justify-between">
              <span className="text-sm font-medium text-[#748075]">{label}</span>
              <span
                className={`grid size-9 place-items-center rounded-xl ${
                  index === 0
                    ? "bg-[#e8eee6] text-[#4d7053]"
                    : index === 1
                    ? "bg-[#f4f0e4] text-[#9c8652]"
                    : "bg-[#eaf1ed] text-[#55776a]"
                }`}
              >
                <Icon size={17} />
              </span>
            </div>
            <div className="mt-4 text-3xl font-semibold tracking-tight text-[#25372a]">
              {loading ? (
                <span className="inline-block h-8 w-12 animate-pulse rounded bg-[#eff1eb]" />
              ) : (
                value
              )}
            </div>
            <div className="mt-1 text-xs text-[#9aa49a]">{note}</div>
          </Card>
        ))}
      </section>

      {/* Hero Banner */}
      <section className="relative overflow-hidden rounded-[1.6rem] bg-[#eaf0e5] p-6 md:p-8">
        <div className="absolute -right-8 -top-20 size-64 rounded-full border border-[#d4e1ce]" />
        <div className="absolute -right-2 -top-14 size-52 rounded-full border border-[#d4e1ce]" />
        <div className="relative flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
          <div>
            <div className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-[.18em] text-[#748c70]">
              <Sparkles size={14} /> YOUR IDEA, CLEARLY DEFINED
            </div>
            <h2 className="mt-3 text-2xl font-semibold tracking-[-.04em] md:text-[28px] text-[#24372a]">
              Ide besar dimulai dari sini.
            </h2>
            <p className="mt-2 text-sm text-[#7c897b]">
              Ceritakan ide produkmu. AI membantu merapikan kebutuhan teknisnya.
            </p>
          </div>
          <Link href="/projects/new">
            <Button variant="secondary">
              Mulai dengan ide <ArrowRight size={15} />
            </Button>
          </Link>
        </div>
      </section>

      {error && (
        <div role="alert" className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}{" "}
          <button className="ml-2 underline font-semibold" onClick={() => void load(search)}>
            Coba lagi
          </button>
        </div>
      )}

      {/* Guide or Document Grid */}
      {view === "guide" ? (
        <Card className="p-6">
          <h2 className="text-lg font-semibold text-[#25372a]">Alur Kerja Nara</h2>
          <div className="mt-5 grid gap-4 md:grid-cols-3">
            {[
              ["01", "Tuangkan ide", "Ceritakan produk dan masalah yang ingin diselesaikan."],
              ["02", "Klarifikasi bersama AI", "Jawab pertanyaan yang membantu memperjelas kebutuhan."],
              ["03", "Edit dan bagikan", "Simpan versi, jalankan AI actions, dan ekspor dokumen."],
            ].map(([n, t, d]) => (
              <div key={n} className="rounded-xl bg-[#f8f8f4] p-4">
                <span className="text-xs font-bold text-[#789176]">{n}</span>
                <h3 className="mt-3 font-semibold text-[#26372a]">{t}</h3>
                <p className="mt-1 text-sm leading-6 text-[#879287]">{d}</p>
              </div>
            ))}
          </div>
        </Card>
      ) : (
        <div className="grid gap-6 xl:grid-cols-[1.35fr_.85fr]">
          {/* Recent documents card */}
          <Card className="overflow-hidden">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[#edf0ea] p-5 md:p-6">
              <div>
                <h2 className="font-semibold text-[#25372a]">Dokumen terbaru</h2>
                <p className="mt-0.5 text-xs text-[#98a197]">Lanjutkan pekerjaan terakhirmu</p>
              </div>
              <Link
                href="/documents"
                className="inline-flex items-center gap-1 text-xs font-semibold text-[#5f7a62] hover:underline"
              >
                Semua dokumen <ArrowUpRight size={13} />
              </Link>
            </div>

            {summary.recentDocuments.length ? (
              <div className="divide-y divide-[#f0f2ed]">
                {summary.recentDocuments.slice(0, 5).map((doc) => (
                  <Link
                    key={doc.prdId}
                    href={`/prd/${doc.prdId}`}
                    className="flex items-center gap-3 px-5 py-4 transition hover:bg-[#fbfcf9] md:px-6"
                  >
                    <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-[#f0f4ee] text-[#648067]">
                      <FileText size={18} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-semibold text-[#25372a]">
                        {doc.title}
                      </span>
                      <span className="mt-0.5 block truncate text-xs text-[#96a095]">
                        {doc.projectName} · {formatDate(doc.updatedAt)}
                      </span>
                    </span>
                    <Badge tone={tone(doc.status)}>{doc.status.replace("_", " ")}</Badge>
                    <ArrowRight className="hidden text-[#adb7ad] sm:block" size={15} />
                  </Link>
                ))}
              </div>
            ) : (
              <div className="p-8 text-center">
                <span className="mx-auto grid size-11 place-items-center rounded-2xl bg-[#f0f4ee] text-[#718b72]">
                  <FileText size={20} />
                </span>
                <h3 className="mt-3 text-sm font-semibold text-[#25372a]">Belum ada dokumen</h3>
                <p className="mt-1 text-xs text-[#98a197]">Mulai dari ide produk pertamamu.</p>
                <Link href="/projects/new">
                  <Button size="sm" className="mt-4">
                    Buat PRD pertama <ArrowRight size={13} />
                  </Button>
                </Link>
              </div>
            )}
          </Card>

          {/* Projects list card */}
          <Card className="overflow-hidden">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[#edf0ea] p-5 md:p-6">
              <div>
                <h2 className="font-semibold text-[#25372a]">Proyekmu</h2>
                <p className="mt-0.5 text-xs text-[#98a197]">Semua ruang ide dalam satu tempat</p>
              </div>
              <Link
                href="/projects/new"
                className="grid size-8 place-items-center rounded-lg bg-[#eaf0e5] text-[#537354] hover:bg-[#dfe9d9] transition"
                aria-label="Buat proyek baru"
              >
                <Plus size={17} />
              </Link>
            </div>

            <div className="px-5 pt-4 md:px-6">
              <div className="relative">
                <Search
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-[#a4ada3]"
                  size={15}
                />
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Cari proyek…"
                  className="h-10 pl-9 text-xs"
                />
              </div>
            </div>

            <div className="p-3">
              {projects.map((project) => (
                <Link
                  key={project.id}
                  href={`/projects/${project.id}`}
                  className="flex items-center gap-3 rounded-xl p-3 hover:bg-[#f8f9f5] transition"
                >
                  <span className="grid size-9 place-items-center rounded-xl bg-[#f1eee5] text-[#8d825f]">
                    <FolderKanban size={16} />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs font-semibold text-[#25372a]">
                      {project.name}
                    </span>
                    <span className="mt-0.5 block text-[10px] text-[#9aa49a]">
                      Diperbarui {formatDate(project.updatedAt)}
                    </span>
                  </span>
                  <span className="size-2 rounded-full bg-[#a1bd96]" />
                </Link>
              ))}

              {!projects.length && !loading && (
                <div className="px-4 py-6 text-center text-xs text-[#98a197]">
                  {search ? "Proyek tidak ditemukan." : "Belum ada proyek."}
                </div>
              )}
            </div>

            <Link
              href="/projects/new"
              className="flex items-center justify-center gap-2 border-t border-[#edf0ea] p-4 text-xs font-semibold text-[#628066] hover:bg-[#fbfcf9] transition"
            >
              <Plus size={14} /> Proyek baru
            </Link>
          </Card>
        </div>
      )}

      <div className="flex items-center justify-center gap-2 pb-2 text-[10px] text-[#a0a99f]">
        <Clock3 size={12} /> Workspace pribadi · Perubahan tersimpan otomatis
      </div>
    </div>
  );
}
