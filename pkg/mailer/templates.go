package mailer

import "fmt"

func VerificationEmail(displayName, code string, expiresInMinutes int) Message {
	subject := "Verifikasi email Anda"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto;padding:24px;color:#111">
<h2 style="margin:0 0 12px">Halo %s,</h2>
<p>Gunakan kode berikut untuk memverifikasi email Anda:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:6px;background:#f4f4f5;padding:16px;text-align:center;border-radius:8px">%s</p>
<p>Kode berlaku selama <strong>%d menit</strong>. Jika Anda tidak merasa mendaftar, abaikan email ini.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0">
<p style="font-size:12px;color:#6b7280">Email ini dikirim otomatis, mohon jangan dibalas.</p>
</body></html>`, escape(displayName), code, expiresInMinutes)

	text := fmt.Sprintf("Halo %s,\n\nKode verifikasi email Anda: %s\nBerlaku %d menit.\n\nJika Anda tidak merasa mendaftar, abaikan email ini.",
		displayName, code, expiresInMinutes)

	return Message{Subject: subject, HTMLBody: html, TextBody: text}
}

func PasswordResetEmail(displayName, resetURL string, expiresInMinutes int) Message {
	subject := "Permintaan reset password"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto;padding:24px;color:#111">
<h2 style="margin:0 0 12px">Halo %s,</h2>
<p>Kami menerima permintaan untuk mereset password Anda. Klik tombol di bawah untuk melanjutkan:</p>
<p style="text-align:center;margin:24px 0">
<a href="%s" style="background:#111827;color:#fff;text-decoration:none;padding:12px 24px;border-radius:6px;display:inline-block">Reset Password</a>
</p>
<p>Atau salin tautan berikut ke browser Anda:</p>
<p style="word-break:break-all;font-size:12px;color:#6b7280">%s</p>
<p>Tautan berlaku <strong>%d menit</strong>. Jika Anda tidak meminta reset password, abaikan email ini — password Anda tidak akan berubah.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0">
<p style="font-size:12px;color:#6b7280">Email ini dikirim otomatis, mohon jangan dibalas.</p>
</body></html>`, escape(displayName), resetURL, resetURL, expiresInMinutes)

	text := fmt.Sprintf("Halo %s,\n\nGunakan tautan berikut untuk mereset password Anda (berlaku %d menit):\n%s\n\nJika Anda tidak meminta reset password, abaikan email ini.",
		displayName, expiresInMinutes, resetURL)

	return Message{Subject: subject, HTMLBody: html, TextBody: text}
}

func NewDeviceLoginEmail(displayName, userAgent, ipAddress, when, revokeURL string) Message {
	subject := "Login baru terdeteksi di akun Anda"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto;padding:24px;color:#111">
<h2 style="margin:0 0 12px">Halo %s,</h2>
<p>Kami mendeteksi login baru di akun Anda:</p>
<ul style="background:#f4f4f5;padding:16px 24px;border-radius:8px;list-style:none">
<li><strong>Perangkat:</strong> %s</li>
<li><strong>IP:</strong> %s</li>
<li><strong>Waktu:</strong> %s</li>
</ul>
<p>Jika ini Anda, aman untuk diabaikan. Jika <strong>bukan Anda</strong>, segera ubah password:</p>
<p style="text-align:center;margin:24px 0">
<a href="%s" style="background:#dc2626;color:#fff;text-decoration:none;padding:12px 24px;border-radius:6px;display:inline-block">Amankan Akun</a>
</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0">
<p style="font-size:12px;color:#6b7280">Email ini dikirim otomatis, mohon jangan dibalas.</p>
</body></html>`, escape(displayName), escape(userAgent), escape(ipAddress), escape(when), revokeURL)

	text := fmt.Sprintf("Halo %s,\n\nLogin baru terdeteksi:\nPerangkat: %s\nIP: %s\nWaktu: %s\n\nJika bukan Anda, segera amankan akun: %s",
		displayName, userAgent, ipAddress, when, revokeURL)

	return Message{Subject: subject, HTMLBody: html, TextBody: text}
}

func PasswordChangedEmail(displayName, when string) Message {
	subject := "Password akun Anda baru saja diubah"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto;padding:24px;color:#111">
<h2 style="margin:0 0 12px">Halo %s,</h2>
<p>Password akun Anda diubah pada <strong>%s</strong>. Semua sesi aktif telah di-logout.</p>
<p>Jika ini <strong>bukan Anda</strong>, segera hubungi dukungan — akun Anda mungkin telah diretas.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0">
<p style="font-size:12px;color:#6b7280">Email ini dikirim otomatis, mohon jangan dibalas.</p>
</body></html>`, escape(displayName), escape(when))

	text := fmt.Sprintf("Halo %s,\n\nPassword akun Anda diubah pada %s. Semua sesi aktif telah di-logout.\nJika bukan Anda, segera hubungi dukungan.",
		displayName, when)

	return Message{Subject: subject, HTMLBody: html, TextBody: text}
}

