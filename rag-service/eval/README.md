# Surau RAG — Eval Harness

Ukur akurasi retrieval + citation + fidelitas jawaban RAG terhadap golden set
Q&A yang diisi manual dari kitab yang sudah ter-ingest.

## Kenapa ini penting

Tanpa eval harness, setiap perubahan (ganti embedding, ganti reranker, ubah
chunking) cuma **feeling**, bukan data. Harness ini yang membuat keputusan
"pindah ke Voyage" atau "buang tree reasoning" bisa didukung angka.

## Metrik yang diukur

Tiap Q&A menghasilkan sampai 4 metrik (diabaikan kalau field expected-nya kosong):

| Metrik | Skala | Arti |
|---|---|---|
| `citation_hit` | 0/1 | ≥1 citation yang di-return overlap halaman yang diharapkan |
| `citation_precision` | 0-1 | Fraksi citation yang overlap |
| `verbatim_fidelity` | 0/1 | Semua string Arabic yang diharapkan muncul di `answer`/`extracted_quotes` |
| `answer_correct` | 0/0.5/1 | LLM-as-judge rating vs `expected_answer_gist` |

Plus `latency_p50/p95`, error rate.

## Setup sekali

```bash
cd rag-service
pip install -r requirements.txt     # sekarang termasuk pyyaml
cp .env.example .env                # lalu isi LLM_API_KEY, LLM_BASE_URL, DATABASE_URL
```

Pastikan `rag-service` jalan dan DB accessible:
```bash
uvicorn main:app --port 8001    # atau docker-compose up rag-service
```

## Isi golden set

Edit `eval/golden_set.yaml`. Target: 30 Q&A covering 2-3 kitab. Tiap Q butuh:

- `question` (wajib) — dalam bahasa apapun (Indo/Ar/En)
- `mode` — `per_kitab` atau `general`
- `kitab_slug` (untuk per_kitab) — harus match `kitab.slug` di DB
- `expected_pages` — range halaman di mana jawaban berada (tanpa ini `citation_hit` tidak diukur)
- `expected_arabic_contains` — lafadz Arabic verbatim yang HARUS muncul (tanpa ini `verbatim_fidelity` tidak diukur)
- `expected_answer_gist` — deskripsi ringkas jawaban benar untuk LLM judge (tanpa ini `answer_correct` tidak diukur)

Tiap field yang Anda isi menambah dimensi eval. Minimum useful: `expected_pages` + `expected_arabic_contains`.

## Run

```bash
cd rag-service
python eval/run_eval.py --label baseline-current
```

Opsi CLI:
```
--endpoint   default http://localhost:8001
--golden     default eval/golden_set.yaml
--label      prefix untuk file output (contoh: "voyage-context3")
--output     default eval/results
```

Output:
- Ringkasan di stdout
- JSON lengkap di `eval/results/<label>_<timestamp>.json` (per-question detail + summary)

## Alur pengambilan keputusan

1. **Baseline**: run dengan sistem sekarang (text-embedding-3-large + tree reasoning). Label = `baseline`.
2. **Experiment**: ganti satu variabel (misal: matikan tree reasoning, atau ganti embedding). Label = `<nama-eksperimen>`.
3. **Compare**: buka 2 JSON, bandingkan `summary` blok. Perubahan dianggap menang kalau `citation_hit_rate` ATAU `answer_correct_mean` naik tanpa yang lain turun.

## Catatan

- Judge model default: `anthropic/claude-haiku-4.5` (cheap + reliable). Override via `EVAL_JUDGE_MODEL`.
- Eval call bypass `INTERNAL_AUTH_TOKEN` kalau env var di-set (otomatis di-kirim di header).
- Kalau `citation_hit_rate` tinggi tapi `answer_correct_mean` rendah → retrieval OK, masalah di prompt/LLM.
- Kalau sebaliknya → retrieval miss, jawaban "pintar tapi ngarang".
