package ddr5

type MR0 struct{ v uint8 }

func (m *MR0) Value() uint8 { return m.v }
func (m *MR0) Set(v uint8)  { m.v = v }

// MR0 (MA[7:0]=00H) Burst Length and CAS Latency
// MR0 代表 DDR5 Mode Register 0 的值，永远只有 8 位

// func (mr MR0) String() string {
// 	return fmt.Sprintf("BL=%d | CL=%d", mr.BurstLength(), mr.CASLatency())
// }

// Bit 7:2  : CAS Latency
// Bit 1:0  : Burst Length

func (mr *MR0) BurstLength() uint8 {
	bl := mr.v & 0x03
	switch bl {
	case 0b00:
		return 16
	case 0b01:
		return 8
	case 0b10:
		return 32
	case 0b11:
		return 32
	}
	return 0
}

// CAS Latency
func (mr MR0) CASLatency() uint16 {
	clCode := mr.v & 0b11111100
	switch clCode {
	case 0b000000:
		return 22
	case 0b000001:
		return 24
	case 0b000010:
		return 28
	case 0b000011:
		return 32
	case 0b111100:
		return 60
	case 0b101:
		return 40
	case 0b110:
		return 42 // 常见
	case 0b111:
		return 48 // 常见于 DDR5-8800+ 常见
	}
	return 0
}
