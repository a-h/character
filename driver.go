package character

import (
	"time"

	"periph.io/x/periph/conn/i2c"
)

// Good list of LCD displays: http://www.cpmspectrepi.uk/raspberry_pi/MoinMoinExport/VariousDisplays.html#A16x2_Character_LCD_.28Yellow_Backlight.29

// First (LSB) 4 bits of i2c are RS, RW, EN, BL.
// Last 4 bits are D4, D5, D6, D7.
// See https://www.raspberrypi.org/forums/viewtopic.php?t=204036#p1266512.
const (
	I2CRSOn uint8 = 0b00000001
	I2CRWOn uint8 = 0b00000010
	I2CENOn uint8 = 0b00000100
	I2CBLOn uint8 = 0b00001000
)

// Instructions, see p24 of https://www.sparkfun.com/datasheets/LCD/HD44780.pdf
const InstructionClearDisplay uint8 = 0b00000001
const InstructionReturnHome uint8 = 0b00000010
const (
	InstructionEntryModeSet          uint8 = 0b00000100
	InstructionEntryModeSetIncrement uint8 = 0b00000010
)
const (
	InstructionDisplayOnOffControl           uint8 = 0b00001000
	InstructionDisplayOnOffControlDisplayOn  uint8 = 0b00000100
	InstructionDisplayOnOffControlCursorOn   uint8 = 0b00000010
	InstructionDisplayOnOffControlBlinkingOn uint8 = 0b00000001
)
const (
	InstructionCursorOrDisplayShift uint8 = 0b00010000
	// See table 7, line 29 of specification.
	InstructionCursorOrDisplayShiftCursorLeft   uint8 = 0b00000000
	InstructionCursorOrDisplayShiftCursorRight  uint8 = 0b00000100
	InstructionCursorOrDisplayShiftDisplayLeft  uint8 = 0b00001000
	InstructionCursorOrDisplayShiftDisplayRight uint8 = 0b00001100
)
const (
	InstructionFunctionSet        uint8 = 0b00100000
	InstructionFunctionSetDL8Bit  uint8 = 0b00010000
	InstructionFunctionSetNNLines uint8 = 0b00001000
	InstructionFunctionSetF5x10   uint8 = 0b00000100
)
const (
	InstructionSetDDRAMAddress uint8 = 0b10000000
)
const (
	InstructionSetCGRAMAddress uint8 = 0b01000000
)

// Display drives an LCD display.
type Display struct {
	Dev       *i2c.Dev
	backlight uint8
}

// NewDisplay creates a new display.
func NewDisplay(dev *i2c.Dev, singleLine bool) *Display {
	d := &Display{
		Dev:       dev,
		backlight: I2CBLOn,
	}

	// Initialize 4 bit mode as per Figure 24 of the Hitachi specification.
	time.Sleep(40 * time.Millisecond)
	d.WriteInstruction(0b00000011)
	time.Sleep(5 * time.Millisecond)
	d.WriteInstruction(0b00000011)
	time.Sleep(100 * time.Millisecond)
	d.WriteInstruction(0b00000011)

	// BF can be checked after the following instruction.
	d.WriteInstruction(0b00000010)

	// Complete initialization.
	var n uint8
	if !singleLine {
		n = InstructionFunctionSetNNLines
	}
	d.WriteInstruction(InstructionFunctionSet | n)
	// In figure 24, it says to have the display off, but that doesn't make sense to me.
	d.WriteInstruction(InstructionDisplayOnOffControl | InstructionDisplayOnOffControlDisplayOn)
	d.WriteInstruction(InstructionClearDisplay)
	// Wait an extra millisecond because clearing the display can take 1.53ms.
	time.Sleep(time.Millisecond + (time.Microsecond * 530))
	d.WriteInstruction(InstructionEntryModeSet | InstructionEntryModeSetIncrement)

	return d
}

func (d *Display) write(data uint8) {
	// Write with enabled flag.
	d.Dev.Write([]byte{data | I2CENOn | d.backlight})
	time.Sleep(time.Microsecond * 500)
	// Switch off.
	d.Dev.Write([]byte{(data &^ I2CENOn) | d.backlight})
	time.Sleep(time.Microsecond * 500)
}

const lowerMask = 0b11110000

// Write instruction.
func (d *Display) WriteInstruction(cmd uint8) {
	// Clear the lower bits.
	d.write(cmd & lowerMask)
	// Move the cmd into the high bits and clear the lower bits.
	d.write((cmd << 4) & lowerMask)
}

// Write data.
func (d *Display) WriteData(cmd uint8) {
	// Clear the lower bits, then set the data register.
	d.write((cmd & lowerMask) | I2CRSOn)
	// Move the cmd into the high bits and clear the lower bits, then
	// set the data register.
	d.write(((cmd << 4) & lowerMask) | I2CRSOn)
}

// Print text.
func (d *Display) Print(s string) {
	for _, c := range s {
		//TODO: Check what happens with Japanese characters.
		d.WriteData(uint8(c))
	}
}

// Goto position.
func (d *Display) Goto(row, col uint8) {
	// Add column offsets.
	// See: https://www.microchip.com/forums/tm.aspx?m=1124585&mpage=1
	switch row {
	case 1:
		col += 0b01000000
	}
	d.WriteInstruction(InstructionSetDDRAMAddress | col)
}

// Clear display.
func (d *Display) Clear() {
	d.WriteInstruction(InstructionClearDisplay)
	d.WriteInstruction(InstructionReturnHome)
}

// SetBacklight on or off.
func (d *Display) SetBacklight(on bool) {
	if on {
		d.backlight = I2CBLOn
		return
	}
	d.backlight = 0
}

// DisplayShiftLeft moves the display to the left.
func (d *Display) DisplayShiftLeft() {
	d.WriteInstruction(InstructionCursorOrDisplayShift | InstructionCursorOrDisplayShiftDisplayLeft)
}

// DisplayShiftRight moves the display to the right.
func (d *Display) DisplayShiftRight() {
	d.WriteInstruction(InstructionCursorOrDisplayShift | InstructionCursorOrDisplayShiftDisplayRight)
}

type CustomChars [8][8]byte

func (d *Display) LoadCustomChars(chars CustomChars) {
	d.WriteInstruction(InstructionSetCGRAMAddress)
	for _, c := range chars {
		for _, b := range c {
			d.WriteData(b)
		}
	}
}
