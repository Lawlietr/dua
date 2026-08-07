# Windows (win64) Porting Investigation

> 此分支用於記錄 Windows 移植的調查結果與待辦事項，尚未開始實作。

## 目標

僅移植 `dua analyze`，`dua status` 暫不處理。

## 編譯錯誤（`GOOS=windows GOARCH=amd64 go build`）

共 **6 個錯誤**，集中在 2 個檔案：

```
cmd/analyze/main.go:91  syscall.Statfs_t  undefined
cmd/analyze/main.go:92  syscall.Statfs    undefined
cmd/analyze/scanner.go:1087  syscall.Stat_t  undefined
cmd/analyze/scanner.go:1099  syscall.Stat_t  undefined
cmd/analyze/scanner.go:1112  syscall.Stat_t  undefined
cmd/analyze/scanner.go:1116  fileLastAccessTime  undefined
```

## 需要新增的 platform-specific 檔案

### 1. `cmd/analyze/atime_windows.go`（新增）

參考 `atime_linux.go` / `atime_darwin.go`，實作 `fileLastAccessTime`。

Windows 的 `syscall.Win32FileInfo` 有 `LastAccessTime` 欄位，直接可用。

```go
//go:build windows

package main

import (
    "syscall"
    "time"
)

func fileLastAccessTime(stat *syscall.Stat_t) time.Time {
    // Windows syscall.Stat_t 不存在，改用 Win32FileInfo
    // 此函數在 Windows 下不會被呼叫（見 scanner.go fallback）
    return time.Time{}
}
```

> **注意**：`scanner.go` 的三個 `syscall.Stat_t` 使用點都有 `if !ok { return info.Size() }` fallback，Windows 下 `Sys()` 回傳 `*syscall.Win32FileInfo` 不會匹配 `*syscall.Stat_t`，會直接走 fallback 回傳 `info.Size()`，**不需要改 scanner.go**。

### 2. `cmd/analyze/statfs_windows.go`（新增）

`main.go:91-92` 用 `syscall.Statfs` 取得磁碟可用空間。

Windows 替代方案：`kernel32.GetDiskFreeSpaceEx`。

```go
//go:build windows

package main

import (
    "syscall"
    "unsafe"
)

var (
    modkernel32        = syscall.NewLazyDLL("kernel32.dll")
    procGetDiskFreeSpaceExW = modkernel32.NewProc("GetDiskFreeSpaceExW")
)

func getDiskFreeBytes(path string) int64 {
    // 呼叫 GetDiskFreeSpaceExW，回傳可用位元組
    // 錯誤時回傳 0（caller 會處理 zero value）
    return 0
}
```

`main.go` 的 `newModel` 需改為呼叫 platform-specific 函數。

## 需要修改的現有檔案

### 3. `cmd/analyze/main.go` — `xdg-open` 替換

```go
// 現有（Linux/macOS）
exec.CommandContext(ctx, "xdg-open", path).Run()

// Windows 需改為
exec.CommandContext(ctx, "explorer.exe", path).Run()
```

需加 `runtime.GOOS == "windows"` 分支，或新增 `openFile_windows.go`。

### 4. `cmd/analyze/main.go` — overview roots

`systemOverviewRoots()` 目前只有 Linux 和 Darwin 分支，Windows 需新增：

```go
if runtime.GOOS == "windows" {
    return []dirEntry{
        {Name: "Desktop", Path: os.Getenv("USERPROFILE") + "\\Desktop", IsDir: true, Size: -1},
        {Name: "Documents", Path: os.Getenv("USERPROFILE") + "\\Documents", IsDir: true, Size: -1},
        {Name: "Downloads", Path: os.Getenv("USERPROFILE") + "\\Downloads", IsDir: true, Size: -1},
        {Name: "Pictures", Path: os.Getenv("USERPROFILE") + "\\Pictures", IsDir: true, Size: -1},
    }
}
```

### 5. `cmd/analyze/cleanable.go` — insight paths

`moCleanHandledPathFragments` 目前只有 Linux 和 Darwin，Windows 需新增：

```go
if runtime.GOOS == "windows" {
    return []string{
        "\\AppData\\Local\\Temp\\",
        "\\AppData\\Roaming\\npm\\cache\\",
        "\\AppData\\Local\\Microsoft\\Windows\\INetCache\\",
    }
}
```

### 6. `dua` router — 改為 PowerShell 或批處理

目前 `dua` 是 bash 腳本，Windows 需改寫。

選項：
- **選項 A**：改寫成 `.ps1`（PowerShell），但需用戶啟用執行策略
- **選項 B**：改寫成 `.bat`（批處理），功能較有限
- **選項 C**：合併 `cmd/analyze` 和 `cmd/status` 成單一 Go binary，移除 router（一勞永逸）

**建議先選 C**，但這是較大的架構變更，可留作後續。

短期方案：Windows 安裝時直接複製 `dua-analyze.exe` 和 `dua-status.exe` 到 PATH，不裝 router。

## `du` 依賴 — 已確認有 fallback

`scanner.go` 中 `getDirectorySizeFromDu` 失敗時會自動降級到 `calculateDirSizeFastWithLimiter`（純 Go 實作）：

```go
// scanner.go:319
size, err := getDirectorySizeFromDu(fullPath)
if err != nil || size <= 0 {
    size = calculateDirSizeFastWithLimiter(...)  // fallback
}
```

Windows 上 `du` 不存在會觸發 fallback，**功能正常，僅速度較慢**。

## 不需要處理的項目

| 項目 | 原因 |
|------|------|
| `mdfind`（Spotlight） | `scanner.go:431` 已有 `runtime.GOOS == "darwin"` 保護，Windows 直接跳過 |
| `dua status` | 不在本次範圍內 |
| `golang.org/x/sys/unix.Flock` | 在 `cmd/status/prefs.go`，不在 analyze 範圍內 |
| `/proc`、`/sys/class/hwmon` | 在 `cmd/status/`，不在 analyze 範圍內 |

## 預估工作量

| 項目 | 預估 |
|------|------|
| `atime_windows.go` | ~15 行 |
| `statfs_windows.go` | ~25 行 |
| `main.go` xdg-open 分支 | ~5 行 |
| `main.go` overview roots | ~8 行 |
| `cleanable.go` insight paths | ~8 行 |
| router 處理 | 暫不處理，或後續合併 binary |
| **總計** | **~60 行新增 + 修改，半天內可完成** |

## 測試策略

1. `GOOS=windows GOARCH=amd64 go build` 通過
2. `go test ./cmd/analyze` 在 Linux 上仍通過（Windows 測試需跳過）
3. 在 Windows 11 上手動測試 `dua-analyze.exe`：
   - 掃描 C:\ 或 D:\
   - 導覽目錄
   - Top-files 檢視
   - `--json` 輸出

## 後續步驟

- [ ] 實作 `atime_windows.go`
- [ ] 實作 `statfs_windows.go`
- [ ] 修改 `main.go` xdg-open
- [ ] 修改 `main.go` overview roots
- [ ] 修改 `cleanable.go` insight paths
- [ ] 處理 router（建議合併 binary）
- [ ] 更新 release workflow 加入 windows/amd64
- [ ] 更新 README 加入 Windows 安裝說明
