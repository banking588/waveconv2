package regcontroller

import (
	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

type RegOperator struct {
	// Command 一般欄位（Row/Col/BG/BA/MA/OP...）要怎麼塞值, 交給對應的CommandWriter
	cmdWriter CommandWriter
}

type Option func(*RegOperator)

func WithCommandWriter(w CommandWriter) Option {
	return func(ro *RegOperator) { ro.cmdWriter = w }
}

func NewRegOperator(opts ...Option) *RegOperator {
	ro := &RegOperator{
		///可自行加入各种不同的command writeer
		cmdWriter: DefaultCommandWriter{},
	}
	for _, opt := range opts {
		opt(ro)
	}
	return ro
}

func (ro *RegOperator) Apply(nodes []*model.PrintNode) {
	if len(nodes) == 0 {
		return
	}

	// Loop PrintNode以及Subrow
	for printNodeIdx := range nodes {
		pn := nodes[printNodeIdx]
		for subRowIndex := range pn.SubRows {
			sr := &pn.SubRows[subRowIndex]
			if !isNormalSubRow(sr) {
				continue
			}

			cmd := sr.Cmd
			if cmd == nil {
				//log.Debugf("Non-Command SR %v", sr.Macro)
				continue
			}

			cur := model.Cursor{PN: printNodeIdx, SR: subRowIndex}

			// normal command
			ro.cmdWriter.WriteCommand(nodes, cur, cmd)

			// 如果有其它writer, 可以直接加在这里, 目前只需要一个
			//ro.rwWriter.WriteReadWrite(nodes, cur, cmd)
		}
	}
}

//
// model.Cursor, 記錄遍歷Pnodes, 不拉平PNodes
//

// StepPrev / StepNext：移動 1 格，回傳是否跨 PN
func StepNext(nodes []*model.PrintNode, cur model.Cursor) (next model.Cursor, crossedPN bool, ok bool) {
	if cur.PN < 0 || cur.PN >= len(nodes) {
		return model.Cursor{}, false, false
	}
	pn := nodes[cur.PN]
	if cur.SR+1 < len(pn.SubRows) {
		return model.Cursor{PN: cur.PN, SR: cur.SR + 1}, false, true
	}
	np := cur.PN + 1
	for np < len(nodes) && len(nodes[np].SubRows) == 0 {
		np++
	}
	if np >= len(nodes) {
		return model.Cursor{}, false, false
	}
	return model.Cursor{PN: np, SR: 0}, true, true
}

func StepPrev(nodes []*model.PrintNode, cur model.Cursor) (prev model.Cursor, crossedPN bool, ok bool) {
	if cur.PN < 0 || cur.PN >= len(nodes) {
		return model.Cursor{}, false, false
	}
	if cur.SR-1 >= 0 {
		return model.Cursor{PN: cur.PN, SR: cur.SR - 1}, false, true
	}
	pp := cur.PN - 1
	for pp >= 0 && len(nodes[pp].SubRows) == 0 {
		pp--
	}
	if pp < 0 {
		return model.Cursor{}, false, false
	}
	return model.Cursor{PN: pp, SR: len(nodes[pp].SubRows) - 1}, true, true
}

// NextNormal / PrevNormal：跳過 violation Subrow, 这是用来记录core timing项的
func NextNormal(nodes []*model.PrintNode, cur model.Cursor) (next model.Cursor, crossedPN bool, ok bool) {
	c := cur
	for {
		nc, crossed, ok := StepNext(nodes, c)
		if !ok {
			return model.Cursor{}, false, false
		}
		if isNormalSubRow(&nodes[nc.PN].SubRows[nc.SR]) {
			return nc, crossed, true
		}
		c = nc
	}
}

func PrevNormal(nodes []*model.PrintNode, cur model.Cursor) (prev model.Cursor, crossedPN bool, ok bool) {
	c := cur
	for {
		pc, crossed, ok := StepPrev(nodes, c)
		if !ok {
			return model.Cursor{}, false, false
		}
		if isNormalSubRow(&nodes[pc.PN].SubRows[pc.SR]) {
			return pc, crossed, true
		}
		c = pc
	}
}

func isNormalSubRow(sr *model.SubRow) bool {
	return sr.Violation == nil
}

func appendAddr(sr *model.SubRow, addrtokens []string) {
	if len(addrtokens) == 0 {
		return
	}
	sr.Addr = append(sr.Addr, addrtokens...)
}

func appendData(sr *model.SubRow, datatokens []string) {
	if len(datatokens) == 0 {
		return
	}
	sr.Data = append(sr.Data, datatokens...)
}

type FillMode int

const (
	// token 是一数组, 会因macro command不同而变化

	//分3种情况

	//情况1: FillBatch：所有 tokens 寫入同一批 subrow
	FillBatch FillMode = iota
	//情况2: FillSpread：每個 token 帶自己的 offset，分發到指定 subrow
	FillSpread
	//情况3:  FillRW：WR/RD 專用，IncludeSelf=true，從自身開始連續逐一塞值
	FillRW

	//以 PN, SR寫入, 要自己算好位置
	FillByPN
)

// SpreadToken：情況 2 用，每個 token 自帶要打到哪個 subrow 的 offset
// Offset > 0 表示往下，Offset < 0 表示往上（相對於 Cursor）
type SpreadToken struct {
	Value  string
	Offset int // 相對 Cursor 的偏移量，例如 -1 = Cursor-1, -3 = Cursor-3
}

// FillByPN 指定PN, SR位置
type PNTarget struct {
	PN    int
	SR    int
	Value string
}

type FillPlan struct {
	Up   int //往上打, 0的话就是不动作
	Down int //往下打

	IncludeSelf            bool //本身行要不要打
	StopWhenCrossPrintNode bool

	Mode      FillMode
	FillBatch bool

	// 情況 2 專用：每個 token 自帶 offset
	SpreadTokens []SpreadToken

	// 情況 3 專用：tokens 中哪個 index 是自身（錨點）
	// 例如 tokens = [LL, LL, HL, HL]，SelfIndex = 2 表示 HL 打在 Cursor 自身
	SelfIndex int

	// Batch 專用：當 command 本身跨了 PrintNode 時，
	// 若為 true，額外把 tokens 塞到上一個 PN 的最後一個 normal subrow
	CrossPNFallback bool
	//对于FillBatch用的command（MPC, MRW, MRR, VrefCA, VrefCS）总是需要将赋值移动到上一个NOP的最后一行
	OverrideCrossPNFallback bool
	WriteToData             bool // true 表示写入 sr.Data，false 表示写入 sr.Addr

	// FillByPN 專用：以絕對索引指定
	PNTargets []PNTarget
}

func appendTokens(sr *model.SubRow, tokens []string, writeToData bool) {
	if writeToData {
		appendData(sr, tokens)
	} else {
		appendAddr(sr, tokens)
	}
}

// ApplyFillPlan：把 tokens 依 plan 寫入（不 flatten）
// 每次移動都會知道 crossedPN
func ApplyFillPlan(nodes []*model.PrintNode, at model.Cursor, tokens []string, plan FillPlan) {
	if len(tokens) == 0 && len(plan.SpreadTokens) == 0 {
		//return
		log.Warnf("Empty fill tokens at PN %v", at.PN)
	}

	switch plan.Mode {
	case FillSpread:
		applyFillSpread(nodes, at, plan)
	case FillRW:
		applyFillRW(nodes, at, tokens, plan)
	case FillByPN:
		applyFillByPN(nodes, plan)
	default:
		applyFillBatch(nodes, at, tokens, plan)
	}
}

// ============================================================
//
// 情況 1：applyFillBatch：所有 tokens 一次寫入同一個 subrow
//
//	但要考虑command是否跨了Pnode
//
// ============================================================
func applyFillBatch(nodes []*model.PrintNode, at model.Cursor, tokens []string, plan FillPlan) {
	if plan.IncludeSelf {
		appendAddr(&nodes[at.PN].SubRows[at.SR], tokens)
	}

	// 向上
	upHandled := false

	// OverrideCrossPNFallback: 总是将内容塞到上一个 PN 的最后一个 normal subrow
	if plan.OverrideCrossPNFallback {
		prevPN := findLastNormalInPrevPN(nodes, at.PN)
		if prevPN != nil {
			appendAddr(&nodes[prevPN.PN].SubRows[prevPN.SR], tokens)
		}
		upHandled = true
	}

	// CrossPNFallback：判斷 command 是否跨 PN
	// 从 Cursor 往下找下一個 normal subrow，若在不同 PN => 跨了 PN
	if !plan.OverrideCrossPNFallback && plan.CrossPNFallback && isCommandCrossPN(nodes, at) {
		// 跨 PN → 取代 Up 邏輯，塞到上一個 PN 的最後一個 normal subrow
		prevPN := findLastNormalInPrevPN(nodes, at.PN)
		if prevPN != nil {
			appendAddr(&nodes[prevPN.PN].SubRows[prevPN.SR], tokens)
		}
		upHandled = true
	}

	// 沒跨 PN 或沒開 CrossPNFallback
	if !upHandled {
		cur := at
		for written := 0; written < plan.Up; written++ {
			pc, crossed, ok := PrevNormal(nodes, cur)
			if !ok {
				break
			}
			if plan.StopWhenCrossPrintNode && crossed {
				break
			}
			appendAddr(&nodes[pc.PN].SubRows[pc.SR], tokens)
			cur = pc
		}
	}

	// 向下
	cur := at
	for written := 0; written < plan.Down; written++ {
		nc, crossed, ok := NextNormal(nodes, cur)
		if !ok {
			break
		}
		if plan.StopWhenCrossPrintNode && crossed {
			break
		}
		appendAddr(&nodes[nc.PN].SubRows[nc.SR], tokens)
		cur = nc
	}
}

// isCommandCrossPN：從 Cursor 往下找下一個 normal subrow，若在不同 PN 則代表 command 跨了 PN
func isCommandCrossPN(nodes []*model.PrintNode, at model.Cursor) bool {
	next, _, ok := NextNormal(nodes, at)
	if !ok {
		return false
	}
	return next.PN != at.PN
}

// findLastNormalInPrevPN：往前找上一個 PN 的最後一個 normal subrow
func findLastNormalInPrevPN(nodes []*model.PrintNode, pnIdx int) *model.Cursor {
	for p := pnIdx - 1; p >= 0; p-- {
		pn := nodes[p]
		for i := len(pn.SubRows) - 1; i >= 0; i-- {
			if isNormalSubRow(&pn.SubRows[i]) {
				return &model.Cursor{PN: p, SR: i}
			}
		}
	}
	return nil
}

// ============================================================
//
// 情況 2：Spread — 每個 token 帶 offset，user 自己決定打到哪個 subrow
func applyFillSpread(nodes []*model.PrintNode, at model.Cursor, plan FillPlan) {
	// ============================================================
	for _, st := range plan.SpreadTokens {
		target, ok := moveCursor(nodes, at, st.Offset, plan.StopWhenCrossPrintNode)
		if !ok {
			continue
		}
		appendAddr(&nodes[target.PN].SubRows[target.SR], []string{st.Value})
	}
}

// moveCursor：從 at 出發，移動 offset(正=往下，負=往上)
func moveCursor(nodes []*model.PrintNode, at model.Cursor, offset int, stopOnCross bool) (model.Cursor, bool) {
	cur := at
	if offset == 0 {
		return cur, true
	}

	steps := offset
	if steps < 0 {
		steps = -steps
	}

	for i := 0; i < steps; i++ {
		var next model.Cursor
		var crossed, ok bool
		if offset > 0 {
			next, crossed, ok = NextNormal(nodes, cur)
		} else {
			next, crossed, ok = PrevNormal(nodes, cur)
		}
		if !ok {
			return model.Cursor{}, false
		}
		if stopOnCross && crossed {
			return model.Cursor{}, false
		}
		cur = next
	}
	return cur, true
}

// ============================================================
//
// 情況 3：RW
//
//	tokens = [LL, LL, HL, HL],  SelfIndex = 2
//
//	Cursor-2 → LL  (tokens[0])   <- 最遠
//	Cursor-1 → LL  (tokens[1])   <- 次近
//	Cursor   → HL  (tokens[2])   <- write or read自身(錨點)
//	Cursor+1 → HL  (tokens[3])   <- 往下
//
//	所有移動都可以跨 PrintNode，碰到邊界自動繼續找。
//
// ============================================================
func applyFillRW(nodes []*model.PrintNode, at model.Cursor, tokens []string, plan FillPlan) {
	if len(tokens) == 0 {
		return
	}

	selfIdx := plan.SelfIndex
	if selfIdx < 0 || selfIdx >= len(tokens) {
		// 若 SelfIndex 不合法，預設為 0（等同於全部往下）
		selfIdx = 0
	}

	// 1. 寫入自身
	appendTokens(&nodes[at.PN].SubRows[at.SR], []string{tokens[selfIdx]}, plan.WriteToData)

	// 2. 先往上：tokens[0..selfIdx-1]，由近到遠
	//   tokens[selfIdx-1] → Cursor-1（最近）
	//   tokens[selfIdx-2] → Cursor-2
	//   ...
	//   tokens[0]         → Cursor-N（最遠）
	cur := at
	for i := selfIdx - 1; i >= 0; i-- {
		pc, _, ok := PrevNormal(nodes, cur)
		if !ok {
			break
		}
		appendTokens(&nodes[pc.PN].SubRows[pc.SR], []string{tokens[i]}, plan.WriteToData)
		cur = pc
	}

	// 3. 往下：tokens[selfIdx+1..end]
	//   tokens[selfIdx+1] → Cursor+1（最近）
	//   tokens[selfIdx+2] → Cursor+2
	//   ...
	cur = at
	for i := selfIdx + 1; i < len(tokens); i++ {
		nc, _, ok := NextNormal(nodes, cur)
		if !ok {
			break
		}
		appendTokens(&nodes[nc.PN].SubRows[nc.SR], []string{tokens[i]}, plan.WriteToData)
		cur = nc
	}
}

func applyFillByPN(nodes []*model.PrintNode, plan FillPlan) {
	for _, t := range plan.PNTargets {
		if t.PN < 0 || t.PN >= len(nodes) {
			continue
		}
		pn := nodes[t.PN]
		if t.SR < 0 || t.SR >= len(pn.SubRows) {
			continue
		}
		appendTokens(&pn.SubRows[t.SR], []string{t.Value}, plan.WriteToData)
	}
}
