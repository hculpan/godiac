package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hculpan/godiac/cardiac"
)

var defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
var execStyle = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite)
var statusStyle = tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack)

var statusMessage string = ""
var statusMessageTime time.Time

const statusMessageDuration int64 = 3000

const (
	NORMAL_PROCESSING = iota
	DUMP_TO_FILE
	READ_FROM_FILE
)

var SimulatorState int = NORMAL_PROCESSING

func main() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	var timing int64 = 1000

	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	cpu := cardiac.NewCardiac()
	lastExec := time.Now()

	s.Sync()
	drawUpdate(s, cpu)

	renderNotificationChannel := make(chan bool)

	go func() {
		for {
			switch event := s.PollEvent().(type) {
			case *tcell.EventKey:
				if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyCtrlC {
					s.Fini()
					os.Exit(0)
				} else if event.Key() == tcell.KeyF6 {
					dumpMemory(s, cpu)
				} else if event.Key() == tcell.KeyF7 {
					restoreMemory(s, cpu)
				} else if event.Rune() == 'R' || event.Rune() == 'r' {
					if cpu.State == cardiac.CPU_HALTED {
						timing = 1000
						cpu.Reset()
						renderNotificationChannel <- true
					} else if cpu.State == cardiac.CPU_PAUSED {
						timing = 1000
						cpu.Unpause()
						renderNotificationChannel <- true
					}
				} else if event.Rune() == 'H' || event.Rune() == 'h' {
					if cpu.State == cardiac.CPU_HALTED {
						timing = 1000
						cpu.HardReset()
						renderNotificationChannel <- true
					}
				} else if event.Rune() == 'P' || event.Rune() == 'p' {
					if cpu.State == cardiac.CPU_RUNNING {
						cpu.Pause()
						renderNotificationChannel <- true
					}
				} else if event.Rune() == 'S' || event.Rune() == 's' {
					if cpu.State == cardiac.CPU_PAUSED || cpu.State == cardiac.CPU_HALTED {
						cpu.State = cardiac.CPU_STEP
						timing = 250
					}
				} else if event.Rune() >= '0' && event.Rune() <= '9' && cpu.State == cardiac.CPU_INPUT {
					cpu.Input += string(event.Rune())
					if len(cpu.Input) > 4 {
						cpu.Input = cpu.Input[len(cpu.Input)-4:]
					}
				} else if (event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyDelete) && cpu.State == cardiac.CPU_INPUT {
					cpu.Input = " " + cpu.Input[0:3]
					if cpu.Input == "    " {
						cpu.Input = "   0"
					}
				} else if event.Key() == tcell.KeyEnter && cpu.State == cardiac.CPU_INPUT {
					timing = 1000
					cpu.EndInput()
				}
			case *tcell.EventResize:
				s.Sync()
				renderNotificationChannel <- true
			default:
				//Unsupported or irrelevant event
			}
		}
	}()

	go func() {
		for {
			<-renderNotificationChannel
			s.Clear()
			drawUpdate(s, cpu)
			s.Show()
		}
	}()

	for {
		if time.Since(lastExec).Milliseconds() >= timing {
			cpu.ExecuteCurrent()
			if cpu.State == cardiac.CPU_INPUT {
				timing = 10
			}
			lastExec = time.Now()
			renderNotificationChannel <- true
		}

	}

}

func dumpMemory(s tcell.Screen, cpu *cardiac.Cardiac) {
	statusMessage = "Dumping to file"
	statusMessageTime = time.Now()

	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	f, err := os.Create(path + "/memdump.godiac")
	if err != nil {
		statusMessage = "Error dumping file: " + err.Error()
	}

	for i := 1; i < 100; i++ {
		f.WriteString(fmt.Sprintf("%02d:%03d\n", i, cpu.Memory[i]))
	}

	defer f.Close()
}

