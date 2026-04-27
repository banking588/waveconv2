package printer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	ddr5mr "waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg/ddr5"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller"
	hs2ddr5 "waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller/hs5503/ddr5"
)

// --- Command conversion tests ---

func GET_DDR5_SEQ_FILE_FT() string {
	var fpath = filepath.Join("..", "..", "testcase", "ddr5")

	//var testFile = filepath.Join(fpath, "loop_test.seq")
	//var testFile = filepath.Join(fpath, "small_test.seq")
	var testFile = filepath.Join(fpath, "372_bbsbfcn3.seq")
	//var testFile = filepath.Join(fpath, "457_bssbfcn.seq")
	//var testFile = filepath.Join(fpath, "485_bmds2dq3.seq")
	//var testFile = filepath.Join(fpath, "459_bsdmbkwd.seq")
	//var testFile = filepath.Join(fpath, "checkin_burst_order.seq")
	//var testFile = filepath.Join(fpath, "checkin_bl16_bc8_otf.seq")
	//var testFile = filepath.Join(fpath, "checkin_bl16_bl32_otf.seq")
	return testFile
}

func TestHS5503_File(t *testing.T) {
	log.ApplyDevelopLogger(nil)
	inputBytes, err := os.ReadFile(GET_DDR5_SEQ_FILE_FT())
	if err != nil {
		t.Fatalf("讀取測試輸入檔失敗: %v", err)
	}

	//解析成SeqNode
	input := string(inputBytes)
	nodes, err := core.ParseSeqFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseInput: %v", err)
	}

	//處理SeqNode, 把不必要的節點刪除
	vars := core.NewVariableStore()
	var mrs mrstatusreg.MRS
	mrs = ddr5mr.NewDDR5MRS()
	newNodes, err := core.ParseSeqNodes(nodes, vars, mrs, model.DDR5)

	//把SeqNode轉換成PrintNode, 含macro的mapper
	p, _ := NewWithWay(model.Tester5503HS, 16, model.ProtocolDDR5)
	p.SetMRS(mrs)
	pnodes := p.Convert(newNodes)

	// Reg运算(只需要 PrintNode，不需要 SeqNode)
	// 根據需求實作CommandWriter(ex, macrodef, lp的WCK, ddr的DQS)

	op := regcontroller.NewRegOperator(
		regcontroller.WithCommandWriter(hs2ddr5.NewHS5503CommandWriter(mrs)),
	)

	//先檢查需要加拍的情況, 一般是手寫的才要, 從asc to seq的就不用檢查了
	newPnodes := regcontroller.TransformPrintNodes(pnodes)
	_ = newPnodes

	// 可代入原pnodes, 也可代入加拍後的pnode
	op.Apply(newPnodes)
	output := p.Format(newPnodes)

	t.Logf("HS5503 WR 16-byte DQS spread output:\n%s", output)
}
