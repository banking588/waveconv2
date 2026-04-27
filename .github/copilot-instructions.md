# Copilot Instructions for waveconv2

> 本文件提供 GitHub Copilot 在此 repository 中產生程式碼、回答問題與審查 PR 時所需的專案背景與規範。請依實際需求調整內容。

## 專案概觀

- **名稱**：`waveconv2`
- **Go module**：`github.com/banking588/waveconv2`
- **語言 / 版本**：Go 1.21
- **用途**：DDR5 / LPDDR5 等記憶體測試 pattern / waveform 轉換工具（wave converter v2）。

## 目錄結構

```
.
├── cmd/
│   └── app/                  # 主執行檔進入點
│       ├── main.go
│       ├── makefile
│       ├── config.ini
│       └── clis/             # CLI 子命令
│           ├── root.go
│           ├── patconv/      # pattern 轉換子命令
│           ├── patreader/    # pattern 讀取子命令
│           └── subcmd/
├── pkg/
│   ├── ddr5/                 # DDR5 相關邏輯（matcher / step）
│   ├── lpddr5/               # LPDDR5 commands / mr / pinwatcher
│   ├── middleend/            # 中介層
│   │   ├── pin/              # pin / clock / state / wave / map
│   │   └── patt/             # pattern / testcycle / cyclebuf / pinnameset
│   ├── pattern/              # pattern 模型、reader、executor (cp/ft12/ft34/rdbi/baseexecutor)
│   ├── patconverter/         # pattern converter 主邏輯（含 parser / convlib / timingcheck）
│   ├── filehandler/
│   ├── iface/
│   ├── model/
│   ├── testcase/             # 測試案例 (T5503HS、ddjob、vec)
│   └── util/
├── scripts/
│   └── build.mk              # 共用 build 設定（多平台、版本資訊注入）
├── makefile                  # 頂層 makefile
└── go.mod
```

## 建置 / 測試 / 開發指令

頂層 `makefile` 會走訪 `cmd/**/main.go` 並對每個子目錄執行 build。

| 目的 | 指令 |
| --- | --- |
| 建置全部子命令 | `make` 或 `make all` |
| 整理依賴 | `make tidy`（等同 `go mod tidy`） |
| 執行單元測試 | `make test`（會跑 `go test -v -count=1 ./...` 再執行各 cmd 的 test） |
| 清理產出 | `make clean` / `make cleanall` |
| 初始化 Go 環境 | `make initenv`（設定 `GOPROXY` / `GOPRIVATE` 等） |

建置時 `scripts/build.mk` 會透過 `-ldflags` 注入：

- `main.Version`（取自 `ver.txt` 或 `git describe`）
- `main.BuildTime`
- `main.ProgName`
- `main.GitHash`

預設目標平台：`windows-64`、`linux-64`，另支援 `linux-32`、`linux-armv7`、`linux-armv8`。

## 編碼規範

- 遵守 **官方 Go 風格**：使用 `gofmt` / `goimports` 格式化，命名採 CamelCase，匯出符號需附 GoDoc 註解。
- **Package 命名**：小寫、單字、與目錄同名（已存在的子目錄請維持原本 package 名，例如 `patt`、`patconv`）。
- **錯誤處理**：使用標準 `error` 介面，必要時以 `fmt.Errorf("...: %w", err)` 包裝以保留錯誤鏈，避免直接 `panic`（除非屬於不可恢復的初始化錯誤）。
- **檔案佈局**：CLI 子命令放在 `cmd/app/clis/<name>/subcmd.go`；對應的領域邏輯放在 `pkg/<domain>/`，避免在 `cmd/` 直接撰寫業務邏輯。
- **依賴**：新增第三方套件前先確認是否能用標準函式庫達成；新增後務必執行 `make tidy`。
- **註解語言**：可使用中文或英文，但同一檔案中盡量一致；公開 API 建議使用英文 GoDoc。

## 測試規範

- 單元測試與被測程式同 package，檔名以 `_test.go` 結尾（例：`pkg/patconverter/converter_test.go`、`pkg/pattern/pat_*_test.go`）。
- 新增功能務必補上對應的測試；修 bug 時先寫能重現的測試再修正。
- 跑測試請使用 `make test` 而非僅 `go test ./...`，以確保 `cmdtest` 也通過。

## 領域知識提示

- **Pattern / Wave**：本專案處理的是半導體記憶體測試機台（如 T5503HS）所用的 pattern / waveform 檔案；轉換流程大致為 `parser → middleend (pin/patt) → executor → output`。
- **Executor 種類**：`cp`、`ft12`、`ft34`、`rdbi`、`baseexecutor`，各自定義 `opcode` / `instruction` / `args`。
- **Memory 規格**：`pkg/ddr5`、`pkg/lpddr5` 為對應 DRAM 規格的命令與 mode register 處理。

## Pull Request / Commit 慣例

- Commit message 採祈使句，盡量簡短說明「做了什麼」（例：`Add DDR5 small test sequence`）。
- 一個 PR 聚焦單一目的；混合 refactor 與 feature 請拆分。
- PR 描述需說明：變更動機、影響範圍、測試方式。

## Copilot 在本 repo 的行為準則

1. **小步修改**：除非明確要求，否則不要重寫大段現有程式碼；以最小變更達成需求。
2. **保留既有結構**：尊重 `cmd/` ↔ `pkg/` 分層與既有 package 名稱。
3. **不要引入未使用的依賴**；新增依賴前先在 PR 描述中說明理由。
4. **產生的程式必須 `gofmt` 過**，且通過 `go vet ./...` 與 `make test`。
5. **若資訊不足**（例如硬體規格、檔案格式細節），請在回覆中明確指出需要使用者補充，而不是臆測。
6. **語言**：與使用者互動時可使用繁體中文；程式碼識別字、錯誤訊息使用英文。

---

> 📝 **TODO（待使用者調整）**：請依實際需求補充／修改下列項目
> - [ ] 專案用途與業務背景的更精確描述
> - [ ] 各 `pkg/*` 子套件的責任邊界
> - [ ] 支援的輸入 / 輸出檔案格式說明
> - [ ] CI / Lint / 額外工具鏈設定
> - [ ] 安全性與相依性政策
