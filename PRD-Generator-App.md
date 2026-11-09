# PROJECT: AI-Powered PRD Generator

## 1. Role

Bertindak sebagai Senior Full-Stack Engineer, AI Engineer, Software Architect, dan UI/UX Designer yang berpengalaman membangun aplikasi SaaS production-ready.

Bangun aplikasi web bernama **AI PRD Generator**, yaitu aplikasi berbasis Artificial Intelligence yang membantu pengguna mengubah ide produk mentah menjadi Product Requirements Document (PRD) yang terstruktur, profesional, dan siap digunakan oleh developer, product manager, UI/UX designer, maupun tim startup.

Jangan hanya membuat prototype atau landing page. Implementasikan aplikasi full-stack dengan fitur yang berfungsi, database persisten, integrasi AI, autentikasi, validasi, error handling, testing, dokumentasi, dan deployment configuration.

## 2. Technology Stack

### Frontend
- Next.js dengan App Router.
- TypeScript dengan strict mode.
- Tailwind CSS.
- shadcn/ui untuk komponen antarmuka.
- React Hook Form dan Zod untuk validasi form.
- TanStack Query jika diperlukan untuk server state.
- Markdown renderer untuk menampilkan hasil PRD.
- Responsive design untuk desktop, tablet, dan mobile.

### Backend
- Golang sebagai bahasa utama backend.
- Gin sebagai HTTP framework.
- Clean Architecture dengan pemisahan domain, use case, repository, dan delivery layer.
- REST API dengan versioning `/api/v1`.
- PostgreSQL sebagai database utama.
- pgx/v5 untuk akses database.
- SQL migrations menggunakan golang-migrate atau tool setara.
- JWT access token dan mekanisme refresh token yang aman.
- Redis untuk caching, rate limiting, atau background processing jika diperlukan.
- Swagger/OpenAPI untuk dokumentasi API.

### AI Integration
- Gunakan provider AI yang mendukung API untuk menghasilkan PRD.
- Implementasikan abstraction layer `AIProvider` agar provider dapat diganti tanpa mengubah business logic.
- API key disimpan hanya di environment backend, tidak pernah di frontend.
- Dukung konfigurasi model melalui environment variables.
- Terapkan timeout, retry terbatas, context cancellation, token usage tracking, dan error handling.
- Jangan mengarang hasil AI ketika provider gagal. Tampilkan pesan error yang jelas dan sediakan mekanisme retry.

### Infrastructure
- Docker dan Docker Compose.
- PostgreSQL dan Redis melalui Docker Compose.
- `.env.example` tanpa secret asli.
- CI pipeline untuk linting dan testing jika sesuai.
- Siapkan konfigurasi deployment untuk frontend dan backend secara terpisah.

## 3. Product Goals

Tujuan aplikasi:
1. Membantu pengguna menyusun PRD tanpa harus menulis dari nol.
2. Mengubah ide sederhana menjadi kebutuhan produk yang dapat dipahami developer.
3. Menghasilkan dokumen yang konsisten dan dapat diedit.
4. Memungkinkan pengguna menyimpan, mengelola, memperbarui, dan mengekspor dokumen.
5. Menjadi fondasi produk SaaS dengan sistem akun dan kuota penggunaan AI.

Target pengguna:
- Founder dan startup.
- Product Manager.
- Software Engineer.
- UI/UX Designer.
- Mahasiswa informatika.
- Developer yang menggunakan AI coding tools.

## 4. Core Features

### A. Authentication
Implementasikan:
- Register.
- Login.
- Logout.
- Refresh token.
- Get current user.
- Edit profil.
- Ganti password.
- Proteksi endpoint privat.
- Password hashing menggunakan Argon2id atau bcrypt.
- Validasi email dan password.
- Rate limiting pada endpoint autentikasi.

Gunakan cookie HttpOnly, Secure pada production, dan konfigurasi SameSite yang sesuai. Jangan menyimpan token sensitif di localStorage.

### B. Product Idea Input Wizard

Buat wizard bertahap yang mudah digunakan.

