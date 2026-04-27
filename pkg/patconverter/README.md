# patconverter

`patconverter` 是 `waveconv2` 中負責將 DRAM 測試序列 (sequence / pattern) 進行解析、時序檢查 (timing check) 與轉換輸出 (pattern conversion) 的 package。

目前主要實作集中在 `timingcheck/ddr5checker`，其餘子套件（`convlib`、`parser` 等）為對外介面與後續擴充用的骨架。

---

## 目錄結構

```
pkg/patconverter/
├── converter.go              # patconverter 對外入口（骨架）
├── converter_test.go
├── convlib/                  # 各種 DRAM 規格的轉換函式庫
│   ├── iface.go              # convlib 共用介面
│   ├── ddr5conv/             # DDR5 轉換實作
│   │   ├── cmd.go
│   │   ├── printer.go
│   │   ├── macro/            # DDR5 macro 定義
│   │   └── statusreg/        # DDR5 mode register 定義
│   │       └── mrstatusreg/  # MR0/MR1/MR2/MR8/MR30 等
│   └── lpddr5conv/           # LPDDR5 轉換實作
│       ├── cmd.go
│       ├── printer.go
│       └── macro/
├── parser/                   # 通用 pattern parser（骨架）
│   └── testcase/
├── testcase/                 # patconverter 層級的測試資料
│   └── newformat/
└── timingcheck/              # 時序檢查與轉換主流程
    ├── main.go
    ├── testcase/             # 給 timingcheck 用的 .seq 測試檔
    │   ├── ddr5/
    │   └── lpddr5/
    └── ddr5checker/          # DDR5 timing checker 實作
        ├── core/             # .seq 解析、展開與變數管理
        ├── model/            # 共用資料模型
        ├── printer/          # 將檢查結果轉換並輸出
        └── regcontroller/    # 不同 tester 平台的暫存器寫入策略
```

---

## 子套件 (subpackages)

### `convlib`

各種 DRAM 規格的轉換函式庫，提供 command / macro / mode-register 的定義與印出邏輯。

- `convlib/iface.go`：所有 `*conv` 子套件共用的介面。
- `convlib/ddr5conv`：DDR5 的 `cmd.go`、`printer.go`，並在 `macro/macrodef.go` 內維護 macro 定義；`statusreg/mrstatusreg` 提供 DDR5 各 MR (MR0/MR1/MR2/MR8/MR30) 的結構與 `cmd/generate_mr.go` 產生器。
- `convlib/lpddr5conv`：LPDDR5 對應的 `cmd.go`、`printer.go`、`macro/macrodef.go`。

### `parser`

`pattern` / `sequence` 的通用解析器，目前為骨架，預留給未來統一 DDR5 / LPDDR5 / LPDDR6 等共用的解析流程。

### `testcase`

`patconverter` 層級的測試資料目錄（含 `newformat/`），與各子套件自帶的 testcase 分開存放。

### `timingcheck`

`patconverter` 目前內容最完整的子套件，負責讀取 `.seq` 測試檔、做 DDR5 時序檢查、再轉換成測試機台所需的輸出格式。內部以 `ddr5checker` 為主要實作。

---

## `timingcheck/ddr5checker` 細部說明

`ddr5checker` 將「序列檔解析 → 命令展開 → 時序檢查 → 多通道輸出」拆成四個責任分明的子套件。

### `core` — 序列解析與展開

- `seq_parser.go`：將 `.seq` 文字檔解析為 AST 節點 `SeqNode`，節點型別包含 `NodeCommand`、`NodeLoop`、`NodeSpecialTag`。
- `seq_expander.go`：將 AST 展開成扁平、可逐 cycle 執行的 `model.Command` 序列，處理 loop 展開、`SPECIAL_TAG` 等。
- `variablestroe.go`：序列中變數 (variable) 的儲存與查詢，支援 `SPECIAL_TAG` 的 assign / incremental 操作。

### `model` — 共用資料模型

- `types.go`：核心型別
  - `Command`：一條展開後的命令，攜帶 `Cycle`、`Type`、`BankGroup` / `Bank` / `Row` / `Column` / `MA` / `OpCode` / `CW`、DQ 相關欄位、`RepeatCnt`、原始字串等。
  - `BankKey`：`(BankGroup, Bank)` 的唯一鍵，搭配 `Command.GetBankKey()` 使用。
  - `TimingViolation` / `CheckResult`：時序違規記錄與檢查結果（提供 `HasViolations`、`ViolationsForLine`、`ViolationsForCycle`）。
