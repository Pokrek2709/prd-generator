"use client";

import { useEffect, useState } from "react";
import { Check, Code2, Copy, Eye, Loader2, AlertCircle } from "lucide-react";

interface MermaidProps {
  chart: string;
  className?: string;
}

export function MermaidChart({ chart, className = "" }: MermaidProps) {
  const [svg, setSvg] = useState<string>("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [showCode, setShowCode] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    let isMounted = true;

    async function renderChart() {
      const cleanChart = chart
        .replace(/^```(?:mermaid)?\r?\n?/i, "")
        .replace(/\r?\n?```$/i, "")
        .trim();

      if (!cleanChart) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const mermaid = (await import("mermaid")).default;
        mermaid.initialize({
          startOnLoad: false,
          theme: "neutral",
          themeVariables: {
            fontFamily: "inherit",
            primaryColor: "#eaf0e5",
            primaryTextColor: "#24372a",
            primaryBorderColor: "#62846c",
            lineColor: "#4f7556",
            secondaryColor: "#f7f8f4",
            tertiaryColor: "#ffffff",
          },
          securityLevel: "loose",
        });

        const uniqueId = `mermaid-${Math.random().toString(36).substring(2, 9)}`;
        const { svg: renderedSvg } = await mermaid.render(uniqueId, cleanChart);

        if (isMounted) {
          setSvg(renderedSvg);
          setLoading(false);
        }
      } catch (err: unknown) {
        if (isMounted) {
          const msg = err instanceof Error ? err.message : "Diagram Mermaid tidak dapat dirender.";
          // Remove internal DOM error elements injected by mermaid if any
          const errorElement = document.getElementById("d" + chart);
          if (errorElement) errorElement.remove();
          setError(msg);
          setLoading(false);
        }
      }
    }

    renderChart();

    return () => {
      isMounted = false;
    };
  }, [chart]);

  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(chart.trim());
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard write failed
    }
  };

  return (
    <div
      className={`my-6 overflow-hidden rounded-2xl border border-[#dfe5dc] bg-[#fafbf9] shadow-xs ${className}`}
    >
      <div className="flex items-center justify-between border-b border-[#e3e8e1] bg-[#f2f5f0] px-4 py-2.5 text-xs text-[#526657]">
        <div className="flex items-center gap-2 font-semibold">
          <span className="size-2 rounded-full bg-[#4f7556]" />
          Diagram Alur & Arsitektur
        </div>
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={() => setShowCode(!showCode)}
            className="inline-flex items-center gap-1 rounded-lg border border-[#d6ded4] bg-white px-2.5 py-1 text-[11px] font-medium text-[#465a4c] hover:bg-[#e8eee6] transition"
          >
            {showCode ? (
              <>
                <Eye size={12} /> Lihat Diagram
              </>
            ) : (
              <>
                <Code2 size={12} /> Lihat Kode
              </>
            )}
          </button>
          <button
            type="button"
            onClick={copyCode}
            className="inline-flex items-center gap-1 rounded-lg border border-[#d6ded4] bg-white px-2.5 py-1 text-[11px] font-medium text-[#465a4c] hover:bg-[#e8eee6] transition"
            title="Salin definisi Mermaid"
          >
            {copied ? (
              <>
                <Check size={12} className="text-[#315642]" /> Tersalin
              </>
            ) : (
              <>
                <Copy size={12} /> Salin
              </>
            )}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex min-h-[140px] items-center justify-center gap-2 text-xs text-[#6e8072]">
          <Loader2 size={16} className="animate-spin text-[#4f7556]" />
          Merender diagram interaktif...
        </div>
      ) : error ? (
        <div className="p-4">
          <div className="mb-3 flex items-start gap-2.5 rounded-xl border border-[#f3d3d3] bg-[#fff6f6] p-3 text-xs text-[#993434]">
            <AlertCircle size={16} className="mt-0.5 shrink-0" />
            <div>
              <p className="font-semibold">Diagram tidak dapat divisualisasikan</p>
              <p className="mt-0.5 text-[11px] opacity-90">
                Sintaks diagram belum sesuai dengan spesifikasi Mermaid. Anda tetap dapat meninjau kode mentahnya di bawah.
              </p>
            </div>
          </div>
          <pre className="overflow-x-auto rounded-xl border border-[#e1e6de] bg-white p-3 text-xs font-mono text-[#24372a]">
            {chart.trim()}
          </pre>
        </div>
      ) : showCode ? (
        <div className="p-4">
          <pre className="overflow-x-auto rounded-xl border border-[#e1e6de] bg-white p-3 text-xs font-mono text-[#24372a]">
            {chart.trim()}
          </pre>
        </div>
      ) : (
        <div
          className="flex justify-center overflow-x-auto p-4 md:p-6 [&_svg]:max-w-full [&_svg]:h-auto"
          dangerouslySetInnerHTML={{ __html: svg }}
        />
      )}
    </div>
  );
}
