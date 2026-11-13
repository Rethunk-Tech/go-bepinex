// Package logline reads and writes the lines of BepInEx's LogOutput.log: a header naming a level and a source, then
// the message, optionally behind the timestamp a logger mod adds. The lines come from mods, so they are untrusted text.
package logline

import (
	"fmt"
	"regexp"
	"strings"
)

// Level is a BepInEx log level, spelled as the header writes it.
type Level string

// BepInEx's levels, most severe first.
const (
	Fatal   Level = "Fatal"
	Error   Level = "Error"
	Warning Level = "Warning"
	Message Level = "Message"
	Info    Level = "Info"
	Debug   Level = "Debug"
)

// Line is one header line. A message's further lines carry no header and are not a Line.
type Line struct {
	// Stamp is the logger mod's "[stamp] " prefix without brackets, or "".
	Stamp string
	Level Level
	// Source is the log source, which for a plugin is its name, without the header's padding.
	Source  string
	Message string
}

// BepInEx writes `[Level  :    Source] message`, padding the level to seven characters and the source to ten.
var header = regexp.MustCompile(`^\[(Fatal|Error|Warning|Message|Info|Debug) *: *([^\]]*)\] ?(.*)$`)

// stamp is the "[stamp] " mattymatty-AsyncLoggers puts before a header (ExtendedLogEventArgs.cs, ToTimestampedEventArgs):
// HH:mm:ss.fffffff by default, or a 16-digit tick, frame or counter, or a 16-character thread name.
var stamp = regexp.MustCompile(`^\[([^\]]*)\] (\[(?:Fatal|Error|Warning|Message|Info|Debug) *:.*)$`)

// SplitStamp separates a logger mod's leading "[stamp] " from a BepInEx line; rest is what every BepInEx reader matches
// on. A line without a stamp comes back whole.
func SplitStamp(line string) (stampText, rest string) {
	if m := stamp.FindStringSubmatch(line); m != nil {
		return m[1], m[2]
	}
	return "", line
}

// Parse reads a header line, with or without a stamp. ok is false for a continuation line or any other text.
func Parse(line string) (l Line, ok bool) {
	st, rest := SplitStamp(line)
	m := header.FindStringSubmatch(rest)
	if m == nil {
		return Line{}, false
	}
	return Line{Stamp: st, Level: Level(m[1]), Source: strings.TrimSpace(m[2]), Message: strings.TrimSuffix(m[3], "\r")}, true
}

// Format writes l the way BepInEx does, so a reader built for LogOutput.log reads it back.
// A CR or LF in a field becomes a space, so untrusted text cannot start a second header line.
func Format(l Line) string {
	head := fmt.Sprintf("[%-7s:%10s] %s", oneLine(string(l.Level)), oneLine(l.Source), oneLine(l.Message))
	if l.Stamp != "" {
		return "[" + oneLine(l.Stamp) + "] " + head
	}
	return head
}

var lineBreaks = strings.NewReplacer("\r", " ", "\n", " ")

func oneLine(s string) string { return lineBreaks.Replace(s) }
