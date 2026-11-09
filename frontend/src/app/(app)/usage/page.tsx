"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ArrowUpRight, Coins, Sparkles, Zap } from "lucide-react";
import { api } from "@/lib/api";
import { Badge, Card } from "@/components/ui";
import { formatDate } from "@/lib/utils";

type Usage = {
  period: string;
  plan: string;
  monthlyAILimit: number;
  remaining: number;
  aiRequests: number;
  generations: number;
  inputTokens: number;
  outputTokens: number;
};

type Event = {
  id: string;
  model: string;
  action: string;
  inputTokens: number;
  outputTokens: number;
  createdAt: string;
};

export default function UsagePage() {
  const [usage, setUsage] = useState<Usage | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([api.get<Usage>("/usage"), api.get<Event[]>("/usage/history?limit=50")])
      .then(([u, e]) => {
        setUsage(u);
        setEvents(e || []);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Data penggunaan belum tersedia."))
      .finally(() => setLoading(false));
  }, []);

  const used = usage ? Math.max(0, usage.monthlyAILimit - usage.remaining) : 0;
  const percent = usage?.monthlyAILimit ? Math.min(100, (used / usage.monthlyAILimit) * 100) : 0;

  return (
    <div className="mx-auto max-w-5xl pb-12">
      <div className="inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-[.16em] text-[#8b9a87]">
        <Sparkles size={13} /> WORKSPACE
      </div>
      <h1 className="mt-3 text-3xl font-semibold tracking-[-.045em] text-[#24372a]">
        Penggunaan AI
      </h1>
      <p className="mt-1 text-sm text-[#829083]">
        Pantau kuota dan aktivitas AI di workspace Anda.
      </p>

      {error && (
        <p role="alert" className="mt-5 rounded-xl bg-red-50 p-4 text-sm text-red-700">
          {error}
        </p>
      )}

      <div className="mt-7 grid gap-4 md:grid-cols-[1.2fr_.8fr]">
        <Card className="p-6 md:p-7">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[.13em] text-[#8d988d]">
                Kuota Bulan Ini
              </p>
              <h2 className="mt-2 text-2xl font-semibold text-[#25372a]">
                {loading ? "—" : usage?.plan || "Free"} <Badge tone="green">Aktif</Badge>
              </h2>
            </div>
            <span className="grid size-11 place-items-center rounded-2xl bg-[#eaf0e5] text-[#628066]">
              <Zap size={20} />
            </span>
          </div>

          <div className="mt-8 flex items-end justify-between">
            <p className="text-sm text-[#748075]">Permintaan AI Digunakan</p>
            <p className="text-sm font-semibold text-[#25372a]">
              {loading ? "—" : usage?.aiRequests}{" "}
              <span className="font-normal text-[#9ba49b]">/ {usage?.monthlyAILimit}</span>
            </p>
          </div>

          <div className="mt-3 h-2 overflow-hidden rounded-full bg-[#edf0ea]">
            <div
              className="h-full rounded-full bg-[#527d55] transition-all duration-300"
              style={{ width: `${percent}%` }}
            />
          </div>

          <div className="mt-3 flex justify-between text-xs text-[#939e93]">
            <span>{loading ? "Memuat…" : `${usage?.remaining} permintaan tersisa`}</span>
            <span>Reset otomatis setiap awal bulan</span>
          </div>
        </Card>

        <Card className="p-6 md:p-7">
          <p className="text-xs font-semibold uppercase tracking-[.13em] text-[#8d988d]">
            Ringkasan Token AI
          </p>
          <div className="mt-5 grid grid-cols-2 gap-3">
            <div className="rounded-xl bg-[#f7f8f4] p-4 border border-[#eff1ec]">
              <p className="text-xs text-[#8d988d]">Input Tokens</p>
              <p className="mt-2 text-xl font-semibold text-[#25372a]">
                {(usage?.inputTokens || 0).toLocaleString("id-ID")}
              </p>
            </div>
            <div className="rounded-xl bg-[#f7f8f4] p-4 border border-[#eff1ec]">
              <p className="text-xs text-[#8d988d]">Output Tokens</p>
              <p className="mt-2 text-xl font-semibold text-[#25372a]">
                {(usage?.outputTokens || 0).toLocaleString("id-ID")}
              </p>
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-xs text-[#8f9a8f]">
            <Coins size={14} /> Dihitung berdasarkan token penggunaan API
          </div>
        </Card>
      </div>

      <div className="mt-8 flex items-end justify-between">
        <div>
          <h2 className="font-semibold text-[#25372a]">Riwayat Penggunaan</h2>
          <p className="mt-0.5 text-xs text-[#96a095]">Aktivitas AI terbaru di akun Anda</p>
        </div>
        <Link
          href="/pricing"
          className="text-xs font-semibold text-[#577a5b] hover:underline inline-flex items-center gap-1"
        >
          Tentang paket <ArrowUpRight size={12} />
        </Link>
      </div>

      <Card className="mt-4 overflow-hidden">
        {events.length ? (
          <div className="divide-y divide-[#f0f2ed]">
            {events.map((event) => (
              <div key={event.id} className="flex items-center gap-3 px-5 py-4">
                <span className="grid size-9 place-items-center rounded-xl bg-[#f0f4ee] text-[#69856b]">
                  <Sparkles size={16} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-semibold capitalize text-[#25372a]">
                    {event.action.replaceAll("_", " ")}
                  </p>
                  <p className="mt-0.5 text-xs text-[#98a197]">
                    Model: {event.model} · {formatDate(event.createdAt)}
                  </p>
                </div>
                <span className="text-right text-xs text-[#889488]">
                  <span className="block font-semibold text-[#48634c]">
                    {(event.inputTokens + event.outputTokens).toLocaleString("id-ID")}
                  </span>{" "}
                  token
                </span>
              </div>
            ))}
          </div>
        ) : (
          <div className="p-8 text-center text-sm text-[#8c978c]">
            {loading ? "Memuat riwayat…" : "Belum ada aktivitas AI."}
          </div>
        )}
      </Card>

      <p className="mt-5 text-xs leading-5 text-[#a0a99f]">
        Paket Free memiliki kuota 10 permintaan AI setiap bulan. Paket Pro tersedia hingga 1.000
        permintaan setiap bulan.
      </p>
    </div>
  );
}
