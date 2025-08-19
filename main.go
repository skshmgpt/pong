package main

import (
	"time"

	gc "github.com/rthornton128/goncurses"
)

type Paddle struct {
	y, x int
	h    int
}

type Ball struct {
	y, x   int
	dy, dx int // direction vector
}

func (p *Paddle) DrawPaddle(stdscr *gc.Window) {
	for i := 0; i < p.h; i++ {
		stdscr.MoveAddChar(p.y+i, p.x, gc.ACS_CKBOARD)
	}
}

func (b *Ball) DrawBall(stdscr *gc.Window) {
	stdscr.MovePrint(b.y, b.x, "@")
}

var (
	TIMEOUT      = 16
	PADDLEHEIGHT = 10
)

func main() {
	stdscr, _ := gc.Init()
	defer gc.End()

	gc.Cursor(0) // disable cursor
	stdscr.Keypad(true)
	stdscr.Timeout(5)
	gc.Echo(false)
	gc.UseDefaultColors()
	// screen limits

	stdscr.Box(0, 0)
	maxY, maxX := stdscr.MaxYX()
	leftPaddle := Paddle{maxY/2 - PADDLEHEIGHT/2, 2, PADDLEHEIGHT}
	rightPaddle := Paddle{maxY/2 - PADDLEHEIGHT/2, maxX - 3, PADDLEHEIGHT}
	ball := Ball{maxY / 2, maxX / 2, 1, 1}

	// game speed control
	frameCount := 0
	ballSpeed := 3

	// direction tracking
	leftDir := 0
	rightDir := 0

	// playable area bw (1, 1) and (maxX-1, maxY-1)

	for {
		frameCount++
		stdscr.Clear()
		stdscr.Box(0, 0)

		leftPaddle.DrawPaddle(stdscr)
		rightPaddle.DrawPaddle(stdscr)

		ball.DrawBall(stdscr)

		stdscr.VLine(1, maxX/2, gc.ACS_VLINE, maxY-2)

		stdscr.Refresh()

		ch := stdscr.GetChar()
		leftDir = 0
		rightDir = 0
		switch ch {
		case 'q':
			return
		case 'w':
			leftDir = -1 // move up
		case 's':
			leftDir = 1 // move down
		case gc.KEY_UP:
			rightDir = -1
		case gc.KEY_DOWN:
			rightDir = 1
		}

		if leftDir == -1 && leftPaddle.y > 1 {
			leftPaddle.y -= 1
		}
		if leftDir == 1 && leftPaddle.y+leftPaddle.h < maxY-1 {
			leftPaddle.y += 1
		}
		if rightDir == -1 && rightPaddle.y > 1 {
			rightPaddle.y -= 1
		}
		if rightDir == 1 && rightPaddle.y+rightPaddle.h < maxY-1 {
			rightPaddle.y += 1
		}

		if frameCount%ballSpeed == 0 {
			ball.y += ball.dy
			ball.x += ball.dx
		}

		if ball.y <= 1 || ball.y >= maxY-2 {
			ball.dy *= -1
		}

		if ball.dx < 0 && ball.x <= leftPaddle.x+1 && ball.x >= leftPaddle.x &&
			ball.y >= leftPaddle.y && ball.y < leftPaddle.y+leftPaddle.h {
			ball.dx *= -1
		}

		if ball.dx > 0 && ball.x >= rightPaddle.x-1 && ball.x <= rightPaddle.x &&
			ball.y >= rightPaddle.y && ball.y < rightPaddle.y+rightPaddle.h {
			ball.dx *= -1
		}

		if ball.x <= 0 {
			ball = Ball{maxY / 2, maxX / 2, 1, 1} // always start moving right
		}
		if ball.x >= maxX-1 {
			ball = Ball{maxY / 2, maxX / 2, 1, -1} // always start moving left
		}
		time.Sleep(time.Duration(TIMEOUT) * time.Millisecond)
	}

}
