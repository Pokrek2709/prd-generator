"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowLeft, ArrowRight, Sparkles } from "lucide-react";
import { api, type User } from "@/lib/api";
import { Button, FieldError, Input, Label } from "@/components/ui";

const schema = z.object({ name: z.string().trim().min(2, "Nama minimal 2 karakter.").max(100), email: z.string().email("Masukkan email yang valid."), password: z.string().min(10, "Password minimal 10 karakter.").max(72) });
type Form = z.infer<typeof schema>;
export default function RegisterPage() {
  const router = useRouter(); const [error, setError] = useState(""); const [loading, setLoading] = useState(false);
  const { register, handleSubmit, formState: { errors } } = useForm<Form>({ resolver: zodResolver(schema) });
  async function submit(values: Form) { setError(""); setLoading(true); try { await api.post<{ user: User }>("/auth/register", values); router.replace("/dashboard"); } catch (e) { setError(e instanceof Error ? e.message : "Tidak dapat membuat akun."); } finally { setLoading(false); } }
  return <><aside className="hidden w-[46%] flex-col justify-between bg-[#315642] p-12 text-white lg:flex"><Link href="/" className="flex items-center gap-3"><span className="grid size-10 place-items-center rounded-2xl bg-white/15"><Sparkles size={19}/></span><span className="text-base font-bold">nara.</span></Link><div className="max-w-md"><p className="text-sm text-[#d0ddce]">Lebih dari sekadar dokumen</p><h1 className="mt-5 text-5xl font-semibold leading-tight tracking-[-.045em]">Ruang yang membuat ide bergerak.</h1><p className="mt-5 leading-7 text-[#d0ddce]">Mulai workspace-mu dan biarkan AI membantumu merapikan langkah pertama.</p></div><p className="text-xs text-white/50">© Nara Workspace</p></aside><section className="flex flex-1 items-center justify-center px-6 py-12"><div className="w-full max-w-[390px]"><Link href="/" className="inline-flex items-center gap-2 text-xs font-semibold text-[#7d8a7d] hover:text-[#315642]"><ArrowLeft size={14}/> Kembali ke beranda</Link><div className="mt-10 lg:hidden"><span className="grid size-11 place-items-center rounded-2xl bg-[#315642] text-white"><Sparkles size={20}/></span></div><h2 className="mt-7 text-3xl font-semibold tracking-[-.04em]">Buat ruang untuk idemu.</h2><p className="mt-2 text-sm text-[#849084]">Mulai gratis. Lanjutkan kapan saja.</p><form onSubmit={handleSubmit(submit)} className="mt-8 space-y-4"><div><Label htmlFor="name">Nama</Label><Input id="name" autoComplete="name" placeholder="Nama lengkap" {...register("name")}/><FieldError>{errors.name?.message}</FieldError></div><div><Label htmlFor="email">Email</Label><Input id="email" type="email" autoComplete="email" placeholder="nama@email.com" {...register("email")}/><FieldError>{errors.email?.message}</FieldError></div><div><Label htmlFor="password">Password</Label><Input id="password" type="password" autoComplete="new-password" placeholder="Minimal 10 karakter" {...register("password")}/><FieldError>{errors.password?.message}</FieldError></div>{error && <p role="alert" className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p>}<Button type="submit" disabled={loading} className="w-full">{loading ? "Membuat akun…" : "Buat akun gratis"}<ArrowRight size={15}/></Button></form><p className="mt-7 text-center text-sm text-[#879287]">Sudah punya akun? <Link href="/login" className="font-semibold text-[#315642]">Masuk</Link></p></div></section></>;
}
