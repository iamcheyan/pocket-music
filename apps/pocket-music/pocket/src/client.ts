// Music client: library/playback over musicd. No audio path exists on the
// device profile yet, so playback actions are visible state changes only.
import { createPortClient } from "./client-kit";
import type { AppConfig, Metric, Section, SettingRow } from "./model";

const sections: Section[] = [
  {
    id: "recent", title: "RECENTLY ADDED", jp: "最近追加", items: [
      { id: "t1", title: "Night Drive", subtitle: "Aster · Neon Transit", meta: "4:12 · FLAC", badge: "FLAC", progress: 0.31, body: "Synth pads over a slow 4/4 pulse; the queue keeps the next two tracks warm.", lines: [{ label: "album", value: "Neon Transit" }, { label: "year", value: "2024" }, { label: "bit", value: "24/48" }] },
      { id: "t2", title: "Glass Harbor", subtitle: "Mira · Low Orbit", meta: "3:48 · MP3", badge: "MP3", progress: 0, body: "A quiet guitar loop with field recordings from the harbour.", lines: [{ label: "album", value: "Low Orbit" }, { label: "year", value: "2023" }, { label: "bit", value: "320k" }] },
      { id: "t3", title: "Signal Bloom", subtitle: "Vela · Deep Field", meta: "5:02 · FLAC", badge: "FLAC", progress: 0, body: "Long-form ambient piece from the Deep Field sessions." },
    ],
  },
  {
    id: "playlists", title: "PLAYLISTS", jp: "プレイリスト", items: [
      { id: "p1", title: "Late Shift", subtitle: "18 tracks · 72 min", meta: "updated today", badge: "PL", body: "Slow tempo, low vocals, no sudden transients." },
      { id: "p2", title: "Survey Mix", subtitle: "24 tracks · 96 min", meta: "updated 2 d", badge: "PL", body: "The working queue for long galaxy runs." },
      { id: "p3", title: "Favourites", subtitle: "41 tracks · 2.8 h", meta: "synced", badge: "FAV", body: "Every hearted track in the library." },
    ],
  },
];

const metrics: Metric[] = [
  { label: "TRACKS", value: "1,284" },
  { label: "ALBUMS", value: "96" },
  { label: "QUEUE", value: "7" },
];

const settings: SettingRow[] = [
  { id: "output", label: "Output", jp: "出力", value: "SPEAKER", options: ["SPEAKER", "HEADPHONES", "HDMI"] },
  { id: "quality", label: "Quality", jp: "音質", value: "LOSSLESS", options: ["LOSSLESS", "HIGH", "DATA"] },
  { id: "scan", label: "Scan on start", jp: "起動時スキャン", value: "ON", options: ["ON", "OFF"] },
];

export const APP: AppConfig = {
  title: "MUSIC",
  jp: "音楽",
  kind: "player",
  page: "MUSIC",
  tabs: ["音楽", "再生", "設定"],
  actions: ["PREV", "PLAY", "NEXT", "FAV"],
  settingsTitle: "PLAYBACK",
  settingsJp: "再生設定",
  deviceGap: "musicd + audio sink missing.",
  sourceNote: "/api/library · /api/playback",
};

export const client = createPortClient({
  resource: "music",
  fixture: { sections, metrics, settings },
  endpoints: {
    list: "/api/library/songs",
    get: (id) => `/api/library/songs/${encodeURIComponent(id)}`,
    action: (name) => `/api/playback/${encodeURIComponent(name.toLowerCase())}`,
  },
  act(name, payload, bundle) {
    const items = bundle.sections.flatMap((section) => section.items);
    const item = items.find((entry) => entry.id === payload?.id) ?? items[0];
    if (!item) return { ok: false, message: "empty queue" };
    if (name === "PLAY") {
      item.state = item.state === "playing" ? "paused" : "playing";
      item.progress = Math.min(1, (item.progress ?? 0) + 0.08);
      return { ok: true, message: `${item.title} ${item.state} · mock sink` };
    }
    if (name === "NEXT" || name === "PREV") {
      const at = items.findIndex((entry) => entry.id === item.id);
      const next = items[(at + (name === "NEXT" ? 1 : items.length - 1)) % items.length]!;
      next.state = "playing";
      return { ok: true, message: `${name}: ${next.title}` };
    }
    if (name === "FAV") { item.badge = "FAV"; return { ok: true, message: `${item.title} favourited` }; }
    return { ok: false, message: `${name} unavailable` };
  },
});