Input minimal:
- Nama produk.
- Deskripsi ide.
- Masalah yang ingin diselesaikan.
- Target pengguna.
- Tujuan bisnis.
- Platform: Web, Mobile, API, atau Multi-platform.
- Jenis produk.
- Fitur utama.
- Preferensi teknologi, jika diketahui.
- Batasan anggaran atau waktu, jika relevan.
- Bahasa dokumen.
- Tingkat detail PRD.

Pengguna tidak wajib mengetahui seluruh detail teknis. AI harus membantu melengkapi informasi yang belum tersedia dengan menandai asumsi secara eksplisit.

### C. AI Clarification Interview

Sebelum menghasilkan PRD, AI harus dapat mengajukan pertanyaan klarifikasi yang relevan untuk mengurangi ambiguitas.

Fitur:
- Menghasilkan pertanyaan berdasarkan ide produk.
- Menampilkan pertanyaan secara bertahap.
- Mendukung pilihan jawaban dan jawaban bebas.
- Memungkinkan pengguna melewati pertanyaan opsional.
- Menyimpan jawaban.
- Memperbarui pemahaman AI berdasarkan jawaban.
- Menentukan apakah informasi sudah cukup untuk menghasilkan PRD.

Hindari pertanyaan yang berulang atau tidak relevan. Batasi jumlah pertanyaan dan sediakan opsi untuk langsung membuat draft dengan asumsi yang jelas.

### D. PRD Generation Engine

Hasilkan PRD terstruktur yang minimal mencakup:

1. Executive Summary.
2. Product Overview.
3. Problem Statement.
4. Product Vision and Goals.
5. Target Audience dan User Personas.
6. User Problems dan Use Cases.
7. User Stories.
8. Functional Requirements.
9. Non-Functional Requirements.
10. Feature Specifications.
11. Acceptance Criteria.
12. User Journey.
13. Scope dan Out of Scope.
14. Business Rules.
15. Data Requirements.
16. API Requirements jika relevan.
17. High-Level Technical Architecture.
18. Security dan Privacy Considerations.
19. Analytics dan Success Metrics.
20. Dependencies dan Constraints.
21. Risks dan Mitigation.
22. Milestones dan Development Roadmap.
23. Open Questions.
24. Assumptions.

Setiap requirement harus memiliki ID unik, deskripsi, prioritas, dan acceptance criteria yang dapat diuji.

Gunakan prioritas seperti P0, P1, dan P2. Bedakan fakta dari input pengguna, asumsi AI, dan rekomendasi AI.

Jangan menyajikan keputusan teknis atau estimasi waktu sebagai fakta jika belum memiliki dasar yang cukup.

### E. PRD Editor

Sediakan editor dokumen dengan fitur:
- Mengedit isi PRD.
- Mengedit section secara terpisah.
- Menambah dan menghapus section.
- Regenerate section tertentu.
- Memperbaiki atau memperluas konten dengan AI.
- Menyederhanakan bahasa.
- Mengubah bahasa dokumen.
- Mengubah tingkat detail.
- Undo/redo jika memungkinkan.
- Autosave dengan indikator status penyimpanan.
- Version history.
- Membandingkan versi.
- Memulihkan versi sebelumnya.

Perubahan hasil AI tidak boleh langsung menimpa konten pengguna tanpa konfirmasi jika perubahan tersebut berpotensi menghilangkan informasi penting.

### F. AI Actions

Sediakan aksi AI kontekstual:
- Improve PRD.
- Find Missing Requirements.
- Detect Ambiguous Requirements.
- Generate User Stories.
- Generate Acceptance Criteria.
- Identify Edge Cases.
- Review Security Requirements.
- Generate Development Tasks.
- Create MVP Scope.
- Estimate Complexity secara indikatif.
- Generate API Contract Draft.
- Generate Database Schema Draft.

Setiap aksi harus beroperasi pada dokumen atau section yang relevan dan menjelaskan hasilnya.

### G. Project Dashboard

Buat dashboard yang menampilkan:
- Total proyek.
- Total PRD.
- Dokumen terbaru.
- Draft yang belum selesai.
- Aktivitas terbaru.
- Tombol Create New PRD.
- Pencarian.
- Filter berdasarkan status dan tanggal.
- Sorting.
- Pagination.

Pengguna dapat membuat, melihat, mengedit, menduplikasi, mengarsipkan, dan menghapus proyek atau PRD.

### H. Export and Sharing

