package ddr5

import (
	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	ddr5mr "waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg/ddr5"
)

// makeHS5503SubRowCommandMapper 建立閉包版的 SubRowCommandMapper
// mrs 會在 ParseSeqNodes 時被寫入 MRW 的值，
// 之後 printer 呼叫此 mapper 時就能讀到最新的 MR 狀態
func makeHS5503SubRowCommandMapper(mrs mrstatusreg.MRS) model.SubRowCommandMapper {
	return func(cmd *model.Command, wayCount int) []model.SubRow {
		return hs5503MapCommand(cmd, wayCount, mrs)
	}
}

// 从 MRS 中获取 MR0 的 Burst Length
func getBurstLength(mrs mrstatusreg.MRS) int {
	mr0, err := mrstatusreg.GetTyped[*ddr5mr.MR0](mrs, ddr5mr.Name_MR_0)
	if err != nil {
		log.Errorf("Failed to get MR0: %v", err)
	}
	return int(mr0.BurstLength())
}

func hs5503MapCommand(cmd *model.Command, wayCount int, mrs mrstatusreg.MRS) []model.SubRow {

	switch cmd.Type {

	case nativeCmd.DES:
		return []model.SubRow{
			{Macro: "D_", Cmd: cmd, RepeatCnt: cmd.RepeatCnt},
		}

	case nativeCmd.ACT:
		return []model.SubRow{
			{Macro: "ACTL", Cmd: cmd},
			{Macro: "ACTH"},
		}

	case nativeCmd.ACTX:
		return []model.SubRow{
			{Macro: "ACTL", Cmd: cmd},
			{Macro: "ACTH\tC1"},
		}

	case nativeCmd.NOP:
		return []model.SubRow{
			{Macro: "NOOP"},
		}

	case nativeCmd.WR:
		bl := getBurstLength(mrs)

		switch bl {
		case 8:
			return []model.SubRow{
				{Macro: "WROL", Cmd: cmd},
				{Macro: "WROH"},
			}

		default:
			return []model.SubRow{
				{Macro: "WRL", Cmd: cmd},
				{Macro: "WRH"},
			}
		}

	case nativeCmd.WRA:
		bl := getBurstLength(mrs)

		switch bl {
		case 8:
			return []model.SubRow{
				{Macro: "WR8APL", Cmd: cmd},
				{Macro: "WR8APH"},
			}

		default:
			return []model.SubRow{
				{Macro: "WRAPL", Cmd: cmd},
				{Macro: "WRAPH"},
			}
		}

	case nativeCmd.RD:
		bl := getBurstLength(mrs)

		switch bl {
		case 8:
			return []model.SubRow{
				{Macro: "RD8APL", Cmd: cmd},
				{Macro: "RD8APH"},
			}
		default:
			return []model.SubRow{
				{Macro: "RDL", Cmd: cmd},
				{Macro: "RDH"},
			}
		}

	case nativeCmd.RDA:
		bl := getBurstLength(mrs)

		switch bl {
		case 8:
			return []model.SubRow{
				{Macro: "RDOL", Cmd: cmd},
				{Macro: "RDOH"},
			}
		default:
			return []model.SubRow{
				{Macro: "RDAPL", Cmd: cmd},
				{Macro: "RDAPH"},
			}
		}

	case nativeCmd.MRW:
		return []model.SubRow{
			{Macro: "MRWL", Cmd: cmd},
			{Macro: "MRWH"},
		}

	case nativeCmd.MRWX:
		return []model.SubRow{
			{Macro: "MRWL", Cmd: cmd},
			{Macro: "MRWH\tC1"},
		}

	case nativeCmd.MRR:
		return []model.SubRow{
			{Macro: "MRRL", Cmd: cmd},
			{Macro: "MRRH"},
		}

	case nativeCmd.MRRX:
		return []model.SubRow{
			{Macro: "MRRL", Cmd: cmd},
			{Macro: "MRRH\tC1"},
		}

	case nativeCmd.PREab:
		return []model.SubRow{
			{Macro: "PREAB"},
		}

	case nativeCmd.PREpb:
		return []model.SubRow{
			{Macro: "PREPB", Cmd: cmd},
		}

	case nativeCmd.PREsb:
		return []model.SubRow{
			{Macro: "PRESB", Cmd: cmd},
		}

	case nativeCmd.MPC:
		return []model.SubRow{
			{Macro: "OP_", Cmd: cmd},
			{Macro: "OP_"},
			{Macro: "OP_"},
			{Macro: "MPC"},
			{Macro: "MPC"},
			{Macro: "MPC"},
			{Macro: "MPC"},
			{Macro: "OP_"},
			{Macro: "OP_"},
			{Macro: "OP_"},
		}

	case nativeCmd.VrefCA:
		return []model.SubRow{
			{Macro: "OP_", Cmd: cmd},
			{Macro: "OP_"},
			{Macro: "OP_"},
			{Macro: "VREFCA"},
			{Macro: "VREFCA"},
			{Macro: "VREFCA"},
			{Macro: "VREFCA"},
			{Macro: "OP_"},
			{Macro: "OP_"},
			{Macro: "OP_"},
		}

	case nativeCmd.VrefCS:
		return []model.SubRow{
			{Macro: "OP_", Cmd: cmd},
			{Macro: "OP_"},
			{Macro: "OP_"},
			{Macro: "VREFCS"},
			{Macro: "VREFCS"},
			{Macro: "VREFCS"},
			{Macro: "VREFCS"},
			{Macro: "OP_"},
			{Macro: "OP_"},
			{Macro: "OP_"},
		}

	case nativeCmd.PDX:
		return []model.SubRow{
			{Macro: "PDEX"},
		}

	case nativeCmd.PDE:
		return []model.SubRow{
			{Macro: "PDE"},
		}

	case nativeCmd.WRP, nativeCmd.WRPA:
		return []model.SubRow{
			{Macro: "WRPL", Cmd: cmd},
			{Macro: "WRPH"},
		}

	case nativeCmd.WRPX, nativeCmd.WRPAX:
		return []model.SubRow{
			{Macro: "WRPL", Cmd: cmd},
			{Macro: "WRPH\tC1"},
		}

	case nativeCmd.RESET:
		return []model.SubRow{
			{Macro: "REST"},
		}

	case nativeCmd.REFab:
		return []model.SubRow{
			{Macro: "REFAB"},
		}

	case nativeCmd.REFsb:
		return []model.SubRow{
			{Macro: "REFSB"},
		}

	case nativeCmd.RFMab:
		return []model.SubRow{
			{Macro: "RFMAB"},
		}

	case nativeCmd.RFMsb:
		return []model.SubRow{
			{Macro: "RFMSB"},
		}

	default:
		log.Errorf("UNKNOWN COMMAND: %v", cmd.Type)
		return []model.SubRow{
			{Macro: "UNKNOWN COMMAND"},
		}
	}
}
