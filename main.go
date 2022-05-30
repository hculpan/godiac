package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hculpan/godiac/cardiac"
)

var defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
var execStyle = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite)

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
				} else if event.Rune() == 'R' || event.Rune() == 'r' {
					if cpu.State == cardiac.CPU_HALTED {
						cpu.Reset()
					}
				} else if event.Rune() == 'H' || event.Rune() == 'h' {
					if cpu.State == cardiac.CPU_HALTED {
						cpu.HardReset()
					}
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
			lastExec = time.Now()
			renderNotificationChannel <- true
		}

	}

}

func drawUpdate(s tcell.Screen, cpu *cardiac.Cardiac) {
	width, _ := s.Size()
	drawText(s, (width/2)-8, 0, defStyle, "---  CARDIAC  ---")
	drawText(s, 5, 2, defStyle, "Accumulator")
	if cpu.Accumulator >= 0 {
		drawText(s, 7, 3, defStyle, fmt.Sprintf("[ %04d]", cpu.Accumulator))
	} else {
		drawText(s, 7, 3, defStyle, fmt.Sprintf("[-%04d]", cpu.Accumulator*-1))
	}

	drawText(s, 9, 5, defStyle, "IP")
	drawText(s, 8, 6, defStyle, fmt.Sprintf("[%03d]", cpu.Ip))

	drawText(s, 7, 8, defStyle, "Output")
	if math.Abs(cpu.Output) < 10000 {
		drawText(s, 7, 9, defStyle, fmt.Sprintf("[%03d]", int(cpu.Output)))
	} else {
		drawText(s, 7, 9, defStyle, "[    ]")
	}

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
			drawText(s, 20+(c*13), r+2, style, fmt.Sprintf("%03d [-%03d]", mem, cpu.Memory[mem]*-1))
		} else {
			drawText(s, 20+(c*13), r+2, style, fmt.Sprintf("%03d [ %03d]", mem, cpu.Memory[mem]))
		}
	}

	switch cpu.State {
	case cardiac.CPU_RUNNING:
		drawText(s, 1, 30, defStyle, "Running")
	case cardiac.CPU_HALTED:
		drawText(s, 1, 30, defStyle, "Halted")
		drawText(s, 15, 30, defStyle, "(R)eset")
		drawText(s, 30, 30, defStyle, "(H)ard reset")
	case cardiac.CPU_PAUSED:
		drawText(s, 1, 30, defStyle, "PAUSED")
	}

	drawText(s, width-17, 30, defStyle, "ESC to terminate")
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
