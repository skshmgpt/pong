package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
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

func DrawPoint(stdscr *gc.Window, pt int, y, x int) {
	stdscr.MovePrint(y, x, pt)
}

var (
	TIMEOUT      = 10
	PADDLEHEIGHT = 10
	PORT         = 6969
)

type Game struct {
	mu sync.RWMutex

	// paddle left
	p_l Paddle
	// paddle right
	p_r Paddle
	// Ball
	b Ball
	// points
	p1, p2 int
	// game speed control
	frameCount int
	ballSpeed  int
	// direction tracking
	leftDir  int
	rightDir int
}

type GameState struct {
	b      Ball
	p_l    Paddle
	p_r    Paddle
	p1, p2 int
}

type netPaddle struct {
	Y int `json:"y"`
	X int `json:"x"`
	H int `json:"h"`
}

type netBall struct {
	Y  int `json:"y"`
	X  int `json:"x"`
	DY int `json:"dy"`
	DX int `json:"dx"`
}

type netState struct {
	B  netBall   `json:"b"`
	PL netPaddle `json:"p_l"`
	PR netPaddle `json:"p_r"`
	P1 int       `json:"p1"`
	P2 int       `json:"p2"`
}

func NewGame(maxY, maxX int) *Game {
	game := &Game{
		p_l: Paddle{
			y: maxY/2 - PADDLEHEIGHT/2,
			x: 2,
			h: PADDLEHEIGHT,
		},
		p_r: Paddle{
			y: maxY/2 - PADDLEHEIGHT/2,
			x: maxX - 3,
			h: PADDLEHEIGHT,
		},
		b: Ball{
			y:  maxY / 2,
			x:  maxX / 2,
			dy: 1,
			dx: 1,
		},
		p1:         0,
		p2:         0,
		frameCount: 0,
		ballSpeed:  3,
		leftDir:    0,
		rightDir:   0,
	}

	return game
}

func (gs *Game) ResetDir() {
	gs.leftDir = 0
	gs.rightDir = 0
}

func handleConn(conn net.Conn, gs *Game, isHost bool) {
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	for {
		gs.mu.RLock()
		state := netState{
			B:  netBall{Y: gs.b.y, X: gs.b.x, DY: gs.b.dy, DX: gs.b.dx},
			PL: netPaddle{Y: gs.p_l.y, X: gs.p_l.x, H: gs.p_l.h},
			PR: netPaddle{Y: gs.p_r.y, X: gs.p_r.x, H: gs.p_r.h},
			P1: gs.p1,
			P2: gs.p2,
		}
		gs.mu.RUnlock()

		if err := encoder.Encode(state); err != nil {
			return
		}

		var remote netState
		if err := decoder.Decode(&remote); err != nil {
			return
		}

		gs.mu.Lock()
		if !isHost {
			// Client receives full game state from host
			gs.b.y, gs.b.x, gs.b.dy, gs.b.dx = remote.B.Y, remote.B.X, remote.B.DY, remote.B.X
			gs.p_l.y, gs.p_l.x, gs.p_l.h = remote.PL.Y, remote.PL.X, remote.PL.H
			gs.p1, gs.p2 = remote.P1, remote.P2
		} else {
			// Host only receives right paddle from client
			gs.p_r.y, gs.p_r.x, gs.p_r.h = remote.PR.Y, remote.PR.X, remote.PR.H
		}
		gs.mu.Unlock()

		time.Sleep(time.Duration(TIMEOUT) * time.Millisecond)
	}
}

