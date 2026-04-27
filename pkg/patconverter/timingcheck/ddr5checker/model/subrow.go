package model

// SubRowStatus 表示 SubRow 的狀態
type SubRowStatus int

const (
	SubRowStatusNormal     SubRowStatus = 0 // 正常
	SubRowStatusAutoPad    SubRowStatus = 1 // 自動補齊
	SubRowStatusSplitByDes SubRowStatus = 2 // 由 DES 拆分吸收
)

// --- PrintNode：一個標頭 + N 個 SubRow ---
type NodeHeader struct {
	Label         string
	Operator      string
	OperatorValue string
	Interrupt     bool
	LoopCount     int
}

// SubRow, 輸出的最小單位
type SubRow struct {
	Macro              string
	Macros             []string // 多 Macro 組成一個重複單位，展開時每個 Macro 各產生一個 subrow
	Addr               []string
	Settings           []string
	SF                 string
	XYC                string
	Timing             string
	TimingGroup        string
	Cmd                *Command
	Status             SubRowStatus
	Violation          *TimingViolation
	RepeatCnt          int
	SourceLineNum      int // 來自原始文檔的行號
	Data               []string
	PredecessorCommand []string //有些command使用前要在該command前2拍設定額外的前置command
}

// EffectiveMacros 回傳這個 SubRow 展開後的 Macro 列表
func (sr *SubRow) EffectiveMacros() []string {
	if len(sr.Macros) > 0 {
		return sr.Macros
	}
	return []string{sr.Macro}
}

// GroupSize 回傳一組重複單位佔幾個 subrow
func (sr *SubRow) GroupSize() int {
	return len(sr.EffectiveMacros())
}

// IsPadding 向後相容的便利方法
func (sr *SubRow) IsPadding() bool {
	return sr.Status == SubRowStatusAutoPad
}

type PrintNode struct {
	Header        NodeHeader
	SubRows       []SubRow
	SourceLineNum int
	Raw           string
}

// NewPrintNode 建立一個固定容量的 PrintNode，len=0, cap=wayCount
func NewPrintNode(header NodeHeader, wayCount int) *PrintNode {
	return &PrintNode{
		Header:  header,
		SubRows: make([]SubRow, 0, wayCount),
	}
}

// SetLabel 設定此節點的標籤
func (pn *PrintNode) SetLabel(label string) *PrintNode {
	pn.Header.Label = label
	return pn
}

// AddSubRow 追加一個 SubRow 到此節點
func (pn *PrintNode) AddSubRow(sr SubRow) *PrintNode {
	pn.SubRows = append(pn.SubRows, sr)
	return pn
}
