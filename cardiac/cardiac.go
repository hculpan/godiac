package cardiac

import (
	"log"
	"math"
)

const (
	CPU_RUNNING = iota
	CPU_HALTED
	CPU_PAUSED
	CPU_STEP
)

type Cardiac struct {
	Accumulator int16
	Ip          byte
	Memory      [100]int16
	State       int
	Output      float64
}

func NewCardiac() *Cardiac {
	result := &Cardiac{
		Accumulator: 0,
		Ip:          0,
	}

	result.HardReset()

	return result
}

func (c *Cardiac) Reset() {
	c.Ip = 0
	c.State = CPU_RUNNING
}

func (c *Cardiac) HardReset() {
	c.Accumulator = 0
	c.Ip = 0
	c.State = CPU_RUNNING
	c.Output = 10000

	for i := 0; i < 100; i++ {
		c.Memory[i] = 0
	}

	c.Memory[0] = 197
	c.Memory[1] = 298
	c.Memory[2] = 420
	c.Memory[3] = 695
	c.Memory[4] = 595
	c.Memory[5] = 900
	c.Memory[97] = -40
	c.Memory[98] = 24
	c.Memory[99] = 12
}

func (c *Cardiac) Pause() {
	if c.State == CPU_RUNNING {
		c.State = CPU_PAUSED
	}
}

func (c *Cardiac) Unpause() {
	if c.State == CPU_PAUSED {
		c.State = CPU_RUNNING
	}
}

func (c *Cardiac) ExecuteCurrent() {
	if c.State != CPU_RUNNING && c.State != CPU_STEP {
		return
	}

	log.Printf("Accumulator = %d, IP = %d", c.Accumulator, c.Ip)
	if c.Memory[c.Ip] > -1 {

		log.Printf("instruction = %d", c.Memory[c.Ip])
		instr := int16(c.Memory[c.Ip] / 100)
		log.Printf("instr = %d", instr)
		addr := int16(c.Memory[c.Ip] - (instr * 100))
		log.Printf("addr = %d", addr)

		switch instr {
		case 0:
		case 1: // CLA
			c.Accumulator = c.Memory[addr]
			c.Ip++
		case 2: // ADD
			c.Accumulator += c.Memory[addr]
			c.Ip++
		case 3: // TAC
			if c.Accumulator < 0 {
				c.Ip = byte(addr)
			} else {
				c.Ip++
			}
		case 4: // SFT
			left := addr / 10
			right := addr % 10
			log.Printf("left = %d, right = %d", left, right)
			c.Accumulator *= int16(math.Pow(10, float64(left)))
			c.Accumulator /= int16(math.Pow(10, float64(right)))
			for math.Abs(float64(c.Accumulator)) > 9999 {
				c.Accumulator %= 10000
			}
			c.Ip++
		case 5: // OUT
			c.Output = float64(c.Memory[addr])
			c.Ip++
		case 6: // STO
			c.Memory[addr] = c.Accumulator % 1000
			c.Ip++
		case 7: // SUB
			c.Accumulator -= c.Memory[addr]
			c.Ip++
		case 8: // JMP
			newAddr := int16(c.Ip) + 801
			c.Memory[99] = newAddr
			c.Ip = byte(addr)
		case 9: // HLT
			c.Reset()
			c.State = CPU_HALTED
		}
	} else {
		c.Ip++
	}

	if c.State == CPU_STEP {
		c.State = CPU_PAUSED
	}
}
