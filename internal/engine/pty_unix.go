//go:build !windows

package engine

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func runWithPTY(opts Options, matcher *Matcher) (int, error) {
	cmd := exec.Command(opts.Command[0], opts.Command[1:]...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 1, fmt.Errorf("starting PTY: %w", err)
	}
	defer ptmx.Close()

	// Handle terminal resize
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	sigCh <- syscall.SIGWINCH // Initial resize

	// Set stdin to raw mode if it's a terminal
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err == nil {
			defer term.Restore(fd, oldState)
		}
	}

	// Create stepper for post-login step execution
	stepper := NewStepper(opts.Steps, opts.Prompt, ptmx)

	var wg sync.WaitGroup

	// Forward user input to PTY
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(ptmx, opts.Stdin)
	}()

	// Read PTY output, match patterns, forward to stdout
	wg.Add(1)
	go func() {
		defer wg.Done()

		lineBuf := ""
		buf := make([]byte, 4096)
		timer := time.NewTimer(100 * time.Millisecond)
		timer.Stop()

		reader := bufio.NewReader(ptmx)

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				opts.Stdout.Write(buf[:n])

				lineBuf += chunk

				for {
					idx := findNewline(lineBuf)
					if idx < 0 {
						timer.Reset(100 * time.Millisecond)
						break
					}
					line := lineBuf[:idx+1]
					lineBuf = lineBuf[idx+1:]
					processLineUnix(matcher, stepper, line, ptmx)
				}
			}
			if err != nil {
				if lineBuf != "" {
					processLineUnix(matcher, stepper, lineBuf, ptmx)
				}
				break
			}

			select {
			case <-timer.C:
				if lineBuf != "" {
					processLineUnix(matcher, stepper, lineBuf, ptmx)
					lineBuf = ""
				}
			default:
			}
		}
	}()

	err = cmd.Wait()
	signal.Stop(sigCh)
	close(sigCh)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 1, fmt.Errorf("waiting for command: %w", err)
		}
	}

	return exitCode, nil
}

func processLineUnix(matcher *Matcher, stepper *Stepper, line string, ptmx *os.File) {
	result := matcher.Check(line)
	if result != nil {
		ptmx.Write([]byte(result.Response + "\n"))
		stepper.Activate()
		return
	}
	stepper.Check(line)
}

func findNewline(s string) int {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return i
		}
	}
	return -1
}
