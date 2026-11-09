"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import ReactMarkdown from "react-markdown";
import { FileText, LoaderCircle, Sparkles } from "lucide-react";
import { api, type PRD } from "@/lib/api";
import { Badge, Card } from "@/components/ui";
import { formatDate } from "@/lib/utils";
import { MermaidChart } from "@/components/mermaid";

function toMarkdown(prd: PRD) {
  const content = prd.content || {};
  return Object.entries(content)
    .filter(
      ([key, value]) =>
        key !== "customSections" &&
        value !== undefined &&
        value !== null &&
        value !== "" &&
        (!Array.isArray(value) || value.length > 0)
    )
    .map(([key, value]) => {
      const title = key.replace(/([A-Z])/g, " $1").replace(/^./, (s) => s.toUpperCase());
      const body = Array.isArray(value)
        ? value
            .map((item) => {
              if (item && typeof item === "object") {
                const req = item as {
                  id?: string;
                  description?: string;
                  priority?: string;
                  acceptanceCriteria?: string[];
                };
                return `- **${req.id || ""} ${req.description || ""}**${
                  req.priority ? ` (${req.priority})` : ""
                }${
                  req.acceptanceCriteria?.length
                    ? `\n${req.acceptanceCriteria.map((c) => `  - ${c}`).join("\n")}`
                    : ""
                }`;
              }
              return `- ${String(item)}`;
            })
            .join("\n")
        : String(value);
      return `## ${title}\n\n${body}`;
    })
    .concat(
      Array.isArray(content.customSections)
        ? (content.customSections as Array<{ title: string; body: string }>).map(
            (section) => `## ${section.title}\n\n${section.body || ""}`
          )
        : []
    )
    .join("\n\n");
}

export default function SharedPRDPage() {
  const { token } = useParams<{ token: string }>();
  const [prd, setPrd] = useState<PRD | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .get<PRD>(`/shared/${encodeURIComponent(token)}`)
      .then(setPrd)
      .catch((e) =>
        setError(e instanceof Error ? e.message : "Link berbagi tidak valid atau sudah kedaluwarsa.")
      )
      .finally(() => setLoading(false));
  }, [token]);

  return (
    <main className="min-h-screen bg-[#f8f8f4] px-4 py-8 md:px-8 md:py-12">
      <div className="mx-auto max-w-4xl">
        <div className="flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-2xl bg-[#315642] text-white">
            <Sparkles size={18} />
          </span>
          <div>
            <p className="text-sm font-bold">
              nara<span className="text-[#809b7d]">.</span>
            </p>
            <p className="text-[10px] uppercase tracking-[.15em] text-[#9ba49b]">
              Dokumen yang dibagikan
            </p>
          </div>
        </div>

        {loading ? (
          <div className="flex min-h-[55vh] items-center justify-center text-[#758575]">
            <LoaderCircle className="animate-spin" />
          </div>
        ) : error ? (
          <Card className="mt-10 p-8 text-center">
            <FileText size={24} className="mx-auto text-[#8b9a87]" />
            <h1 className="mt-4 text-lg font-semibold">Dokumen tidak tersedia</h1>
            <p className="mt-2 text-sm text-[#829083]">{error}</p>
          </Card>
        ) : (
          prd && (
            <>
              <div className="mt-9 rounded-[1.6rem] bg-[#eaf0e5] p-6 md:p-9">
                <div className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-[.15em] text-[#748c70]">
                  <FileText size={13} /> Product requirements document
                </div>
                <h1 className="mt-4 text-3xl font-semibold tracking-[-.045em] md:text-4xl">
                  {prd.title}
                </h1>
                <p className="mt-3 text-xs text-[#829083]">
                  Versi {prd.version} · Diperbarui {formatDate(prd.updatedAt)}{" "}
                  <span className="mx-1">·</span>{" "}
                  <Badge tone={prd.status === "completed" ? "green" : "gray"}>
                    {prd.status.replace("_", " ")}
                  </Badge>
                </p>
              </div>

              <article className="prose-prd prose prose-sm mt-5 max-w-none rounded-2xl border border-[#e6ebe3] bg-white p-6 md:p-10">
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
                  {`# ${prd.title}\n\n${toMarkdown(prd)}`}
                </ReactMarkdown>
              </article>

              <p className="mt-5 text-center text-[10px] text-[#a0a99f]">
                Dibagikan secara privat oleh pemilik dokumen melalui Nara.
              </p>
            </>
          )
        )}
      </div>
    </main>
  );
}