func AccountDeletionRequestedEmail(displayName, when, gracePeriod, cancelURL string) Message {
	subject := "Permintaan penghapusan akun diterima"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto;padding:24px;color:#111">
<h2 style="margin:0 0 12px">Halo %s,</h2>
<p>Kami menerima permintaan penghapusan akun Anda pada <strong>%s</strong>.</p>
<p>Akun dan seluruh data Anda akan <strong>dihapus permanen</strong> setelah <strong>%s</strong>. Selama masa tunggu ini, Anda masih bisa membatalkan permintaan dengan login dan mengklik tombol di bawah:</p>
<p style="text-align:center;margin:24px 0">
<a href="%s" style="background:#111827;color:#fff;text-decoration:none;padding:12px 24px;border-radius:6px;display:inline-block">Batalkan Penghapusan</a>
</p>
<p>Jika Anda <strong>tidak meminta</strong> penghapusan akun, segera login dan batalkan — akun Anda mungkin telah disusupi.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0">
<p style="font-size:12px;color:#6b7280">Email ini dikirim otomatis, mohon jangan dibalas.</p>
</body></html>`, escape(displayName), escape(when), escape(gracePeriod), cancelURL)

	text := fmt.Sprintf("Halo %s,\n\nPermintaan penghapusan akun diterima pada %s. Akun akan dihapus permanen setelah %s.\nBatalkan di: %s\n\nJika bukan Anda, segera login dan batalkan.",
		displayName, when, gracePeriod, cancelURL)

	return Message{Subject: subject, HTMLBody: html, TextBody: text}
}

func AccountDeletionCancelledEmail(displayName, when string) Message {
	subject := "Penghapusan akun dibatalkan"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto;padding:24px;color:#111">
<h2 style="margin:0 0 12px">Halo %s,</h2>
<p>Permintaan penghapusan akun Anda telah <strong>dibatalkan</strong> pada %s. Akun Anda kembali aktif normal.</p>
<p>Jika ini <strong>bukan Anda</strong>, segera ubah password dan hubungi dukungan.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:24px 0">
<p style="font-size:12px;color:#6b7280">Email ini dikirim otomatis, mohon jangan dibalas.</p>
</body></html>`, escape(displayName), escape(when))

	text := fmt.Sprintf("Halo %s,\n\nPermintaan penghapusan akun Anda dibatalkan pada %s. Akun kembali aktif.\nJika bukan Anda, segera ubah password.", displayName, when)

	return Message{Subject: subject, HTMLBody: html, TextBody: text}
}

func escape(s string) string {
	var out []rune
	for _, r := range s {
		switch r {
		case '<':
			out = append(out, []rune("&lt;")...)
		case '>':
			out = append(out, []rune("&gt;")...)
		case '&':
			out = append(out, []rune("&amp;")...)
		case '"':
			out = append(out, []rune("&quot;")...)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
