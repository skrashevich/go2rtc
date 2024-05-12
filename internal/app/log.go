package app

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var MemoryLog *circularBuffer

func NewLogger(format string, level string) zerolog.Logger {
	var writer io.Writer = os.Stdout

	if format != "json" {
		writer = zerolog.ConsoleWriter{
			Out: writer, TimeFormat: "15:04:05.000", NoColor: format == "text",
		}
	}

	MemoryLog = newBuffer(16)

	writer = zerolog.MultiLevelWriter(writer, MemoryLog)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	lvl, err := zerolog.ParseLevel(level)
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(writer).With().Timestamp().Logger().Level(lvl)
}

func GetLogger(module string) zerolog.Logger {
	if s, ok := modules[module]; ok {
		lvl, err := zerolog.ParseLevel(s)
		if err == nil {
			return log.Level(lvl)
		}
		log.Warn().Err(err).Caller().Send()
	}

	return log.Logger
}

// modules log levels
var modules map[string]string

const chunkSize = 1 << 16

type circularBuffer struct {
	chunks [][]byte
	r, w   int
	mu     sync.Mutex
}

func newBuffer(chunks int) *circularBuffer {
	b := &circularBuffer{chunks: make([][]byte, 0, chunks)}
	// create first chunk
	b.chunks = append(b.chunks, make([]byte, 0, chunkSize))
	return b
}

func (b *circularBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n = len(p)
	if n == 0 {
		return 0, nil
	}

	if len(b.chunks) == 0 {
		b.chunks = append(b.chunks, make([]byte, 0, chunkSize))
	}

	// check if chunk has size
	if len(b.chunks[b.w])+n > chunkSize {
		// increase write chunk index
		b.w++
		if b.w == cap(b.chunks) {
			b.w = 0
		}
		// check overflow
		if b.r == b.w {
			return 0, errors.New("circularBuffer overflow, cannot write without overwriting unread data")
		}
		// check if current chunk exists
		if b.w == len(b.chunks) {
			// allocate new chunk
			b.chunks = append(b.chunks, make([]byte, 0, chunkSize))
		} else {
			// reset len of current chunk
			b.chunks[b.w] = b.chunks[b.w][:0]
		}
	}

	b.chunks[b.w] = append(b.chunks[b.w], p...)
	return n, nil
}

func (b *circularBuffer) WriteTo(w io.Writer) (n int64, err error) {
	for i := b.r; ; {
		var nn int
		if nn, err = w.Write(b.chunks[i]); err != nil {
			return
		}
		n += int64(nn)

		if i == b.w {
			break
		}
		if i++; i == cap(b.chunks) {
			i = 0
		}
	}
	return
}

func (b *circularBuffer) Reset() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.chunks {
		b.chunks[i] = b.chunks[i][:0]
	}

	b.r = 0
	b.w = 0

	return nil
}
