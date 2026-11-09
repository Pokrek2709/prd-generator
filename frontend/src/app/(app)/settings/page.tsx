"use client";

import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertCircle,
  Bot,
  Check,
  CheckCircle2,
  Cpu,
  Eye,
  EyeOff,
  KeyRound,
  LoaderCircle,
  Save,
  Sparkles,
  UserRound,
} from "lucide-react";
import { api, type AIProviderInfo, type User, type UserAISettings } from "@/lib/api";
import { Button, Card, FieldError, Input, Label, Select } from "@/components/ui";
import { useToast } from "@/components/toast";

const profileSchema = z.object({
  name: z.string().trim().min(2, "Nama minimal 2 karakter.").max(100),
});
type Profile = z.infer<typeof profileSchema>;

const passSchema = z.object({
  currentPassword: z.string().min(1, "Password saat ini wajib diisi."),
  newPassword: z.string().min(10, "Password baru minimal 10 karakter.").max(72),
});
type Password = z.infer<typeof passSchema>;

export default function SettingsPage() {
  const toast = useToast();

  const [user, setUser] = useState<User | null>(null);
  const [profileMessage, setProfileMessage] = useState("");
  const [passwordMessage, setPasswordMessage] = useState("");
  const [profileError, setProfileError] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [profileBusy, setProfileBusy] = useState(false);
  const [passwordBusy, setPasswordBusy] = useState(false);

  // AI Configuration State
  const [providers, setProviders] = useState<AIProviderInfo[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<string>("openai");
  const [selectedModel, setSelectedModel] = useState<string>("gpt-4o-mini");
  const [customModel, setCustomModel] = useState<string>("");
  const [apiKey, setApiKey] = useState<string>("");
  const [baseUrl, setBaseUrl] = useState<string>("");
  const [showApiKey, setShowApiKey] = useState(false);
  const [aiBusy, setAiBusy] = useState(false);
  const [aiTesting, setAiTesting] = useState(false);
  const [aiTestResult, setAiTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [aiMessage, setAiMessage] = useState("");
  const [aiError, setAiError] = useState("");

  const profile = useForm<Profile>({ resolver: zodResolver(profileSchema) });
  const password = useForm<Password>({ resolver: zodResolver(passSchema) });
  const resetProfile = profile.reset;

  useEffect(() => {
    api
      .get<User>("/auth/me")
      .then((u) => {
        setUser(u);
        resetProfile({ name: u.name });
      })
      .catch((e) => setProfileError(e instanceof Error ? e.message : "Profil tidak tersedia."));

    // Fetch AI Providers and current user settings
    Promise.all([api.getAIProviders(), api.getAISettings()])
      .then(([provs, settings]) => {
        setProviders(provs);
        if (settings) {
          setSelectedProvider(settings.provider || "openai");
          setSelectedModel(settings.model || "gpt-4o-mini");
          setCustomModel(settings.customModel || "");
          setApiKey(settings.apiKey || "");
          setBaseUrl(settings.baseUrl || "");
        }
      })
      .catch((err) => {
        console.error("Failed to load AI settings:", err);
      });
  }, [resetProfile]);

  const activeProviderInfo = providers.find((p) => p.id === selectedProvider);

  function handleProviderChange(providerId: string) {
    setSelectedProvider(providerId);
    setAiTestResult(null);
    setAiError("");
    setAiMessage("");

    const target = providers.find((p) => p.id === providerId);
    if (target) {
      setSelectedModel(target.defaultModel);
      setBaseUrl(target.defaultBaseUrl);
    }
  }

  async function saveProfile(values: Profile) {
    setProfileBusy(true);
    setProfileError("");
    setProfileMessage("");
    try {
      const updated = await api.patch<User>("/auth/profile", values);
      setUser(updated);
      setProfileMessage("Profil berhasil diperbarui.");
      toast.success("Profil berhasil diperbarui.");
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Perubahan tidak tersimpan.";
      setProfileError(msg);
      toast.error(msg);
    } finally {
      setProfileBusy(false);
    }
  }

  async function savePassword(values: Password) {
    setPasswordBusy(true);
    setPasswordError("");
    setPasswordMessage("");
    try {
      await api.put("/auth/password", values);
      password.reset();
      setPasswordMessage("Password berhasil diperbarui. Sesi lain telah dicabut.");
      toast.success("Password berhasil diperbarui.");
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Password tidak dapat diperbarui.";
      setPasswordError(msg);
      toast.error(msg);
    } finally {
      setPasswordBusy(false);
    }
  }

  async function saveAISettings(e: React.FormEvent) {
    e.preventDefault();
    setAiBusy(true);
    setAiError("");
    setAiMessage("");

    try {
      const payload: Partial<UserAISettings> = {
        provider: selectedProvider,
        model: selectedModel,
        baseUrl: baseUrl.trim(),
        apiKey: apiKey.trim(),
        customModel: customModel.trim(),
      };

      const updated = await api.updateAISettings(payload);
      setSelectedProvider(updated.provider);
      setSelectedModel(updated.model);
      setCustomModel(updated.customModel || "");
      setApiKey(updated.apiKey || "");
      setBaseUrl(updated.baseUrl || "");

      setAiMessage("Preferensi AI berhasil disimpan! Dokumen baru akan menggunakan model ini.");
      toast.success("Pengaturan engine AI tersimpan.");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Gagal menyimpan pengaturan AI.";
      setAiError(msg);
      toast.error(msg);
    } finally {
      setAiBusy(false);
    }
  }

  async function testAIConnection() {
    setAiTesting(true);
    setAiTestResult(null);
    setAiError("");

    try {
      const payload: Partial<UserAISettings> = {
        provider: selectedProvider,
        model: selectedModel,
        baseUrl: baseUrl.trim(),
        apiKey: apiKey.trim(),
        customModel: customModel.trim(),
      };

      const res = await api.testAISettings(payload);
      setAiTestResult({ success: true, message: res.message });
      toast.success("Koneksi AI berhasil!");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Koneksi ke AI provider gagal.";
      setAiTestResult({ success: false, message: msg });
      toast.error(msg);
    } finally {
      setAiTesting(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl pb-12">
      <div className="text-[10px] font-bold uppercase tracking-[.16em] text-[#8b9a87]">
        WORKSPACE
      </div>
      <h1 className="mt-3 text-3xl font-semibold tracking-[-.045em] text-[#24372a]">
        Pengaturan Akun & Engine AI
      </h1>
      <p className="mt-1 text-sm text-[#829083]">
        Kelola profil, konfigurasi model AI pilihan, dan keamanan akses akun Anda.
      </p>

      <div className="mt-7 grid gap-6">
        {/* Card: Engine & Model AI Preferences */}
        <Card className="p-5 md:p-7">
          <div className="flex items-center justify-between border-b border-[#edf0ea] pb-5">
            <div className="flex items-center gap-3">
              <span className="grid size-10 place-items-center rounded-xl bg-[#eaf0e5] text-[#315642]">
                <Cpu size={18} />
              </span>
              <div>
                <h2 className="font-semibold text-[#25372a]">Preferensi Engine & Model AI</h2>
                <p className="mt-0.5 text-xs text-[#929d92]">
                  Pilih provider AI dan sesuaikan model default yang digunakan saat menyusun dokumen PRD.
                </p>
              </div>
            </div>
            <span className="hidden rounded-full bg-[#eaf0e5] px-3 py-1 text-[11px] font-semibold text-[#3b6045] sm:inline-flex">
              Multi-Provider
            </span>
          </div>

          <form onSubmit={saveAISettings} className="mt-5 space-y-5">
            {/* Provider Selection */}
            <div>
              <Label htmlFor="aiProvider">Pilih Provider AI</Label>
              <Select
                id="aiProvider"
                value={selectedProvider}
                onChange={(e) => handleProviderChange(e.target.value)}
                className="font-medium"
              >
                {providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} {p.requiresApiKey ? "(Cloud API)" : "(Local / Free)"}
                  </option>
                ))}
              </Select>
              {activeProviderInfo && (
                <p className="mt-1.5 text-xs text-[#7d8c7e] leading-relaxed">
                  {activeProviderInfo.description}
                </p>
              )}
            </div>

            {/* Model Selection */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <Label htmlFor="aiModel">Pilih Model Default</Label>
                <Select
                  id="aiModel"
                  value={selectedModel}
                  onChange={(e) => {
                    setSelectedModel(e.target.value);
                    setAiTestResult(null);
                  }}
                  className="font-mono text-xs font-semibold"
                >
                  {activeProviderInfo?.models.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.name} {m.isDefault ? "· (Rekomendasi Bawaan)" : ""}
                    </option>
                  ))}
                  {(activeProviderInfo?.supportsCustom || activeProviderInfo?.models.length === 0) && (
                    <option value="custom">Kustom / Model Lainnya...</option>
                  )}
                </Select>
                {activeProviderInfo?.models.find((m) => m.id === selectedModel)?.description && (
                  <p className="mt-1.5 text-[11px] text-[#6d7e6e]">
                    💡 {activeProviderInfo.models.find((m) => m.id === selectedModel)?.description}
                  </p>
                )}
              </div>

              {/* Custom Model Input (if selected or supported) */}
              {(selectedModel === "custom" || activeProviderInfo?.supportsCustom) && (
                <div>
                  <Label htmlFor="customModel">Nama Model Kustom</Label>
                  <Input
                    id="customModel"
                    placeholder="Contoh: llama3.2:latest atau deepseek-r1:14b"
                    value={customModel}
                    onChange={(e) => setCustomModel(e.target.value)}
                    className="font-mono text-xs"
                  />
                  <p className="mt-1 text-[11px] text-[#859486]">
                    Ketik persis identifier model sesuai yang terpasang di provider Anda.
                  </p>
                </div>
              )}
            </div>

            {/* Base URL and API Key */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <Label htmlFor="baseUrl">Base URL Endpoint</Label>
                <Input
                  id="baseUrl"
                  placeholder="https://..."
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  className="font-mono text-xs"
                />
                <p className="mt-1 text-[11px] text-[#859486]">
                  Endpoint REST API (dapat disesuaikan jika menggunakan reverse proxy atau Ollama lokal).
                </p>
              </div>

              <div>
                <Label htmlFor="apiKey">API Key Pribadi</Label>
                <div className="relative">
                  <Input
                    id="apiKey"
                    type={showApiKey ? "text" : "password"}
                    placeholder={
                      selectedProvider === "ollama"
                        ? "Tidak memerlukan API Key"
                        : "Biarkan kosong untuk memakai key default sistem"
                    }
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    disabled={selectedProvider === "ollama"}
                    className="pr-10 font-mono text-xs"
                  />
                  {selectedProvider !== "ollama" && (
                    <button
                      type="button"
                      onClick={() => setShowApiKey(!showApiKey)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-[#7f8f81] hover:text-[#2d4233]"
                    >
                      {showApiKey ? <EyeOff size={16} /> : <Eye size={16} />}
                    </button>
                  )}
                </div>
                <p className="mt-1 text-[11px] text-[#859486]">
                  {selectedProvider === "ollama"
                    ? "Server Ollama lokal berjalan tanpa otentikasi API Key."
                    : "Tersimpan aman di database dengan enkripsi sesi akun Anda."}
                </p>
              </div>
            </div>

            {/* Test Connection Results */}
            {aiTestResult && (
              <div
                className={`flex items-start gap-2.5 rounded-xl border p-3.5 text-xs ${
                  aiTestResult.success
                    ? "border-[#b8ddbe] bg-[#f0f8f1] text-[#285732]"
                    : "border-[#f4d2d2] bg-[#fff5f5] text-[#932828]"
                }`}
              >
                {aiTestResult.success ? (
                  <CheckCircle2 size={16} className="mt-0.5 shrink-0 text-[#285732]" />
                ) : (
                  <AlertCircle size={16} className="mt-0.5 shrink-0 text-[#932828]" />
                )}
                <div>
                  <p className="font-semibold">
                    {aiTestResult.success ? "Koneksi Berhasil Diverifikasi" : "Verifikasi Koneksi Gagal"}
                  </p>
                  <p className="mt-0.5 opacity-90">{aiTestResult.message}</p>
                </div>
              </div>
            )}

            {aiError && (
              <p role="alert" className="text-xs text-red-700">
                {aiError}
              </p>
            )}
            {aiMessage && (
              <p role="status" className="flex items-center gap-1.5 text-xs text-[#315642]">
                <Check size={14} /> {aiMessage}
              </p>
            )}

            <div className="flex flex-wrap items-center gap-3 pt-2">
              <Button type="submit" disabled={aiBusy || aiTesting}>
                {aiBusy ? (
                  <LoaderCircle className="animate-spin" size={15} />
                ) : (
                  <Save size={15} />
                )}{" "}
                Simpan Preferensi AI
              </Button>

              <Button
                type="button"
                variant="secondary"
                disabled={aiTesting || aiBusy}
                onClick={testAIConnection}
              >
                {aiTesting ? (
                  <>
                    <LoaderCircle className="animate-spin" size={15} /> Menguji Koneksi…
                  </>
                ) : (
                  <>
                    <Sparkles size={15} /> Tes Koneksi Provider
                  </>
                )}
              </Button>
            </div>
          </form>
        </Card>

        {/* Card: Profile Info */}
        <Card className="p-5 md:p-7">
          <div className="flex items-center gap-3 border-b border-[#edf0ea] pb-5">
            <span className="grid size-10 place-items-center rounded-xl bg-[#eaf0e5] text-[#628066]">
              <UserRound size={18} />
            </span>
            <div>
              <h2 className="font-semibold text-[#25372a]">Informasi Profil</h2>
              <p className="mt-0.5 text-xs text-[#929d92]">
                Email login: {user?.email || "Memuat…"}
              </p>
            </div>
          </div>

          <form onSubmit={profile.handleSubmit(saveProfile)} className="mt-5 max-w-lg">
            <Label htmlFor="name">Nama Lengkap</Label>
            <Input id="name" autoComplete="name" {...profile.register("name")} />
            <FieldError>{profile.formState.errors.name?.message}</FieldError>

            {profileError && (
              <p role="alert" className="mt-3 text-sm text-red-700">
                {profileError}
              </p>
            )}
            {profileMessage && (
              <p role="status" className="mt-3 flex items-center gap-2 text-sm text-[#46704f]">
                <Check size={15} />
                {profileMessage}
              </p>
            )}

            <Button className="mt-5" disabled={profileBusy}>
              {profileBusy ? (
                <LoaderCircle className="animate-spin" size={15} />
              ) : (
                <Save size={15} />
              )}{" "}
              Simpan Profil
            </Button>
          </form>
        </Card>

        {/* Card: Password Change */}
        <Card className="p-5 md:p-7">
          <div className="flex items-center gap-3 border-b border-[#edf0ea] pb-5">
            <span className="grid size-10 place-items-center rounded-xl bg-[#f2efe5] text-[#88794e]">
              <KeyRound size={18} />
            </span>
            <div>
              <h2 className="font-semibold text-[#25372a]">Ganti Password</h2>
              <p className="mt-0.5 text-xs text-[#929d92]">
                Gunakan minimal 10 karakter untuk keamanan akun Anda.
              </p>
            </div>
          </div>

          <form onSubmit={password.handleSubmit(savePassword)} className="mt-5 grid max-w-2xl gap-5 md:grid-cols-2">
            <div>
              <Label htmlFor="currentPassword">Password Saat Ini</Label>
              <Input
                id="currentPassword"
                type="password"
                autoComplete="current-password"
                {...password.register("currentPassword")}
              />
              <FieldError>{password.formState.errors.currentPassword?.message}</FieldError>
            </div>

            <div>
              <Label htmlFor="newPassword">Password Baru</Label>
              <Input
                id="newPassword"
                type="password"
                autoComplete="new-password"
                {...password.register("newPassword")}
              />
              <FieldError>{password.formState.errors.newPassword?.message}</FieldError>
            </div>

            <div className="md:col-span-2">
              {passwordError && (
                <p role="alert" className="text-sm text-red-700">
                  {passwordError}
                </p>
              )}
              {passwordMessage && (
                <p role="status" className="flex items-center gap-2 text-sm text-[#46704f]">
                  <Check size={15} />
                  {passwordMessage}
                </p>
              )}
              <Button disabled={passwordBusy} className="mt-2">
                {passwordBusy ? (
                  <LoaderCircle className="animate-spin" size={15} />
                ) : (
                  <KeyRound size={15} />
                )}{" "}
                Perbarui Password
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </div>
  );
}