func restoreMemory(s tcell.Screen, cpu *cardiac.Cardiac) {
	statusMessage = "Reading from file"
	statusMessageTime = time.Now()

	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	file, err := os.Open(path + "/memdump.godiac")
	if err != nil {
		statusMessage = "Error reading file: " + err.Error()
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Split(scanner.Text(), ":")
		m, err := strconv.Atoi(s[0])
		if err != nil {
			statusMessage = "Error parsing memory location '" + s[0] + "': " + err.Error()
			break
		}
		v, err := strconv.Atoi(s[1])
		if err != nil {
			statusMessage = "Error parsing memory value '" + s[1] + "': " + err.Error()
			break
		}

		cpu.Memory[m] = int16(v)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func drawUpdate(s tcell.Screen, cpu *cardiac.Cardiac) {
	width, _ := s.Size()
	drawText(s, (width/2)-8, 0, defStyle, "---  CARDIAC  ---")

	drawText(s, 10, 2, defStyle, "F6: Dump memory to file")
	drawText(s, 40, 2, defStyle, "F7: Restore memory from file")

	drawText(s, 5, 4, defStyle, "Accumulator")
	if cpu.Accumulator >= 0 {
		drawText(s, 7, 5, defStyle, fmt.Sprintf("[ %04d]", cpu.Accumulator))
	} else {
		drawText(s, 7, 5, defStyle, fmt.Sprintf("[-%04d]", cpu.Accumulator*-1))
	}

	drawText(s, 9, 7, defStyle, "IP")
	drawText(s, 8, 8, defStyle, fmt.Sprintf("[%03d]", cpu.Ip))

	drawText(s, 7, 11, defStyle, "Output")
	if cpu.Output < 0 {
		drawText(s, 7, 12, defStyle, fmt.Sprintf("[-%03d]", int(cpu.Output)*-1))
	} else if cpu.Output < 10000 {
		drawText(s, 7, 12, defStyle, fmt.Sprintf("[ %03d]", int(cpu.Output)))
	} else {
		drawText(s, 7, 12, defStyle, "[    ]")
	}

	drawText(s, 7, 14, defStyle, "Input")
	drawText(s, 7, 15, defStyle, fmt.Sprintf("[%s]", cpu.Input))

	for mem := 0; mem < 100; mem++ {
		r := mem - 75
		c := 3
		if mem < 25 {
			r = mem
			c = 0
		} else if mem < 50 {
			r = mem - 25
			c = 1
		} else if mem < 75 {
			r = mem - 50
			c = 2
		}

		style := defStyle
		if mem == int(cpu.Ip) {
			style = execStyle
		}

		if cpu.Memory[mem] < 0 {
			drawText(s, 20+(c*13), r+4, style, fmt.Sprintf("%03d [-%03d]", mem, cpu.Memory[mem]*-1))
		} else {
			drawText(s, 20+(c*13), r+4, style, fmt.Sprintf("%03d [ %03d]", mem, cpu.Memory[mem]))
		}

	}

	switch cpu.State {
	case cardiac.CPU_RUNNING:
		drawText(s, 1, 30, statusStyle, "Running")
		drawText(s, 15, 30, defStyle, "(P)ause")
	case cardiac.CPU_HALTED:
		drawText(s, 1, 30, statusStyle, "Halted")
		drawText(s, 15, 30, defStyle, "(R)eset")
		drawText(s, 30, 30, defStyle, "(H)ard reset")
		drawText(s, 45, 30, defStyle, "(S)tep")
	case cardiac.CPU_PAUSED:
		drawText(s, 1, 30, statusStyle, "Paused")
		drawText(s, 15, 30, defStyle, "(R)esume")
		drawText(s, 30, 30, defStyle, "(S)tep")
	case cardiac.CPU_INPUT:
		drawText(s, 1, 30, statusStyle, "Input")
		drawText(s, 15, 30, defStyle, "ENTER to end")
	}

	drawText(s, width-17, 30, defStyle, "ESC to terminate")

	if statusMessage != "" {
		drawText(s, (width/2)-(len(statusMessage)/2), 32, statusStyle, statusMessage)
		if time.Since(statusMessageTime).Milliseconds() > statusMessageDuration {
			statusMessage = ""
		}
	}
}

func drawText(s tcell.Screen, x int, y int, style tcell.Style, text string) {
	xoffset := 0
	for _, r := range []rune(text) {
		s.SetContent(x+xoffset, y, r, nil, style)
		xoffset++
	}
}

func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	// Fill background
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}

	//	drawText(s, x1+1, y1+1, x2-1, y2-1, style, text)
}
