"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  BookOpen,
  ChevronDown,
  CircleHelp,
  FileText,
  FolderKanban,
  Gauge,
  LayoutDashboard,
  LogOut,
  Menu,
  Settings,
  Sparkles,
  X,
} from "lucide-react";
import { api, type User } from "@/lib/api";
import { initials } from "@/lib/utils";

const nav = [
  { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { label: "Proyek", href: "/projects", icon: FolderKanban },
  { label: "Dokumen PRD", href: "/documents", icon: FileText },
  { label: "Panduan", href: "/dashboard?view=guide", icon: BookOpen },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    api
      .get<User>("/auth/me")
      .then(setUser)
      .catch(() => router.replace("/login?next=" + encodeURIComponent(pathname)))
      .finally(() => setReady(true));
  }, [router, pathname]);

  async function logout() {
    try {
      await api.post("/auth/logout");
    } finally {
      router.replace("/login");
    }
  }

  if (!ready || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#f8f8f4]">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-[#d9e2d8] border-t-[#315642]" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f8f8f4] text-[#26372d]">
      {/* Sidebar for Desktop & Mobile */}
      <aside
        className={`fixed inset-y-0 left-0 z-40 flex w-[260px] flex-col border-r border-[#e9ece5] bg-[#fbfbf8] px-5 py-6 transition-transform lg:translate-x-0 ${
          mobileOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="flex items-center justify-between px-1">
          <Link href="/dashboard" className="flex items-center gap-3">
            <span className="grid size-10 place-items-center rounded-2xl bg-[#315642] text-white shadow-sm">
              <Sparkles size={19} />
            </span>
            <span>
              <span className="block text-[15px] font-bold tracking-tight">
                nara<span className="text-[#7f977f]">.</span>
              </span>
              <span className="block text-[10px] font-medium uppercase tracking-[.17em] text-[#9aa398]">
                product workspace
              </span>
            </span>
          </Link>
          <button
            onClick={() => setMobileOpen(false)}
            className="text-[#728075] hover:text-[#333] lg:hidden"
            aria-label="Tutup menu samping"
          >
            <X size={18} />
          </button>
        </div>

        <div className="mt-10 px-2 text-[10px] font-bold uppercase tracking-[.14em] text-[#a1aaa0]">
          Workspace
        </div>

        <nav className="mt-3 space-y-1">
          {nav.map(({ label, href, icon: Icon }) => {
            const active =
              href === "/dashboard"
                ? pathname === "/dashboard"
                : pathname === href || pathname.startsWith(`${href}/`);
            return (
              <Link
                key={label}
                href={href}
                onClick={() => setMobileOpen(false)}
                className={`flex h-11 items-center gap-3 rounded-xl px-3 text-sm font-medium transition ${
                  active
                    ? "bg-[#e8eee6] text-[#2d563d]"
                    : "text-[#738076] hover:bg-[#f0f2ed] hover:text-[#34523c]"
                }`}
              >
                <Icon size={17} strokeWidth={1.8} />
                {label}
              </Link>
            );
          })}
        </nav>

        <div className="mt-8 flex items-center justify-between px-2">
          <span className="text-[10px] font-bold uppercase tracking-[.14em] text-[#a1aaa0]">
            Proyek Anda
          </span>
          <Link
            href="/projects/new"
            className="grid size-7 place-items-center rounded-lg text-[#6e8070] hover:bg-[#edf1eb] transition"
            aria-label="Tambah proyek baru"
          >
            <span className="text-xl leading-none">+</span>
          </Link>
        </div>

        <div className="mt-3 rounded-2xl border border-[#ebeee8] bg-white p-3">
          <div className="flex items-center gap-2.5">
            <span className="size-2 rounded-full bg-[#9bb88e]" />
            <span className="truncate text-xs font-semibold text-[#2b4031]">Ruang ide</span>
          </div>
          <p className="mt-1.5 pl-[18px] text-[11px] text-[#9ba49b]">
            Susun konsep produk berikutnya
          </p>
        </div>

        <div className="mt-auto space-y-1 border-t border-[#e9ece5] pt-5">
          <Link
            href="/usage"
            className={`flex h-10 items-center gap-3 rounded-xl px-3 text-sm transition ${
              pathname === "/usage"
                ? "bg-[#e8eee6] text-[#2d563d]"
                : "text-[#738076] hover:bg-[#f0f2ed]"
            }`}
          >
            <Gauge size={17} /> Penggunaan AI
          </Link>
          <Link
            href="/settings"
            className={`flex h-10 items-center gap-3 rounded-xl px-3 text-sm transition ${
              pathname === "/settings"
                ? "bg-[#e8eee6] text-[#2d563d]"
                : "text-[#738076] hover:bg-[#f0f2ed]"
            }`}
          >
            <Settings size={17} /> Pengaturan
          </Link>
          <Link
            href="/dashboard?view=guide"
            className="flex h-10 w-full items-center gap-3 rounded-xl px-3 text-left text-sm text-[#738076] hover:bg-[#f0f2ed] transition"
          >
            <CircleHelp size={17} /> Panduan
          </Link>
        </div>

        <div className="mt-4 flex items-center gap-3 border-t border-[#e9ece5] pt-4">
          <span className="grid size-9 place-items-center rounded-full bg-[#dfe9d8] text-xs font-bold text-[#385440]">
            {initials(user.name)}
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate text-xs font-semibold text-[#25372a]">{user.name}</div>
            <div className="truncate text-[10px] text-[#9aa398]">{user.email}</div>
          </div>
          <button
            onClick={logout}
            title="Keluar dari akun"
            className="text-[#94a096] hover:text-[#315642] p-1 rounded-lg transition"
          >
            <LogOut size={16} />
          </button>
        </div>
      </aside>

      {/* Backdrop overlay for mobile */}
      {mobileOpen && (
        <button
          onClick={() => setMobileOpen(false)}
          aria-label="Tutup menu navigasi"
          className="fixed inset-0 z-30 bg-black/25 lg:hidden"
        />
      )}

      {/* Main page content area */}
      <main className="min-h-screen lg:pl-[260px]">
        <header className="sticky top-0 z-20 flex h-[68px] items-center justify-between border-b border-[#e9ece5] bg-[#f8f8f4]/90 px-5 backdrop-blur md:px-9">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setMobileOpen(true)}
              className="text-[#68786a] hover:text-[#333] lg:hidden"
              aria-label="Buka menu navigasi"
            >
              <Menu size={20} />
            </button>
            <span className="text-xs text-[#9aa398]">Workspace</span>
            <span className="text-xs text-[#c4cbc1]">/</span>
            <span className="text-xs font-semibold capitalize text-[#25372a]">
              {pathname.split("/")[1]?.replace("-", " ") || "Dashboard"}
            </span>
          </div>

          <div className="flex items-center gap-3">
            <Link
              href="/usage"
              className="hidden items-center gap-2 rounded-full border border-[#e4e9e1] bg-white px-3 py-1.5 text-[11px] font-semibold text-[#667669] hover:border-[#cbdbca] sm:flex transition"
            >
              <Sparkles size={13} className="text-[#64855a]" /> AI Workspace <ChevronDown size={12} />
            </Link>
            <span className="grid size-8 place-items-center rounded-full bg-[#dfe9d8] text-[10px] font-bold text-[#385440]">
              {initials(user.name)}
            </span>
          </div>
        </header>

        <div className="mx-auto max-w-[1440px] px-5 py-7 md:px-9 md:py-9">{children}</div>
      </main>
    </div>
  );
}
