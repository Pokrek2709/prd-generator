"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  ArrowLeft,
  ArrowUpRight,
  Copy,
  FilePlus2,
  FolderKanban,
  Pencil,
  Plus,
  Save,
  Trash2,
  X,
} from "lucide-react";
import { api, type PRD, type Project } from "@/lib/api";
import { Badge, Button, Card } from "@/components/ui";
import { useToast } from "@/components/toast";
import { formatDate } from "@/lib/utils";

export default function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const router = useRouter();
  const toast = useToast();

  const [project, setProject] = useState<Project | null>(null);
  const [prds, setPrds] = useState<PRD[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");

  async function load() {
    setError("");
    try {
      const [p, d] = await Promise.all([
        api.get<Project>(`/projects/${projectId}`),
        api.get<PRD[]>(`/projects/${projectId}/prds?limit=100`),
      ]);
      setProject(p);
      setPrds(d || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Proyek tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [projectId]);

  async function createDraft() {
    setBusy(true);
    setError("");
    try {
      const p = await api.post<PRD>(`/projects/${projectId}/prds`, {
        title: "Dokumen PRD baru",
      });
      toast.success("Draft PRD baru berhasil dibuat.");
      router.push(`/prd/${p.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Draft tidak dapat dibuat.");
      toast.error("Gagal membuat draft baru.");
    } finally {
      setBusy(false);
    }
  }

  async function duplicate() {
    setBusy(true);
    try {
      const p = await api.post<Project>(`/projects/${projectId}/duplicate`);
      toast.success("Proyek berhasil diduplikasi.");
      router.push(`/projects/${p.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Proyek tidak dapat disalin.");
      toast.error("Gagal menduplikasi proyek.");
    } finally {
      setBusy(false);
    }
  }

  async function saveProject() {
    setBusy(true);
    setError("");
    try {
      const updated = await api.patch<Project>(`/projects/${projectId}`, {
        name: editName,
        description: editDescription,
      });
      setProject(updated);
      setEditing(false);
      toast.success("Detail proyek berhasil disimpan.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Perubahan proyek tidak tersimpan.");
      toast.error("Gagal menyimpan perubahan.");
    } finally {
      setBusy(false);
    }
  }

  async function archive() {
    if (!confirm("Arsipkan proyek ini? Proyek dapat ditemukan dengan filter arsip.")) return;
    setBusy(true);
    try {
      await api.post(`/projects/${projectId}/archive`);
      toast.info("Proyek berhasil diarsipkan.");
      router.push("/dashboard");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Proyek tidak dapat diarsipkan.");
      toast.error("Gagal mengarsipkan proyek.");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (
      !confirm(
        "Hapus proyek dan semua dokumen PRD di dalamnya? Tindakan ini tidak dapat dibatalkan."
      )
    )
      return;
    setBusy(true);
    try {
      await api.delete(`/projects/${projectId}`);
      toast.info("Proyek telah dihapus.");
      router.push("/dashboard");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Proyek tidak dapat dihapus.");
      toast.error("Gagal menghapus proyek.");
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return <div className="h-64 animate-pulse rounded-2xl bg-white" />;
  }

  if (!project) {
    return (
      <Card className="mx-auto max-w-xl p-8 text-center">
        <h1 className="text-lg font-semibold text-[#25372a]">Proyek tidak tersedia</h1>
        <p className="mt-2 text-sm text-[#849084]">{error}</p>
        <Link href="/dashboard">
          <Button variant="secondary" className="mt-5">
            Kembali ke dashboard
          </Button>
        </Link>
      </Card>
    );
  }

  return (
    <div className="mx-auto max-w-5xl pb-12">
      <Link
        href="/projects"
        className="inline-flex items-center gap-2 text-xs font-semibold text-[#809080] hover:text-[#315642] transition"
      >
        <ArrowLeft size={14} /> Semua proyek
      </Link>

      <div className="mt-6 flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
        <div className="min-w-0 flex-1">
          <div className="inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-[.16em] text-[#8b9a87]">
            <FolderKanban size={13} /> PROYEK
          </div>
          {editing ? (
            <div className="mt-3 max-w-2xl space-y-3">
              <input
                aria-label="Nama proyek"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                className="h-11 w-full rounded-xl border border-[#dfe5dc] bg-white px-3.5 text-lg font-semibold outline-none focus:border-[#62846c] focus:ring-4 focus:ring-[#315642]/10"
              />
              <textarea
                aria-label="Deskripsi proyek"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                className="min-h-20 w-full rounded-xl border border-[#dfe5dc] bg-white px-3.5 py-3 text-sm outline-none focus:border-[#62846c] focus:ring-4 focus:ring-[#315642]/10"
              />
            </div>
          ) : (
            <>
              <h1 className="mt-3 text-3xl font-semibold tracking-[-.045em] text-[#24372a]">
                {project.name}
              </h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-[#829083]">
                {project.description || "Belum ada deskripsi proyek."}
              </p>
            </>
          )}
          <p className="mt-3 text-xs text-[#a0a99f]">
            Diperbarui {formatDate(project.updatedAt)} <span className="mx-1">·</span>{" "}
            <Badge tone={project.status === "active" ? "green" : "gray"}>
              {project.status}
            </Badge>
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          {editing ? (
            <>
              <Button variant="secondary" onClick={() => setEditing(false)} disabled={busy}>
                <X size={14} /> Batal
              </Button>
              <Button onClick={() => void saveProject()} disabled={busy}>
                <Save size={14} /> Simpan
              </Button>
            </>
          ) : (
            <Button
              variant="secondary"
              onClick={() => {
                setEditName(project.name);
                setEditDescription(project.description);
                setEditing(true);
              }}
            >
              <Pencil size={14} /> Edit
            </Button>
          )}
          <Button variant="secondary" onClick={duplicate} disabled={busy}>
            <Copy size={15} /> Duplikat
          </Button>
          <Button variant="secondary" onClick={archive} disabled={busy}>
            Arsipkan
          </Button>
          <Button variant="ghost" onClick={remove} disabled={busy} aria-label="Hapus proyek">
            <Trash2 size={16} className="text-red-600" />
          </Button>
          <Button onClick={createDraft} disabled={busy}>
            <Plus size={16} /> Buat PRD
          </Button>
        </div>
      </div>

      {error && (
        <div role="alert" className="mt-6 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="mt-9 flex items-end justify-between border-b border-[#e5eae2] pb-3">
        <div>
          <h2 className="font-semibold text-[#25372a]">Dokumen PRD</h2>
          <p className="mt-0.5 text-xs text-[#96a095]">{prds.length} dokumen dalam proyek ini</p>
        </div>
        <button
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-[#4e7452] hover:underline"
          onClick={createDraft}
        >
          <FilePlus2 size={14} /> Dokumen baru
        </button>
      </div>

      {prds.length ? (
        <div className="mt-4 grid gap-3">
          {prds.map((prd) => (
            <Link href={`/prd/${prd.id}`} key={prd.id}>
              <Card className="flex items-center gap-4 p-4 transition hover:-translate-y-0.5 hover:shadow-soft md:p-5">
                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-[#eef3eb] text-[#69856b]">
                  <FilePlus2 size={19} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-semibold text-[#25372a]">
                    {prd.title}
                  </span>
                  <span className="mt-0.5 block text-xs text-[#98a197]">
                    Versi {prd.version} · Diperbarui {formatDate(prd.updatedAt)}
                  </span>
                </span>
                <Badge
                  tone={
                    prd.status === "completed"
                      ? "green"
                      : prd.status === "in_progress"
                      ? "amber"
                      : "gray"
                  }
                >
                  {prd.status.replace("_", " ")}
                </Badge>
                <ArrowUpRight size={16} className="text-[#a4ada3]" />
              </Card>
            </Link>
          ))}
        </div>
      ) : (
        <Card className="mt-4 px-6 py-12 text-center">
          <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-[#eef3eb] text-[#69856b]">
            <FilePlus2 size={20} />
          </span>
          <h3 className="mt-4 font-semibold text-[#25372a]">Belum ada dokumen PRD</h3>
          <p className="mt-1 text-sm text-[#8d988d]">
            Buat draft baru atau mulai dengan ide produk dari wizard.
          </p>
          <div className="mt-5 flex justify-center gap-2">
            <Button variant="secondary" onClick={createDraft}>
              <Plus size={15} /> Draft kosong
            </Button>
            <Link href="/projects/new">
              <Button>
                Mulai dengan AI <ArrowUpRight size={15} />
              </Button>
            </Link>
          </div>
        </Card>
      )}
    </div>
  );
}
