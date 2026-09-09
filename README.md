# ndWhatsApp 🔬
### Advanced Native Client & Reverse Engineering Suite for WhatsApp Web

[![Platform](https://img.shields.io/badge/platform-Linux%20(Fedora%20%7C%20Ubuntu%20%7C%20Arch)%20%7C%20macOS%20%7C%20Windows-blue)](#)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](https://golang.org)
[![Engine](https://img.shields.io/badge/Engine-WebKitGTK%204.1%20%2F%20WebKit-green)](#)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**ndWhatsApp** adalah desktop client WhatsApp berbasis **Go + WebKitGTK 4.1** yang sangat ringan (~6.7 MB binary, ~80 MB RAM), dilengkapi dengan **perbaikan native Linux Wayland**, **resizable chat sidebar 60 FPS**, dan **perangkat rekayasa balik (*Reverse Engineering & Developer Suite*)** terintegrasi.

---

## 🌟 Fitur Utama

### 1. 🔬 Reverse Engineering & Developer Suite
- **Remote WebKit Inspector Server**: Berjalan otomatis di `http://127.0.0.1:9222`. Anda dapat membuka browser Chromium/Chrome atau Edge, mengetikkan URL tersebut, dan melakukan visual debugging penuh terhadap DOM, Network WebSocket, Console, dan Memory Heap WhatsApp Web.
- **`window.ndWA` & `window.WA` Store Hook**: Otomatis menyusup ke dalam chunk loader Webpack WhatsApp Web (`webpackChunkwhatsapp_web_client`) dan mengekstrak modul-modul internal penting:
  - `WA.Store.Msg` — Koleksi dan model pesan.
  - `WA.Store.Chat` — Model obrolan aktif, unread count, dan pin status.
  - `WA.Store.Contact` — Kontak WhatsApp, nomor telepon, dan status profil.
  - `WA.Store.Conn` — Status konektivitas dan informasi akun login.
  - `WA.Store.Socket` — Socket channel dan state transport Noise Protocol.
- **In-App DevTools HUD (`Ctrl + Shift + D` / `F12`)**: Panel instrumen mengambang (*floating HUD*) di dalam aplikasi untuk memantau:
  - **Metrics**: PID proses, alokasi memori Go runtime, jumlah goroutine.
  - **Store**: Daftar obrolan aktif beserta JID dan unread count.
  - **Live Events**: Stream pesan masuk secara real-time (`Store.Msg.on('add')`).
  - **Quick JS Runner**: Menjalankan ekspresi JavaScript langsung di konteks WhatsApp Web dengan output terformat.
- **ContextMenu Unblocker**: Tekan **`Shift + Klik Kanan`** pada elemen mana pun untuk memunculkan context menu asli WebKit (Inspect Element, Reload, Back, Forward).

### 2. 📐 Resizable Chat List Sidebar (60 FPS Smooth Drag)
- **Tarik Garis Pembatas Langsung**: Arahkan kursor ke garis vertikal pembatas list chat, klik dan geser untuk memperlebar atau memperkecil area obrolan.
- **Zero-Lag 60 FPS Sync**: Ditenagai oleh `requestAnimationFrame` dan bypassing pointer-events, memindahkan batas list chat dan garis vertikal background secara sinkron (*lockstep*) tanpa patah-patah (*zero layout thrashing*).
- **Shortcut & Slider**: Gunakan `Ctrl + [` (perkecil), `Ctrl + ]` (perlebar), `Ctrl + \` (reset ke 380px), atau slider interaktif pada dialog Settings.

### 3. 🐧 Fedora & Modern Wayland Optimizations
- Menggunakan **WebKitGTK 4.1** (libxml2, soup3) yang kompatibel penuh dengan Fedora 41/42/43/44, Arch Linux, dan Ubuntu modern.
- Menangani crash WebKit di Wayland/Mesa EGL secara otomatis menggunakan flag komposit aman (`WEBKIT_DISABLE_DMABUF_RENDERER=1`, `WEBKIT_DISABLE_COMPOSITING_MODE=1`).

### 4. 📄 In-App Document Preview & Media
- Pratinjau langsung untuk file PDF, Word (`.docx`), Excel (`.xlsx`), dan teks tanpa harus mengunduh file terlebih dahulu.
- Mendukung pemutaran audio/video inline, voice notes, serta panggilan suara dan video.

---

## ⌨️ Pintasan Keyboard (Shortcuts Reference)

| Shortcut | Aksi |
| :--- | :--- |
| **`F12`** / **`Ctrl + Shift + D`** | **Buka / Tutup In-App DevTools & RE HUD** |
| **`Shift + Klik Kanan`** | **Buka Context Menu Asli Browser (Inspect Element)** |
| **`Ctrl + [`** | Perkecil lebar list chat (-30px) |
| **`Ctrl + ]`** | Perlebar lebar list chat (+30px) |
| **`Ctrl + \`** | Reset lebar list chat ke ukuran default (380px) |
| **`Ctrl + ,`** | Buka dialog Pengaturan (Settings) |
| **`Ctrl + Shift + P`** | Mode Privasi (Anti-Peeking blur) |
| **`Ctrl + Shift + T`** | Always on Top (Window Pinning) |
| **`Ctrl + Shift + M`** | Mute / Unmute notifikasi suara |
| **`Ctrl + Shift + O`** | Buka folder unduhan (*Downloads*) |
| **`Ctrl + Shift + U`** | Periksa pembaruan (*Check for updates*) |
| **`F5`** / **`Ctrl + R`** | Muat ulang obrolan (*Reload*) |
| **`Ctrl + Shift + R`** | Hard refresh antarmuka |

---

## 🛠️ Panduan RE Console API (`window.ndWA`)

Buka Remote Inspector di `http://127.0.0.1:9222` atau tekan `F12` / `Ctrl+Shift+D` untuk membuka tab *Run JS*. Anda dapat mengeksekusi perintah berikut:

```javascript
// 1. Dapatkan semua model obrolan
var chats = ndWA.getChats();
console.log('Total chat:', chats.length);

// 2. Tampilkan 5 chat teratas
chats.slice(0, 5).map(c => ({
    name: c.name || c.formattedTitle,
    jid: c.id._serialized,
    unread: c.unreadCount
}));

// 3. Cari modul internal Webpack berdasarkan kata kunci
ndWA.inspectModule('crypto');
ndWA.inspectModule('socket');

// 4. Dengarkan pesan masuk secara terprogram
ndWA.onMessage(function(msg) {
    console.log('📥 Pesan baru dari:', msg.from, 'Isi:', msg.body);
});

// 5. Cek status koneksi dan sesi
console.log(ndWA.getConn());
```

---

## 🚀 Kompilasi & Menjalankan

### Persyaratan Sistem (Linux / Fedora):
Pastikan pustaka WebKitGTK 4.1 dan GTK3 terpasang di sistem Anda:

```bash
# Fedora
sudo dnf install webkit2gtk4.1-devel gtk3-devel

# Ubuntu / Debian
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev

# Arch Linux
sudo pacman -S webkit2gtk-4.1 gtk3
```

### Build Binary:
```bash
cd ~/ndwhatsapp
go build -ldflags="-s -w -buildid=" -trimpath -o ndwhatsapp .
```

### Jalankan:
```bash
./ndwhatsapp
```

Untuk memasang ke user PATH:
```bash
cp -f ndwhatsapp ~/.local/bin/ndwhatsapp
```

---

## 🔒 Privasi & Keamanan
- **100% Enkripsi Asli**: ndWhatsApp tidak mengubah protokol transmisi WhatsApp; semua obrolan tetap dienkripsi penuh menggunakan Signal Protocol bawaan WhatsApp Web.
- **Lokal & Mandiri**: Seluruh instrumen reverse engineering dan sniffer berjalan murni di memori lokal komputer Anda tanpa transmisi ke server pihak ketiga.

---

## 📜 Lisensi
Proyek ini dilisensikan di bawah [MIT License](LICENSE).
