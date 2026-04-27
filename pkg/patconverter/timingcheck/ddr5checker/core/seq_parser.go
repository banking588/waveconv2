package core

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

// ========================================
// SeqNode: 序列文件的 AST 節點
// ========================================

type SeqNodeType int

const (
	NodeCommand SeqNodeType = iota
	NodeLoop
	NodeSpecialTag
)

type SeqNode struct {
	Type     SeqNodeType
	LineNum  int
	Raw      string
	Children []*SeqNode

	// NodeCommand — 解析前暫存原始欄位，展開後由 Cmd 取代。
	CmdFields []string

	// NodeSpecialTage data
	SpecialTagDQSetting []string

	// NodeCommand — 展開後的結構化指令。
	Cmd *model.Command

	// NodeLoop
	LoopCount int
	LoopName  string

	// NodeSpecialTag
	ExprString string

	//TagExpr的判斷
	SpecialTagOpType SpecialTagOperatorType

	//SkipThisNode
	// 假設這個SpecialTag從上一次宣告到當前行都沒變動過, 則必略該node, 不要建到[]SeqNode裡
	//TODO: 還沒做, 先用S的標記處理
	SkipThisSpecialTagNode bool
}

// ========================================
// ParseSeqFile: 解析序列文件為 AST
// ========================================

func ParseSeqFile(reader io.Reader) ([]*SeqNode, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lines []rawSeqLine
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		lines = append(lines, rawSeqLine{lineNum: lineNum, content: trimmed})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	nodes, _, err := parseNodes(lines, 0)
	return nodes, err

}

type rawSeqLine struct {
	lineNum int
	content string
}

func parseNodes(lines []rawSeqLine, start int) ([]*SeqNode, int, error) {
	var nodes []*SeqNode
	i := start

	for i < len(lines) {
		line := lines[i]
		content := line.content

		if content == "}" {
			return nodes, i + 1, nil
		}

		if strings.HasPrefix(content, "PARA_LOOP_START") {
			loopCount, loopName, err := parseLoopHeader(content)
			if err != nil {
				return nil, 0, fmt.Errorf("line %d: %w", line.lineNum, err)
			}

			children, nextIdx, err := parseNodes(lines, i+1)
			if err != nil {
				return nil, 0, err
			}

			nodes = append(nodes, &SeqNode{
				Type:      NodeLoop,
				LineNum:   line.lineNum,
				Raw:       content,
				LoopCount: loopCount,
				LoopName:  loopName,
				Children:  children,
			})
			i = nextIdx
			continue
		}

		if strings.HasPrefix(content, "SPECIAL_TAG") {
			if strings.HasSuffix(content, "S") {
				//SPECIAL_TAG BA_1 = 0x0000 S 尾符有S跳过
				i++
				continue
			}
			expr := strings.TrimPrefix(content, "SPECIAL_TAG")
			expr = strings.TrimSpace(expr)
			nodes = append(nodes, &SeqNode{
				Type:       NodeSpecialTag,
				LineNum:    line.lineNum,
				Raw:        content,
				ExprString: expr,
			})
			i++
			continue
		}

		fields := strings.Fields(content)
		nodes = append(nodes, &SeqNode{
			Type:      NodeCommand,
			LineNum:   line.lineNum,
			Raw:       content,
			CmdFields: fields,
		})
		i++
	}

	return nodes, i, nil
}

// parseLoopHeader 解析迴圈標頭
//
// 支持格式:
//
//	PARA_LOOP_START 40064 {
//	PARA_LOOP_START 128 : WRRDJMP {
//	PARA_LOOP_START 98304 : SOME_NAME {
//
// 回傳 (loopCount, loopName, error)
// loopName 為空字串表示沒有名稱
func parseLoopHeader(content string) (int, string, error) {
	// 去掉尾部的 {
	content = strings.TrimSuffix(strings.TrimSpace(content), "{")
	content = strings.TrimSpace(content)

	// 去掉 "PARA_LOOP_START"
	content = strings.TrimPrefix(content, "PARA_LOOP_START")
	content = strings.TrimSpace(content)

	// 檢查是否有 ": NAME"
	loopName := ""
	if colonIdx := strings.Index(content, ":"); colonIdx >= 0 {
		loopName = strings.TrimSpace(content[colonIdx+1:])
		content = strings.TrimSpace(content[:colonIdx])
	}

	if content == "" {
		return 0, "", fmt.Errorf("PARA_LOOP_START requires loop count")
	}

	count, err := strconv.Atoi(content)
	if err != nil {
		return 0, "", fmt.Errorf("invalid loop count: %s", content)
	}

	return count, loopName, nil
}