Implementasikan:
- Export Markdown.
- Export PDF.
- Export DOCX jika memungkinkan.
- Print-friendly document.
- Copy section atau seluruh PRD.
- Share link dengan izin akses yang jelas jika fitur sharing diaktifkan.

Dokumen privat tidak boleh dapat diakses pengguna lain tanpa izin. Untuk share link, dukung pencabutan akses dan token yang sulit ditebak.

### I. Usage Limits

Siapkan fondasi penggunaan berbasis kuota:
- Jumlah generasi AI.
- Token input dan output.
- Penggunaan per user.
- Batas request per periode.
- Usage history.
- Estimasi biaya jika provider memberikan data yang memadai.

Kuota harus divalidasi di backend, bukan hanya disembunyikan melalui UI. Buat konfigurasi plan Free dan Pro, tetapi jangan mengaktifkan pembayaran sungguhan sebelum integrasi billing selesai.

## 5. UI/UX Requirements

Gunakan desain SaaS modern yang profesional, minimalis, dan nyaman digunakan.

Referensi visual:
- Linear untuk kerapian dashboard.
- Notion untuk pengalaman mengelola dokumen.
- Vercel untuk visual yang sederhana.
- Produk AI writing tools untuk interaksi generasi konten.

Jangan menyalin branding atau aset produk lain.

Halaman yang dibutuhkan:
- `/` — landing page.
- `/login` — login.
- `/register` — registrasi.
- `/dashboard` — dashboard.
- `/projects/new` — membuat proyek.
- `/projects/[projectId]` — detail proyek.
- `/prd/[prdId]` — editor PRD.
- `/settings` — pengaturan akun.
- `/usage` — penggunaan AI.
- `/pricing` — halaman plan, jika sistem plan diaktifkan.

Sediakan:
- Light mode dan dark mode.
- Loading skeleton.
- Empty states.
- Error states.
- Toast notification.
- Konfirmasi untuk tindakan destruktif.
- Accessible form dan keyboard navigation.
- Responsive layout.

Hindari membuat tombol dekoratif yang tidak melakukan apa pun.

## 6. Clean Architecture — Go Backend

Gunakan struktur direktori yang jelas dan konsisten, misalnya:

```text
backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── user.go
│   │   ├── project.go
│   │   ├── prd.go
│   │   ├── requirement.go
│   │   └── errors.go
│   ├── usecase/
│   │   ├── auth/
│   │   ├── project/
│   │   ├── prd/
│   │   ├── generation/
│   │   └── usage/
│   ├── repository/
│   │   └── postgres/
│   ├── delivery/
│   │   └── http/
│   │       ├── handler/
│   │       ├── middleware/
│   │       ├── request/
│   │       └── response/
│   ├── infrastructure/
│   │   ├── ai/
│   │   ├── database/
│   │   ├── cache/
│   │   ├── auth/
│   │   └── config/
│   └── platform/
├── migrations/
├── tests/
│   ├── unit/
│   └── integration/
├── Dockerfile
├── go.mod
└── README.md
```

Struktur ini merupakan titik awal dan boleh disesuaikan jika ada alasan arsitektural yang jelas.

Aturan arsitektur:
- Domain tidak bergantung pada Gin, PostgreSQL, Redis, atau provider AI.
- Use case mengatur business logic dan bergantung pada interface.
- Repository interface ditempatkan sesuai kebutuhan domain/use case.
- Implementasi PostgreSQL berada di lapisan infrastructure.
- Handler HTTP hanya menangani HTTP request/response dan validasi transport.
- Dependency injection dilakukan dari composition root.
- Hindari global mutable state.
- Gunakan `context.Context` untuk operasi I/O.
- Gunakan error wrapping dan error mapping yang konsisten.
- Hindari membuat abstraksi berlebihan yang tidak dibutuhkan.

## 7. Database Design

Rancang skema PostgreSQL untuk entitas minimal:

- users
- refresh_tokens atau sessions
- projects
- prds
- prd_versions
- prd_sections atau struktur JSONB yang tervalidasi
- ai_generation_jobs
- ai_usage_logs
- share_links jika sharing diaktifkan