func main() {
	stdscr, err := gc.Init()
	if err != nil {
		panic(err)
	}
	defer gc.End()

	gc.Cursor(0) // disable cursor
	stdscr.Keypad(true)
	stdscr.Timeout(1)
	gc.Echo(false)
	gc.UseDefaultColors()

	maxY, maxX := stdscr.MaxYX()
	gs := NewGame(maxY, maxX)

	// p2p setup
	host := flag.Bool("host", false, "Run as host")
	client := flag.String("client", "", "Run as client with given id")
	flag.Parse()

	gameReady := make(chan bool, 1)

	// Start networking in background
	if *host {
		go func() {
			port := fmt.Sprintf(":%d", PORT)
			ln, err := net.Listen("tcp4", port)
			if err != nil {
				fmt.Printf("Failed to start server: %v\n", err)
				os.Exit(1)
			}
			defer ln.Close()

			ip := ln.Addr().String()
			// encoded_ip := utils.Encode(ip, PORT)
			encoded_ip := ip

			// Show game ID in a separate goroutine that can be interrupted
			uiStop := make(chan struct{})
			go func() {
				t := time.NewTicker(500 * time.Millisecond)
				defer t.Stop()
				for {
					select {
					case <-uiStop:
						return
					case <-t.C:
						stdscr.Clear()
						stdscr.Box(0, 0)
						stdscr.MovePrint(maxY/2-1, maxX/2-len(encoded_ip)/2, "Game ID: "+encoded_ip)
						stdscr.MovePrint(maxY/2, maxX/2-15, "Waiting for player to connect...")
						stdscr.MovePrint(maxY/2+2, maxX/2-10, "Press 'q' to quit")
						stdscr.Refresh()
						time.Sleep(500 * time.Millisecond)
					}
				}
			}()

			conn, err := ln.Accept()
			if err != nil {
				return
			}
			close(uiStop)
			// Connection established
			go handleConn(conn, gs, true)
			gameReady <- true
		}()
	} else if *client != "" {
		go func() {
			// addr := utils.Decode(*client)
			addr := *client

			uiStop := make(chan struct{})
			// Show connecting message
			go func() {
				t := time.NewTicker(500 * time.Millisecond)
				defer t.Stop()
				for {
					select {
					case <-uiStop:
						return
					case <-t.C:
						stdscr.Clear()
						stdscr.Box(0, 0)
						stdscr.MovePrint(maxY/2, maxX/2-8, "Connecting...")
						stdscr.Refresh()
						time.Sleep(500 * time.Millisecond)
					}
				}
			}()

			conn, err := net.Dial("tcp", addr)
			if err != nil {
				close(uiStop)
				stdscr.Clear()
				stdscr.Box(0, 0)
				stdscr.MovePrint(maxY/2, maxX/2-10, "Connection failed!")
				stdscr.MovePrint(maxY/2+1, maxX/2-15, fmt.Sprintf("Error: %v", err))
				stdscr.MovePrint(maxY/2+3, maxX/2-8, "Press any key...")
				stdscr.Refresh()
				stdscr.GetChar()
				os.Exit(1)
			}
			close(uiStop)
			go handleConn(conn, gs, false)
			gameReady <- true
		}()
	} else {
		// Single player mode
		gameReady <- true
	}

	// Wait for game to be ready
	<-gameReady

	// Main game loop
	for {
		gs.frameCount++
		stdscr.Clear()
		stdscr.Box(0, 0)

		// Draw game elements
		gs.mu.RLock()
		gs.p_l.DrawPaddle(stdscr)
		gs.p_r.DrawPaddle(stdscr)
		gs.b.DrawBall(stdscr)
		p1, p2 := gs.p1, gs.p2
		gs.mu.RUnlock()

		stdscr.VLine(1, maxX/2, gc.ACS_VLINE, maxY-2)
		DrawPoint(stdscr, p1, maxY/7, maxX/4)
		DrawPoint(stdscr, p2, maxY/7, maxX/2+maxX/4)

		stdscr.Refresh()

		// Handle input
		ch := stdscr.GetChar()
		gs.ResetDir()
		switch ch {
		case 'q':
			return
		case 'w':
			gs.mu.Lock()
			gs.leftDir = -1
			gs.mu.Unlock()
		case 's':
			gs.mu.Lock()
			gs.leftDir = 1
			gs.mu.Unlock()
		case gc.KEY_UP:
			gs.mu.Lock()
			gs.rightDir = -1
			gs.mu.Unlock()
		case gc.KEY_DOWN:
			gs.mu.Lock()
			gs.rightDir = 1
			gs.mu.Unlock()
		}

		// Move paddles
		paddleSpeed := 2
		gs.mu.Lock()

		if gs.leftDir == -1 && gs.p_l.y > 1 {
			if gs.p_l.y == 2 {
				gs.p_l.y -= 1
			} else {
				gs.p_l.y -= paddleSpeed
			}
		}
		if gs.leftDir == 1 && gs.p_l.y+gs.p_l.h < maxY-1 {
			if gs.p_l.y+gs.p_l.h == maxY-2 {
				gs.p_l.y += 1
			} else {
				gs.p_l.y += paddleSpeed
			}
		}
		if gs.rightDir == -1 && gs.p_r.y > 1 {
			if gs.p_r.y == 2 {
				gs.p_r.y -= 1
			} else {
				gs.p_r.y -= paddleSpeed
			}
		}
		if gs.rightDir == 1 && gs.p_r.y+gs.p_r.h < maxY-1 {
			if gs.p_r.y+gs.p_r.h == maxY-2 {
				gs.p_r.y += 1
			} else {
				gs.p_r.y += paddleSpeed
			}
		}

		// Ball physics (only for single player or host)
		if *client == "" || *host {
			if gs.frameCount%gs.ballSpeed == 0 {
				gs.b.y += gs.b.dy
				gs.b.x += gs.b.dx
			}

			// Wall bouncing
			if gs.b.y <= 1 || gs.b.y >= maxY-2 {
				gs.b.dy *= -1
			}

			// Paddle collisions
			if gs.b.dx < 0 && gs.b.x <= gs.p_l.x+1 && gs.b.x >= gs.p_l.x &&
				gs.b.y >= gs.p_l.y && gs.b.y < gs.p_l.y+gs.p_l.h {
				gs.b.dx *= -1
			}

			if gs.b.dx > 0 && gs.b.x >= gs.p_r.x-1 && gs.b.x <= gs.p_r.x &&
				gs.b.y >= gs.p_r.y && gs.b.y < gs.p_r.y+gs.p_r.h {
				gs.b.dx *= -1
			}

			// Scoring
			if gs.b.x <= 0 {
				gs.p2++
				gs.b = Ball{maxY / 2, maxX / 2, 1, 1}
			}
			if gs.b.x >= maxX-1 {
				gs.p1++
				gs.b = Ball{maxY / 2, maxX / 2, 1, -1}
			}
		}

		gs.mu.Unlock()

		time.Sleep(time.Duration(TIMEOUT) * time.Millisecond)
	}
}
