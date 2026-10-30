import { useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import type { Pocket } from '../app/usePocket';
import { api } from '../app/api';
import type { ScanProgress } from '../app/types';
import { PageHeader, Segmented, Select, Toggle } from '../components/ui';

const LAYOUTS = [
  { value: 'music', labelKey: 'music' },
  { value: 'library', labelKey: 'libraryLayout' },
  { value: 'minimal', labelKey: 'minimal' },
];

const THEMES = [
  { value: 'system', labelKey: 'system' },
  { value: 'light', labelKey: 'light' },
  { value: 'dark', labelKey: 'dark' },
];

const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'zh', label: '简体中文' },
  { value: 'ja', label: '日本語' },
];

const CACHE_SIZES = ['128', '256', '512', '1024', '2048'].map((value) => ({ value, label: `${value} MB` }));
const AUDIO_OUTPUTS = ['null', 'pulse', 'alsa', 'pipewire', 'jack'].map((value) => ({ value, label: value }));

export function SettingsPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions } = pocket;
  const [folder, setFolder] = useState(data.settings.music_folder || '');
  const [cacheSize, setCacheSize] = useState(data.settings.cache_size || '512');
  const [audioOutput, setAudioOutput] = useState(data.settings.audio_output || 'null');
  const [autoScan, setAutoScan] = useState(data.settings.auto_scan === 'true');
  const [scan, setScan] = useState<ScanProgress>(data.scan);

  useEffect(() => {
    setFolder(data.settings.music_folder || '');
    setCacheSize(data.settings.cache_size || '512');
    setAudioOutput(data.settings.audio_output || 'null');
    setAutoScan(data.settings.auto_scan === 'true');
  }, [data.settings]);

  useEffect(() => {
    void api.scanStatus().then(setScan).catch(() => {});
  }, []);

  useEffect(() => {
    if (!scan.scanning) return;
    const timer = window.setInterval(() => {
      void api.scanStatus().then(setScan).catch(() => {});
    }, 600);
    return () => window.clearInterval(timer);
  }, [scan.scanning]);

  const percent = scan.total > 0 ? Math.min(100, Math.round((scan.found / scan.total) * 100)) : 0;

  return (
    <>
      <PageHeader eyebrow={tr('preferences')} title={tr('settings')} />

      <div className="settings-panel">
        <div className="settings-group">
          <h2>{tr('appearance')}</h2>
          <div className="setting-row">
            <div>
              <b>{tr('layout')}</b>
              <span>
                {tr('music')} · {tr('libraryLayout')} · {tr('minimal')}
              </span>
            </div>
            <Segmented
              label={tr('layout')}
              value={pocket.layout}
              onChange={(value) => actions.setLayout(value as typeof pocket.layout)}
              options={LAYOUTS.map((item) => ({ value: item.value, label: tr(item.labelKey) }))}
            />
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('theme')}</b>
              <span>{data.settings.theme}</span>
            </div>
            <Segmented
              label={tr('theme')}
              value={pocket.theme}
              onChange={(value) => actions.setTheme(value as typeof pocket.theme)}
              options={THEMES.map((item) => ({ value: item.value, label: tr(item.labelKey) }))}
            />
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('language')}</b>
              <span>{pocket.lang}</span>
            </div>
            <Segmented label={tr('language')} value={pocket.lang} onChange={(value) => actions.setLang(value as typeof pocket.lang)} options={LANGUAGES} />
          </div>
        </div>

        <div className="settings-group">
          <h2>{tr('libraryStats')}</h2>
          <div className="setting-row">
            <div>
              <b>{tr('musicFolder')}</b>
              <span>{data.health?.root || folder}</span>
            </div>
            <input className="small-input" value={folder} onChange={(event) => setFolder(event.target.value)} aria-label={tr('musicFolder')} />
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('scan')}</b>
              <span>
                {scan.scanning
                  ? `${tr('scanning')} ${percent}% (${scan.found}/${scan.total})`
                  : scan.done
                    ? `${tr('scanDone')} · ${scan.found} ${tr('trackCount')}`
                    : tr('scanIdle')}
                {scan.error ? ` · ${scan.error}` : ''}
              </span>
              {scan.scanning && (
                <div className="progress-track" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent} aria-label={tr('scanProgress')}>
                  <div className="progress-fill" style={{ width: `${percent}%` }} />
                </div>
              )}
            </div>
            <div className="row-actions">
              <button type="button" className="secondary-button" onClick={() => { actions.startScan(); setScan((current) => ({ ...current, scanning: true })); }} disabled={scan.scanning}>
                <RefreshCw size={15} /> {tr('scanNow')}
              </button>
              {scan.scanning && (
                <button type="button" className="secondary-button" onClick={actions.cancelScan}>
                  {tr('scanCancel')}
                </button>
              )}
            </div>
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('autoScan')}</b>
              <span>{autoScan ? 'on' : 'off'}</span>
            </div>
            <Toggle checked={autoScan} onChange={setAutoScan} label={tr('autoScan')} />
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('cacheSize')}</b>
              <span>{cacheSize} MB</span>
            </div>
            <Select label={tr('cacheSize')} value={cacheSize} onChange={setCacheSize} options={CACHE_SIZES} />
          </div>
        </div>

        <div className="settings-group">
          <h2>{tr('audioOutput')}</h2>
          <div className="setting-row">
            <div>
              <b>{tr('audioOutput')}</b>
              <span>
                {data.health?.audio.engine ?? 'mpv'} · {data.health?.audio.ao ?? audioOutput}
              </span>
            </div>
            <Select label={tr('audioOutput')} value={audioOutput} onChange={setAudioOutput} options={AUDIO_OUTPUTS} />
          </div>
        </div>

        <div className="settings-group">
          <h2>{tr('serviceStatus')}</h2>
          <div className="setting-row">
            <div>
              <b>{tr('status')}</b>
              <span>
                {data.health ? `${data.health.provider} · ${data.health.mock ? tr('mockLibrary') : tr('realLibrary')}` : tr('offline')}
              </span>
            </div>
            <span className={`status-dot ${data.health ? 'online' : ''}`} aria-hidden="true" />
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('libraryStats')}</b>
              <span>
                {data.health ? `${data.health.library.tracks} ${tr('trackCount')} · ${data.health.library.albums} ${tr('albumCount')} · ${data.health.library.artists} ${tr('artists')}` : '—'}
              </span>
            </div>
          </div>
          <div className="setting-row">
            <div>
              <b>{tr('keyboardShortcuts')}</b>
              <div className="keyboard-help">
                <span>{tr('shortcutPlay')}</span>
                <span>{tr('shortcutNav')}</span>
                <span>{tr('shortcutSearch')}</span>
                <span>{tr('shortcutHome')}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="setting-actions">
          <button
            type="button"
            className="primary-button"
            disabled={data.scan.scanning}
            onClick={() => {
              actions.saveSettings({ music_folder: folder, cache_size: cacheSize, audio_output: audioOutput, auto_scan: autoScan ? 'true' : 'false' });
            }}
          >
            {tr('saveSettings')}
          </button>
        </div>
      </div>
    </>
  );
}