Pertimbangkan:
- Primary key dan foreign key.
- Unique constraints.
- Index untuk query yang sering digunakan.
- `created_at` dan `updated_at`.
- Transaction untuk operasi yang harus atomik.
- Ownership validation.
- Optimistic locking atau mekanisme pencegahan konflik saat autosave.
- Soft delete hanya jika benar-benar dibutuhkan.

Gunakan migration versioned dan jangan mengandalkan auto-migration saat aplikasi production berjalan.

Simpan struktur PRD dalam format yang konsisten dan tervalidasi. Gunakan JSONB jika cocok, tetapi jangan menjadikan semua data sebagai satu kolom JSON besar tanpa pertimbangan query dan versioning.

## 8. REST API

Rancang endpoint versi pertama, antara lain:

Authentication:
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`

Projects:
- `POST /api/v1/projects`
- `GET /api/v1/projects`
- `GET /api/v1/projects/:projectId`
- `PATCH /api/v1/projects/:projectId`
- `DELETE /api/v1/projects/:projectId`

PRD:
- `POST /api/v1/projects/:projectId/prds`
- `GET /api/v1/prds/:prdId`
- `PATCH /api/v1/prds/:prdId`
- `DELETE /api/v1/prds/:prdId`
- `GET /api/v1/prds/:prdId/versions`
- `POST /api/v1/prds/:prdId/restore/:versionId`

AI:
- `POST /api/v1/ai/clarify`
- `POST /api/v1/ai/generate-prd`
- `POST /api/v1/ai/regenerate-section`
- `POST /api/v1/ai/review-prd`
- `POST /api/v1/ai/generate-tasks`

Usage:
- `GET /api/v1/usage`
- `GET /api/v1/usage/history`

Buat request dan response DTO yang konsisten, HTTP status code yang benar, pagination, validasi, serta format error standar.

Daftar di atas adalah rancangan awal. Implementasikan hanya endpoint yang benar-benar didukung oleh use case dan fitur yang telah dibuat.

## 9. AI Generation Architecture

Rancang alur berikut:

1. Pengguna memasukkan ide produk.
2. Backend memvalidasi input dan kepemilikan proyek.
3. Backend memeriksa rate limit dan kuota.
4. AI menganalisis kelengkapan informasi.
5. Backend mengembalikan pertanyaan klarifikasi atau membuat draft PRD.
6. AI menghasilkan output berdasarkan schema yang ditentukan.
7. Backend memvalidasi struktur dan kelengkapan hasil.
8. Backend menyimpan dokumen dan version history secara konsisten.
9. Usage log dicatat.
10. Frontend menampilkan hasil dan menyediakan editor.

Gunakan structured output atau JSON schema jika provider mendukungnya. Validasi hasil AI di backend. Tangani output tidak valid, timeout, kegagalan provider, dan pembatalan request.

Untuk generasi panjang, pertimbangkan background job dan status `pending`, `processing`, `completed`, serta `failed`. Gunakan idempotency key atau mekanisme serupa agar retry tidak menghasilkan duplikasi job atau pemotongan kuota yang tidak konsisten.

Jangan memasukkan data rahasia pengguna ke log. Terapkan batas panjang input dan output, serta jangan mempercayai instruksi yang muncul dari konten dokumen pengguna sebagai instruksi sistem.

## 10. Security and Reliability

Implementasikan:
- Input validation di backend.
- SQL injection prevention melalui parameterized query.
- Authentication dan authorization pada setiap resource privat.
- Proteksi IDOR melalui pemeriksaan kepemilikan resource.
- Rate limiting.
- CORS dengan allowlist origin.
- Secure headers.
- Secret management melalui environment variables.
- Timeout untuk request eksternal.
- Request size limit.
- Logging terstruktur tanpa secret.
- Graceful shutdown.
- Health dan readiness endpoints.
- Database transaction.
- Backup dan migration documentation.
- Unit test dan integration test.
- Dependensi yang terjaga dan dapat diperbarui.

Jangan menganggap penggunaan JWT saja sudah cukup untuk mengamankan aplikasi.

## 11. Testing Requirements

Backend:
- Unit test untuk use case.
- Repository integration test menggunakan PostgreSQL.
- HTTP handler test.
- Authentication dan authorization test.
- AI provider mock.
- Test quota enforcement.
- Test versioning dan autosave.
- Test concurrent updates.
- Test error handling dan timeout.

Frontend:
- Component test untuk form dan editor.
- API error handling test.
- Authentication flow test.
- E2E test untuk alur utama jika memungkinkan.

Semua test harus dapat dijalankan melalui perintah yang terdokumentasi.

## 12. Development Workflow

Kerjakan proyek secara bertahap:

Phase 1 — Requirements and Architecture
- Analisis kebutuhan.
- Tetapkan MVP.
- Buat arsitektur sistem.
- Buat ERD.
- Buat API contract.
- Identifikasi risiko dan asumsi.

Phase 2 — Foundation
- Setup repository frontend dan backend.
- Setup Docker Compose.
- Setup PostgreSQL dan migrations.
- Setup konfigurasi environment.
- Implementasikan health checks dan logging.

Phase 3 — Core Backend
- Authentication.
- Project CRUD.
- PRD CRUD.
- Repository dan use case.
- Authorization.
- Unit dan integration test.

Phase 4 — AI Engine
- Provider interface.
- Prompt templates yang terversi.
- Clarification flow.
- Structured PRD generation.
- Validation, quota, retry, dan usage tracking.

Phase 5 — Frontend
- Landing page.
- Authentication pages.
- Dashboard.
- Project wizard.
- PRD editor.
- Version history.
- Export dan usage page.

Phase 6 — Integration
- Integrasikan frontend dan backend.
- Validasi seluruh alur.
- Perbaiki bug.
- Uji error state dan security.
- Tambahkan E2E test.

Phase 7 — Deployment
- Docker build.
- Environment configuration.
- CI pipeline.
- Deployment documentation.
- Production checklist.
- Monitoring dan backup guidance.

## 13. Definition of Done

Proyek dianggap selesai apabila:

- Frontend dan backend berjalan secara terpisah.
- `docker compose up --build` dapat menjalankan lingkungan development.
- Database migrations berhasil.
- Pengguna dapat registrasi dan login.
- Pengguna dapat membuat dan mengelola proyek.
- Pengguna dapat menghasilkan PRD melalui provider AI yang dikonfigurasi.
- PRD tersimpan secara persisten.
- Pengguna dapat mengedit, menyimpan, dan mengembalikan versi dokumen.
- Dokumen dapat diekspor dalam format yang diimplementasikan.
- Resource privat terlindungi.
- Quota dan usage tracking bekerja di backend.
- Automated tests tersedia dan lulus.
- Tidak ada tombol utama yang sekadar menjadi placeholder.
- README mencakup setup, environment variables, migrasi, testing, troubleshooting, dan deployment.
- Tidak ada API key atau secret yang di-commit.
- Fitur yang belum selesai didokumentasikan secara jujur.

## 14. Instructions for the Coding Agent

Sebelum menulis kode:
1. Periksa repository dan file yang sudah ada.
2. Jangan menghapus atau menimpa pekerjaan sebelumnya tanpa alasan.
3. Identifikasi asumsi dan keputusan yang belum ditentukan.
4. Buat implementation plan dan checklist.
5. Tetapkan MVP yang dapat berjalan end-to-end.

Saat implementasi:
- Implementasikan kode nyata, bukan hanya pseudocode.
- Jangan membuat seluruh aplikasi dalam satu file.
- Jangan membuat endpoint palsu atau data dummy sebagai pengganti fitur inti.
- Jangan menampilkan klaim bahwa aplikasi sudah berjalan jika belum diuji.
- Jalankan formatter, linter, migration, dan test yang tersedia.
- Perbaiki error sebelum melanjutkan ke fase berikutnya.
- Dokumentasikan command yang benar-benar telah diverifikasi.
- Jika API key belum tersedia, buat interface dan mock untuk testing, lalu dokumentasikan cara menghubungkan provider sungguhan.
- Jika pekerjaan terlalu besar untuk satu sesi, selesaikan satu vertical slice yang berfungsi terlebih dahulu, lalu lanjutkan fase berikutnya.

Mulai dari audit repository, architecture plan, ERD, dan API contract. Setelah itu implementasikan MVP end-to-end dengan prioritas pada authentication, project management, AI PRD generation, persistent storage, dan editor. Lanjutkan dengan fitur tambahan secara bertahap.