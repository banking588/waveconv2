package regcontroller

import (
	"strings"

	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

// MRWIndex 記錄一個 MRW/MPC 命令在 []PrintNode 中的位置
type MRWIndex struct {
	NodeIndex   int
	SubRowIndex int
	Macro       string
}

// MRWGroup 記錄一組連續的 MRW/MPC 命令
type MRWGroup struct {
	Entries []MRWIndex
}

// TransformPrintNodes 主入口：三階段處理。wayCount 由外部傳入。
func TransformPrintNodes(nodes []*model.PrintNode) []*model.PrintNode {
	op := NewPNodeOperator(nodes)

	// Phase 1: 收集所有 MRW/MPC 的位置與分組
	groups := collectMRWGroups(op)

	// Phase 1.5: 把跨 PNode 的同組 MRW/MPC 透過 swap 合併到同一個 Node
	groups = mergeCrossNodeGroups(op, groups)

	// Phase 2: 根據分組資訊，拆分/補 pad
	applyTransform(op, groups)

	return op.Nodes
}

// Phase1: collectMRWGroups 遍歷所有 PrintNode，找出所有 MRW/MPC 的位置並分組。
func collectMRWGroups(op *PNodeOperator) []MRWGroup {
	var groups []MRWGroup
	var currentGroup *MRWGroup

	for ni := 0; ni < op.NodeCount(); ni++ {
		node := op.GetNode(ni)
		for si, sr := range node.SubRows {
			if isMRWorMPC(sr.Macro) {
				entry := MRWIndex{
					NodeIndex:   ni,
					SubRowIndex: si,
					Macro:       sr.Macro,
				}
				if currentGroup == nil {
					currentGroup = &MRWGroup{}
				}
				currentGroup.Entries = append(currentGroup.Entries, entry)
			} else {
				if currentGroup != nil {
					groups = append(groups, *currentGroup)
					currentGroup = nil
				}
			}
		}
	}

	if currentGroup != nil {
		groups = append(groups, *currentGroup)
	}

	return groups
}

// Phase1.5: mergeCrossNodeGroups 把跨 PNode 的同組 MRW/MPC 透過 swap 合併。
func mergeCrossNodeGroups(op *PNodeOperator, groups []MRWGroup) []MRWGroup {
	for g := range groups {
		group := &groups[g]
		if len(group.Entries) <= 1 {
			continue
		}

		targetNI := group.Entries[0].NodeIndex
		for _, e := range group.Entries {
			if e.NodeIndex > targetNI {
				targetNI = e.NodeIndex
			}
		}

		allSame := true
		for _, e := range group.Entries {
			if e.NodeIndex != targetNI {
				allSame = false
				break
			}
		}
		if allSame {
			continue
		}

		// Step 1: 在 targetNI 內部騰位
		needSlots := 0
		for _, e := range group.Entries {
			if e.NodeIndex < targetNI {
				needSlots++
			}
		}

		for i := len(group.Entries) - 1; i >= 0; i-- {
			entry := &group.Entries[i]
			if entry.NodeIndex != targetNI {
				continue
			}
			op.MoveSubRowDown(targetNI, entry.SubRowIndex, needSlots)
			entry.SubRowIndex += needSlots
		}

		// Step 2: 跨 Node swap
		for i := range group.Entries {
			entry := &group.Entries[i]
			if entry.NodeIndex == targetNI {
				continue
			}

			for entry.NodeIndex < targetNI {
				op.MoveSubRowToNextNode(entry.NodeIndex, entry.SubRowIndex)
				entry.NodeIndex++
				entry.SubRowIndex = 0
			}
		}
	}

	return groups
}

// Phase2: applyTransform 根據分組資訊拆分、補 pad。
func applyTransform(op *PNodeOperator, groups []MRWGroup) {
	if len(groups) <= 1 {
		return
	}

	for g := len(groups) - 1; g >= 1; g-- {
		group := groups[g]
		firstEntry := group.Entries[0]

		ni := firstEntry.NodeIndex
		si := firstEntry.SubRowIndex

		if si > 0 {
			op.SplitNodeAt(ni, si)
			op.PadAtTail(ni)
			op.PadAfterLastMRWGroup(ni + 1)
			updateGroupIndices(groups, g, ni, si)
		} else if si == 0 {
			op.PadAfterLastMRWGroup(ni)
		}
	}
}

func updateGroupIndices(groups []MRWGroup, fromGroup int, origNodeIndex int, cutSubRowIndex int) {
	for g := fromGroup; g < len(groups); g++ {
		for e := range groups[g].Entries {
			entry := &groups[g].Entries[e]
			if entry.NodeIndex == origNodeIndex && entry.SubRowIndex >= cutSubRowIndex {
				entry.NodeIndex = origNodeIndex + 1
				entry.SubRowIndex -= cutSubRowIndex
			} else if entry.NodeIndex > origNodeIndex {
				entry.NodeIndex++
			}
		}
	}
}

func isMRWorMPC(macro string) bool {
	return strings.HasPrefix(macro, "MRW")
}

func makePaddingSubRow() model.SubRow {
	return model.SubRow{
		Macro:  "D_",
		Status: model.SubRowStatusAutoPad,
	}
}
