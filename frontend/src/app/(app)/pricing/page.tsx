"use client";

import { useEffect, useState } from "react";
import { Check, Sparkles } from "lucide-react";
import { api } from "@/lib/api";
import { Badge, Button, Card } from "@/components/ui";

type Plan = {
  code: string;
  monthlyAILimit: number;
};

export default function PricingPage() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [usage, setUsage] = useState<{ plan: string } | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([api.get<Plan[]>("/plans"), api.get<{ plan: string }>("/usage")])
      .then(([p, u]) => {
        setPlans(p);
        setUsage(u);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Paket belum tersedia."));
  }, []);

  return (
    <div className="mx-auto max-w-4xl pb-12">
      <div className="text-[10px] font-bold uppercase tracking-[.16em] text-[#8b9a87]">
        PILIH RUANG KERJA ANDA
      </div>
      <h1 className="mt-3 text-3xl font-semibold tracking-[-.045em] text-[#24372a]">
        Paket yang jelas, tanpa kejutan.
      </h1>
      <p className="mt-1 text-sm text-[#829083]">
        Kuota dihitung di backend untuk menjaga keandalan generasi AI.
      </p>

      {error && <p role="alert" className="mt-4 text-sm text-red-700">{error}</p>}

      <div className="mt-8 grid gap-5 sm:grid-cols-2">
        {plans.map((plan) => {
          const isCurrent = usage?.plan === plan.code;
          return (
            <Card
              key={plan.code}
              className={`p-6 transition ${
                isCurrent ? "border-[#7fa27a] ring-2 ring-[#dfe9d9] shadow-md" : ""
              }`}
            >
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold capitalize text-[#25372a]">
                  Paket {plan.code.toLowerCase()}
                </h2>
                {isCurrent && <Badge tone="green">Paket aktif</Badge>}
              </div>

              <p className="mt-4 text-3xl font-semibold text-[#25372a]">
                {plan.monthlyAILimit.toLocaleString("id-ID")}
                <span className="ml-2 text-sm font-normal text-[#839084]">
                  permintaan AI / bulan
                </span>
              </p>

              <div className="mt-6 space-y-3 text-sm text-[#5d6e60]">
                <p className="flex items-center gap-2">
                  <Check size={15} className="text-[#55785a]" /> Akses AI PRD Generator & Clarifier
                </p>
                <p className="flex items-center gap-2">
                  <Check size={15} className="text-[#55785a]" /> Simpan dan kelola revisi dokumen
                </p>
                <p className="flex items-center gap-2">
                  <Check size={15} className="text-[#55785a]" /> Ekspor Markdown, PDF, dan DOCX
                </p>
                <p className="flex items-center gap-2">
                  <Check size={15} className="text-[#55785a]" /> Riwayat versi & perbandingan diff
                </p>
              </div>

              <Button
                className="mt-8 w-full"
                variant={isCurrent ? "secondary" : "primary"}
                disabled
              >
                <Sparkles size={14} />
                {isCurrent ? "Paket Anda Saat Ini" : "Pembayaran Segera Hadir"}
              </Button>
            </Card>
          );
        })}
      </div>

      <p className="mt-6 text-xs leading-5 text-[#9aa49a] text-center">
        Modul pembayaran (billing gateway) sedang dalam tahap pengembangan. Paket saat ini dapat
        dikonfigurasi oleh administrator sistem.
      </p>
    </div>
  );
}
