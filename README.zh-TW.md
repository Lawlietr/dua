<div align="center">
  <h1>dua</h1>
  <p><em>Linux 磁碟分析與系統狀態，從終端機開始。</em></p>
  <p><a href="README.md">English</a> | 繁體中文</p>
</div>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL_v3-blue.svg?style=flat-square" alt="License"></a>
</p>

dua 是一套終端機優先的 Linux 檢視工具，提供兩個唯讀命令，源自 macOS Mole CLI 的 `analyze` 與 `status`，並重新定位為獨立專案。它能從 CLI、腳本或精簡 TUI 查看磁碟空間用到哪去了、系統健康度如何。dua **絕不會刪除或修改使用者資料**：整個程式碼庫不存在任何清理（cleanup）功能。

## 安裝

### 二進制安裝（curl）

由 GitHub Actions 為 Linux 建置（amd64 與 arm64）。不需 sudo，安裝到 `~/.local/bin`：

```bash
curl -fsSL "https://raw.githubusercontent.com/Lawlietr/dua/main/scripts/install.sh" | bash
```

將 `~/.local/bin` 加入 `PATH`（通常加進 `~/.bashrc` 或 `~/.zshrc`）即可永久使用。

### 從原始碼建置

需要 Go 1.25+：

```bash
make build
./dua status --json        # 機器可讀的狀態
./dua analyze --json /some/path
./dua analyze              # 概覽 TUI（不帶 PATH 即掃描整機）
./dua status               # TUI 儀表板
```

把二進制放到 PATH 上的目錄：

```bash
cp bin/dua-analyze bin/dua-status /usr/local/bin/
```

## 功能

- **`dua analyze`** — 視覺化磁碟瀏覽器：
  - 系統概覽（`/usr`、`/opt`、`/var`、`/home`）加上隱藏空間洞察（npm、Go build、pip、Gradle、JetBrains 快取、資源回收筒、舊的下載）
  - 逐層下鑽導覽，含目錄比例條與最近存取提示
  - Top-files（T）檢視、即時掃描回饋（可切換排序模式）、磁碟用量快取
  - `--json` 輸出供自動化使用
- **`dua status`** — 精簡健康儀表板：
  - CPU、記憶體、磁碟、溫度（hwmon）、硬體型號、OS 資訊與 top 程序
  - 高 CPU 程序警示
  - `--json` 單次輸出、`--watch` NDJSON 串流

## 命令

### `dua analyze [PATH]`

- 不帶 PATH：系統概覽（系統根目錄 + 洞察目錄），Enter 下鑽。
- 帶 `PATH`：掃描該目錄。
- 按鍵：`↑↓←→` 導覽、`Enter` 下鑽、`Esc` 返回、`R` 重新整理、`/` 過濾、`T` Top 檔案、`O` 用 `xdg-open` 開啟、`P` 預覽、`F` 在檔案管理員中顯示、`Q`/`Ctrl+C` 離開。
- 旗標：`--json` 以 JSON 輸出掃描結果。

環境變數：`DUA_ANALYZE_PATH` 在未指定 PATH 時設定掃描目標；`DUA_ANALYZE_LIVE_SORT` 選擇即時掃描排序模式。

### `dua version`

顯示版本與提交資訊。

## 運作原理

- `dua` 是薄型 bash router，派發到兩個 Go 二進制（`dua-analyze`、`dua-status`）。
- 大小來自受限制的並行走訪；`du -skPx`（含平台感知的排除旗標）量測摺疊目錄，快取避免重複掃描未變更的樹。
- 狀態指標來自 `/proc`、`/sys/class/hwmon`、DMI 與 `/etc/os-release`；不需要特權存取。
- 分析在結構上就是唯讀：不存在任何刪除、截斷或修改的程式碼路徑。

## 開發

```bash
make build     # 建置 bin/dua-analyze 與 bin/dua-status
make check     # go vet ./... + go test ./...
go test ./...  # 完整測試套件
```

專案契約、熱點檔案與測試注意事項見 `AGENTS.md`。

## License

GPL v3。見 [LICENSE](LICENSE)。
