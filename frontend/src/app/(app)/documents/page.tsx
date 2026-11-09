"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { ArrowRight, FileText, Plus, Search } from "lucide-react";
import { api, type PRD } from "@/lib/api";
import { Badge, Button, Card, Input, Select } from "@/components/ui";
import { formatDate } from "@/lib/utils";

export default function DocumentsPage() {
  const [prds, setPrds] = useState<PRD[]>([]);
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");
  const [sort, setSort] = useState("updated_desc");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const limit = 12;

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams({
      q,
      sort,
      page: String(page),
      limit: String(limit),
    });
    if (status) params.set("status", status);
    if (from) params.set("dateFrom", from);
    if (to) params.set("dateTo", to);

    try {
      const result = await api.getPaged<PRD>(`/prds?${params.toString()}`);
      setPrds(result.data || []);
      setTotal(result.meta?.total || 0);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Dokumen belum dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, [q, status, sort, from, to, page]);

  useEffect(() => {
    const timer = setTimeout(() => void load(), 250);
    return () => clearTimeout(timer);
  }, [load]);

  const pages = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="mx-auto max-w-6xl pb-12">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div>
          <div className="text-[10px] font-bold uppercase tracking-[.16em] text-[#8b9a87]">
            WORKSPACE
          </div>
          <h1 className="mt-3 text-3xl font-semibold tracking-[-.045em] text-[#24372a]">
            Dokumen PRD
          </h1>
          <p className="mt-1 text-sm text-[#829083]">
            Cari, sortir, dan lanjutkan dokumen produk Anda.
          </p>
        </div>
        <Link href="/projects/new">
          <Button>
            <Plus size={16} /> Buat PRD baru
          </Button>
        </Link>
      </div>

      {/* Filter and Search Bar */}
      <Card className="mt-7 p-4">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-[1fr_145px_175px_145px_145px]">
          <div className="relative sm:col-span-2 xl:col-span-1">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 text-[#a4ada3]"
              size={15}
            />
            <Input
              value={q}
              onChange={(e) => {
                setQ(e.target.value);
                setPage(1);
              }}
              placeholder="Cari judul atau isi dokumen…"
              className="pl-9 text-xs"
            />
          </div>
          <Select
            aria-label="Status PRD"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
            className="text-xs"
          >
            <option value="">Semua status</option>
            <option value="draft">Draft</option>
            <option value="in_progress">In progress</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </Select>
          <Select
            aria-label="Urutkan dokumen"
            value={sort}
            onChange={(e) => setSort(e.target.value)}
            className="text-xs"
          >
            <option value="updated_desc">Terbaru diperbarui</option>
            <option value="updated_asc">Terlama diperbarui</option>
            <option value="created_desc">Dokumen terbaru dibuat</option>
            <option value="created_asc">Dokumen pertama dibuat</option>
            <option value="title_asc">Judul A–Z</option>
            <option value="title_desc">Judul Z–A</option>
          </Select>
          <Input
            aria-label="Tanggal mulai"
            type="date"
            value={from}
            onChange={(e) => {
              setFrom(e.target.value);
              setPage(1);
            }}
            className="text-xs"
          />
          <Input
            aria-label="Tanggal akhir"
            type="date"
            value={to}
            onChange={(e) => {
              setTo(e.target.value);
              setPage(1);
            }}
            className="text-xs"
          />
        </div>
      </Card>

      {error && (
        <div role="alert" className="mt-5 rounded-xl bg-red-50 p-4 text-sm text-red-700">
          {error}{" "}
          <button onClick={() => void load()} className="ml-2 underline font-semibold">
            Coba lagi
          </button>
        </div>
      )}

      {loading ? (
        <div className="mt-5 grid gap-3">
          {[1, 2, 3].map((n) => (
            <div key={n} className="h-20 animate-pulse rounded-2xl bg-white" />
          ))}
        </div>
      ) : prds.length ? (
        <Card className="mt-5 overflow-hidden">
          <div className="divide-y divide-[#f0f2ed]">
            {prds.map((prd) => (
              <Link
                href={`/prd/${prd.id}`}
                key={prd.id}
                className="flex items-center gap-3 px-5 py-4 transition hover:bg-[#fbfcf9] md:px-6"
              >
                <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-[#f0f4ee] text-[#648067]">
                  <FileText size={18} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-semibold text-[#25372a]">
                    {prd.title}
                  </span>
                  <span className="mt-0.5 block text-xs text-[#96a095]">
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
                <ArrowRight size={15} className="hidden text-[#adb7ad] sm:block" />
              </Link>
            ))}
          </div>
        </Card>
      ) : (
        <Card className="mt-5 px-6 py-14 text-center">
          <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-[#eef3eb] text-[#69856b]">
            <FileText size={20} />
          </span>
          <h2 className="mt-4 font-semibold text-[#25372a]">
            {q || status ? "Dokumen tidak ditemukan" : "Belum ada dokumen PRD"}
          </h2>
          <p className="mt-1 text-sm text-[#8d988d]">
            Coba sesuaikan kata kunci pencarian atau mulai dengan ide baru.
          </p>
          <Link href="/projects/new">
            <Button className="mt-5">
              <Plus size={15} /> Mulai dengan ide
            </Button>
          </Link>
        </Card>
      )}

      {!loading && total > 0 && (
        <div className="mt-5 flex items-center justify-between text-xs text-[#879287]">
          <span>
            {total} dokumen · halaman {page} dari {pages}
          </span>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="secondary"
              disabled={page <= 1}
              onClick={() => setPage((n) => n - 1)}
            >
              Sebelumnya
            </Button>
            <Button
              size="sm"
              variant="secondary"
              disabled={page >= pages}
              onClick={() => setPage((n) => n + 1)}
            >
              Berikutnya
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