- `command_type.go`、`command_type_ddr5.go`、`command_type_lpddr5.go`：DDR5 / LPDDR5 命令型別列舉與對應屬性。
- `command_parser.go`：將原始欄位轉為 `Command` 結構。
- `format.go`、`subrow.go`：輸出層使用的 `PrintNode` / `SubRow` / `Formatter` / `VarMapper` 等抽象。
- `mrstatusreg/`：DDR5 與 LPDDR5 的 Mode Register 結構，依規格分到 `ddr5/` 與 `lpddr5/` 子目錄，各自附 `cmd/generate_mr.go` 產生器。

### `printer` — 結果轉換與輸出

`printer` 將 `core` 產出的 `[]*SeqNode` 與 `model.CheckResult` 一起轉成 `[]*model.PrintNode`，再透過對應的 `Formatter` 印出實際輸出檔。

主要型別：
- `basePrinter` / `wrappedPrinter`：所有 printer 的基底，持有 `*model.Config`、`*model.CheckResult`、way count 與 `PrinterStrategy`。
  - `Convert(nodes)` / `ConvertWithCheck(nodes, checkResult)`：核心轉換入口。
- `PrinterStrategy`（`printer_strategy.go` / `printer_single.go` / `printer_multi.go`）：依 way count 切換單通道 / 多通道輸出策略。
- `LoopStrategy`（`loop_strategy.go` / `loop_single.go` / `loop_default.go` / `loop_helper.go`）：處理 loop 展開時的不同行為。
- `accumulator.go`：累積轉換過程中的 `PrintNode`，並在 flush 時檢查是否有未綁定的 `SPECIAL_TAG`。
- `command.go`：command 層級的輸出輔助。
- 與多個 tester 平台連動：`hs5503/ddr5`、`m5ssv/ddr5`、`t5833/ddr5`、`t5833/lpddr5`。
- 測試：`printer_5503_test.go`、`printer_5833_test.go`。

### `regcontroller` — 機台暫存器寫入

`regcontroller` 負責把 `Command` 對應到實際機台 (tester) 的暫存器寫入動作。

- `regoperator.go`：`RegOperator` 為主要操作器，透過 `Option` 注入不同的 `CommandWriter`；`Apply(nodes)` 會走訪每個 `PrintNode` 與 `SubRow`，呼叫 `cmdWriter.WriteCommand` 寫入。
- `defaultwriter.go`：預設的 `CommandWriter` 實作。
- `pnode_operator.go`：`PrintNode` 層級的輔助操作。
- `transform.go`：跨平台共用的 transform 工具。
- 平台子目錄，每個目錄都提供自己的 `cmdwriter.go`、`formater.go` 與 macro 定義：
  - `hs5503/ddr5/`：HS5503 + DDR5。
  - `hs5503/lpddr6/{uhs,ui1,ui2}/`：HS5503 + LPDDR6 的不同 UI rate 版本，並各自帶 `macrodef*.asc` / `macrodef_uhs.asc`。
  - `m5ssv/ddr5/`：M5SSV + DDR5。
  - `t5833/ddr5/`、`t5833/lpddr5/`：T5833 + DDR5 / LPDDR5。

### `testcase`

提供給 `timingcheck` 測試使用的 `.seq` 檔，依規格分目錄：

- `testcase/ddr5/`：含 `small_test.seq`、`small_test2.seq`、`loop_test.seq`、`variable_test.seq`、`checkin_bl16_bc8_otf.seq`、`checkin_burst_order.seq`、`457_bssbfcn.seq`、`372_bbsbfcn3.seq`，以及 `memo/`、`overview/` 文件目錄。
- `testcase/lpddr5/`：LPDDR5 對應的測試序列。

---

## 典型資料流

```
.seq 檔
   │
   ▼
core.SeqParser  ──►  []*core.SeqNode
   │
   ▼
core.SeqExpander  ──►  []*model.Command   （展開 loop / special tag / variable）
   │
   ▼
ddr5checker (timing check)  ──►  *model.CheckResult（含 TimingViolation）
   │
   ▼
printer.basePrinter.ConvertWithCheck  ──►  []*model.PrintNode
   │
   ▼
regcontroller.RegOperator.Apply  ──►  各機台 CommandWriter 寫入暫存器 / 輸出檔
```

---

## Build / Test

`patconverter` 隨整個 `waveconv2` 一起編譯與測試：

```bash
# 編譯整個 module
go build ./...

# 只跑 patconverter 相關測試
go test ./pkg/patconverter/...
```

或使用 repo 根目錄的 `makefile`。
