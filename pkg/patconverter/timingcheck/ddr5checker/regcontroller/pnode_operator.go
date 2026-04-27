package regcontroller

import "waveconv/pkg/patconverter/timingcheck/ddr5checker/model"

// PNodeOperator 提供對 []*model.PrintNode 的各種操作方法
type PNodeOperator struct {
	Nodes     []*model.PrintNode
	SubRowCap int // 每個 PNode 固定的 SubRow 數量，由 wayCount 決定
}

func NewPNodeOperator(nodes []*model.PrintNode) *PNodeOperator {
	subRowCap := 0
	if len(nodes) > 0 {
		subRowCap = cap(nodes[0].SubRows)
	}
	return &PNodeOperator{
		Nodes:     nodes,
		SubRowCap: subRowCap,
	}
}

// =============================================================================
// Node 層級操作
// =============================================================================

// InsertNode 在指定 index 插入一個新的 PrintNode，後面的 Node 往後移
func (op *PNodeOperator) InsertNode(index int, node *model.PrintNode) {
	op.Nodes = append(op.Nodes, nil)
	copy(op.Nodes[index+1:], op.Nodes[index:])
	op.Nodes[index] = node
}

// overflow 用 make([]SubRow, 0, op.SubRowCap)，保持固定容量
func (op *PNodeOperator) SplitNodeAt(nodeIndex int, subRowIndex int) *model.PrintNode {
	node := op.Nodes[nodeIndex]

	// 切走的部分，用固定容量
	overflow := make([]model.SubRow, 0, op.SubRowCap)
	overflow = append(overflow, node.SubRows[subRowIndex:]...)

	// 原 Node 截斷，保持原容量
	node.SubRows = node.SubRows[:subRowIndex]

	newNode := &model.PrintNode{
		Header:  node.Header,
		SubRows: overflow,
	}

	op.InsertNode(nodeIndex+1, newNode)
	return newNode
}

// NodeCount 回傳目前的 Node 數量
func (op *PNodeOperator) NodeCount() int {
	return len(op.Nodes)
}

// GetNode 取得指定 index 的 Node
func (op *PNodeOperator) GetNode(nodeIndex int) *model.PrintNode {
	return op.Nodes[nodeIndex]
}

// SubRowCount 取得指定 Node 的 SubRow 數量
func (op *PNodeOperator) SubRowCount(nodeIndex int) int {
	return len(op.Nodes[nodeIndex].SubRows)
}

// =============================================================================
// SubRow 層級操作
// =============================================================================

// SwapSubRows 在同一個 Node 內交換兩個 SubRow
func (op *PNodeOperator) SwapSubRows(nodeIndex int, si1, si2 int) {
	srs := op.Nodes[nodeIndex].SubRows
	srs[si1], srs[si2] = srs[si2], srs[si1]
}

// SwapSubRowAcrossNodes 跨 Node 交換兩個 SubRow
func (op *PNodeOperator) SwapSubRowAcrossNodes(nodeIndex1, si1, nodeIndex2, si2 int) {
	op.Nodes[nodeIndex1].SubRows[si1], op.Nodes[nodeIndex2].SubRows[si2] =
		op.Nodes[nodeIndex2].SubRows[si2], op.Nodes[nodeIndex1].SubRows[si1]
}

// MoveSubRowDown 把指定 SubRow 在同一個 Node 內往下移動 n 格（透過連續 swap）
func (op *PNodeOperator) MoveSubRowDown(nodeIndex int, subRowIndex int, n int) {
	srs := op.Nodes[nodeIndex].SubRows
	for i := 0; i < n; i++ {
		cur := subRowIndex + i
		if cur+1 < len(srs) {
			srs[cur], srs[cur+1] = srs[cur+1], srs[cur]
		}
	}
}

// MoveSubRowUp 把指定 SubRow 在同一個 Node 內往上移動 n 格（透過連續 swap）
func (op *PNodeOperator) MoveSubRowUp(nodeIndex int, subRowIndex int, n int) {
	srs := op.Nodes[nodeIndex].SubRows
	for i := 0; i < n; i++ {
		cur := subRowIndex - i
		if cur-1 >= 0 {
			srs[cur], srs[cur-1] = srs[cur-1], srs[cur]
		}
	}
}

// MoveSubRowToTail 把指定 SubRow 在同一個 Node 內 swap 到尾部
func (op *PNodeOperator) MoveSubRowToTail(nodeIndex int, subRowIndex int) {
	srs := op.Nodes[nodeIndex].SubRows
	for j := subRowIndex; j < len(srs)-1; j++ {
		srs[j], srs[j+1] = srs[j+1], srs[j]
	}
}

// MoveSubRowToHead 把指定 SubRow 在同一個 Node 內 swap 到頭部
func (op *PNodeOperator) MoveSubRowToHead(nodeIndex int, subRowIndex int) {
	srs := op.Nodes[nodeIndex].SubRows
	for j := subRowIndex; j > 0; j-- {
		srs[j], srs[j-1] = srs[j-1], srs[j]
	}
}

// MoveSubRowToNextNode 把 SubRow 從當前 Node 移到下一個 Node 的頭部
// 先在當前 Node 內 swap 到尾部，再跨 Node swap
func (op *PNodeOperator) MoveSubRowToNextNode(nodeIndex int, subRowIndex int) {
	op.MoveSubRowToTail(nodeIndex, subRowIndex)

	lastIdx := len(op.Nodes[nodeIndex].SubRows) - 1
	op.SwapSubRowAcrossNodes(nodeIndex, lastIdx, nodeIndex+1, 0)
}

