# Git Tips — Branch & Commit Workflow

> Diringkas dari `WORKFLOW-GUIDE.md`. Baca §1 dan §4 di sana untuk konteks lengkap.

## Alur dasar

```
tulis task file di tasks/<ID>.md
  → buat branch: feat/<ID>-short-name
  → tulis test untuk "Done =" TERLEBIH DAHULU
  → implementasi sampai test hijau
  → buka PR (kecil, lihat aturan di bawah), isi "how to test" + bukti
  → 1 reviewer approve → merge
```

## Membuat branch

- Branch selalu dibuat dari task yang sudah punya file `tasks/<ID>.md`.
- Format nama: `feat/<ID>-short-name` (gunakan prefix lain sesuai jenis kerja, mis. `fix/`, `chore/`).
- Contoh:
  ```bash
  git checkout -b feat/S-014-login-page
  ```
- Satu branch = satu task. Jangan campur beberapa task dalam satu branch.

## Membuat commit

- Gunakan **Conventional Commits**: `<type>(<scope>): <description>`
  ```bash
  git commit -m "feat(auth): add refresh token rotation"
  ```
- Commit message akhir harus jelas menjelaskan perubahan, bukan sekadar "update" atau "fix".
- **Tidak ada push langsung ke `main`.** Selalu lewat branch + PR.

## Membuka PR

- **PR kecil**: batasi diff sekitar 300–400 baris agar review efektif.
- **Squash merge** saat merge ke `main` — judul PR menjadi commit message final, tetap pakai Conventional Commits.
- **Wajib ada bukti kerja**: screenshot/recording untuk perubahan UI, atau request/response + test output untuk perubahan API. PR tanpa bukti dikembalikan, bukan direview.
- **Minimal 1 reviewer** approve sebelum merge — rotasi siapa yang review agar semua orang membaca kode satu sama lain.

## Checklist singkat sebelum PR

- [ ] Branch dibuat dari task file (`tasks/<ID>.md`)
- [ ] Commit pakai Conventional Commits
- [ ] Diff kecil (~300–400 baris)
- [ ] Ada bukti kerja (screenshot / test output / request-response)
- [ ] Semua item "Done =" di task file tercentang
- [ ] Reviewer sudah ditunjuk, belum merge sendiri