// MoveSubRowToPrevNode 把 SubRow 從當前 Node 移到上一個 Node 的尾部
// 先在當前 Node 內 swap 到頭部，再跨 Node swap
func (op *PNodeOperator) MoveSubRowToPrevNode(nodeIndex int, subRowIndex int) {
	op.MoveSubRowToHead(nodeIndex, subRowIndex)

	prevLastIdx := len(op.Nodes[nodeIndex-1].SubRows) - 1
	op.SwapSubRowAcrossNodes(nodeIndex, 0, nodeIndex-1, prevLastIdx)
}

// 用 cap 判斷，超出後截斷 len 但保留 cap
func (op *PNodeOperator) InsertSubRow(nodeIndex int, subRowIndex int, sr model.SubRow) {
	node := op.Nodes[nodeIndex]

	node.SubRows = append(node.SubRows, model.SubRow{})
	copy(node.SubRows[subRowIndex+1:], node.SubRows[subRowIndex:])
	node.SubRows[subRowIndex] = sr

	// 沒有超出 cap，不需要溢出
	if len(node.SubRows) <= op.SubRowCap {
		return
	}

	// 超出 cap，取出最後一個，截斷回固定長度
	overflow := node.SubRows[op.SubRowCap]
	node.SubRows = node.SubRows[:op.SubRowCap]

	if nodeIndex+1 < len(op.Nodes) {
		op.InsertSubRow(nodeIndex+1, 0, overflow)
	}
}

// 保持 len 不超過 cap
func (op *PNodeOperator) RemoveSubRow(nodeIndex int, subRowIndex int) model.SubRow {
	node := op.Nodes[nodeIndex]

	removed := node.SubRows[subRowIndex]
	node.SubRows = append(node.SubRows[:subRowIndex], node.SubRows[subRowIndex+1:]...)

	if nodeIndex+1 < len(op.Nodes) && len(node.SubRows) < op.SubRowCap {
		pulled := op.RemoveSubRow(nodeIndex+1, 0)
		node.SubRows = append(node.SubRows, pulled)
	} else if len(node.SubRows) < op.SubRowCap {
		node.SubRows = append(node.SubRows, makePaddingSubRow())
	}

	return removed
}

// =============================================================================
// Padding 操作
// =============================================================================

// cap 已固定，append 不會重新分配
func (op *PNodeOperator) PadAtTail(nodeIndex int) {
	node := op.Nodes[nodeIndex]
	for len(node.SubRows) < op.SubRowCap {
		node.SubRows = append(node.SubRows, makePaddingSubRow())
	}
}

// 保持固定容量操作
func (op *PNodeOperator) PadAfterLastMRWGroup(nodeIndex int) {
	node := op.Nodes[nodeIndex]
	if len(node.SubRows) >= op.SubRowCap {
		return
	}

	lastMRW := -1
	for i := len(node.SubRows) - 1; i >= 0; i-- {
		if isMRWorMPC(node.SubRows[i].Macro) {
			lastMRW = i
			break
		}
	}

	if lastMRW < 0 {
		op.PadAtTail(nodeIndex)
		return
	}

	insertAt := lastMRW + 1
	padCount := op.SubRowCap - len(node.SubRows)

	// 先把現有的 tail 暫存
	tail := make([]model.SubRow, len(node.SubRows[insertAt:]))
	copy(tail, node.SubRows[insertAt:])

	// 截斷到插入點
	node.SubRows = node.SubRows[:insertAt]

	// 插入 pad
	for i := 0; i < padCount; i++ {
		node.SubRows = append(node.SubRows, makePaddingSubRow())
	}

	// 放回 tail
	node.SubRows = append(node.SubRows, tail...)

	// 確保不超過 cap
	if len(node.SubRows) > op.SubRowCap {
		node.SubRows = node.SubRows[:op.SubRowCap]
	}
}

// =============================================================================
// 查詢操作
// =============================================================================

// FindSubRow 在指定 Node 中從 startIndex 開始找符合條件的 SubRow，回傳 index，找不到回傳 -1
func (op *PNodeOperator) FindSubRow(nodeIndex int, startIndex int, predicate func(model.SubRow) bool) int {
	srs := op.Nodes[nodeIndex].SubRows
	for i := startIndex; i < len(srs); i++ {
		if predicate(srs[i]) {
			return i
		}
	}
	return -1
}

// FindLastSubRow 在指定 Node 中從尾部往前找符合條件的 SubRow，回傳 index，找不到回傳 -1
func (op *PNodeOperator) FindLastSubRow(nodeIndex int, predicate func(model.SubRow) bool) int {
	srs := op.Nodes[nodeIndex].SubRows
	for i := len(srs) - 1; i >= 0; i-- {
		if predicate(srs[i]) {
			return i
		}
	}
	return -1
}

// LastNodeSubRowIsMRW 判斷指定 Node 的最後一個 SubRow 是否為 MRW/MPC
func (op *PNodeOperator) LastNodeSubRowIsMRW(nodeIndex int) bool {
	srs := op.Nodes[nodeIndex].SubRows
	if len(srs) == 0 {
		return false
	}
	return isMRWorMPC(srs[len(srs)-1].Macro)
}
