package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

type Command struct {
	Type string
	Val  int
}

type NoteEvent struct {
	Tick  int64
	Type  string
	Note  uint8
	Freq  int
	Track int
}

type TempoChange struct {
	Tick int64
	BPM  float64
}

var gmInstruments = map[uint8]string{
	0: "Acoustic Grand Piano", 1: "Bright Acoustic Piano", 4: "Electric Piano 1", 5: "Electric Piano 2",
	6: "Harpsichord", 16: "Hammond Organ", 19: "Church Organ", 24: "Acoustic Guitar (nylon)",
	25: "Acoustic Guitar (steel)", 26: "Electric Guitar (jazz)", 27: "Electric Guitar (clean)",
	30: "Overdriven Guitar", 31: "Distortion Guitar", 32: "Acoustic Bass", 33: "Electric Bass (finger)",
	34: "Electric Bass (pick)", 40: "Violin", 42: "Cello", 48: "String Ensemble 1", 52: "Choir Aahs",
	56: "Trumpet", 57: "Trombone", 60: "French Horn", 61: "Brass Section", 65: "Alto Sax",
	66: "Tenor Sax", 71: "Clarinet", 73: "Flute", 80: "Lead 1 (square)", 81: "Lead 2 (sawtooth)",
}

func midiNoteToFreq(note uint8) int {
	return int(math.Round(440.0 * math.Pow(2.0, float64(int(note)-69)/12.0)))
}

func inspectTrack(track smf.Track) (string, string) {
	trackName := ""
	instrumentName := ""

	for _, msg := range track {
		var text string
		if msg.Message.GetMetaText(&text) {
			metaType := msg.Message.Bytes()[1]
			if metaType == 0x03 && trackName == "" {
				trackName = text
			} else if metaType == 0x04 && instrumentName == "" {
				instrumentName = text
			}
		}

		var channel, program uint8
		if msg.Message.GetProgramChange(&channel, &program) {
			if name, ok := gmInstruments[program]; ok && instrumentName == "" {
				instrumentName = name
			}
		}
	}
	return trackName, instrumentName
}

func main() {
	fmt.Println(` ▄▄▄▄  ▓█████▓█████ ██▓███   ██████ ▄████▄  ▄▄▄      ██▓███ ▓█████
▓█████▄▓█   ▀▓█   ▀▓██░  ██▒██    ▒▒██▀ ▀█ ▒████▄   ▓██░  ██▓█   ▀
▒██▒ ▄█▒███  ▒███  ▓██░ ██▓░ ▓██▄  ▒▓█    ▄▒██  ▀█▄ ▓██░ ██▓▒███
▒██░█▀ ▒▓█  ▄▒▓█  ▄▒██▄█▓▒ ▒ ▒   ██▒▓▓▄ ▄██░██▄▄▄▄██▒██▄█▓▒ ▒▓█  ▄
░▓█  ▀█░▒████░▒████▒██▒ ░  ▒██████▒▒ ▓███▀ ░▓█   ▓██▒██▒ ░  ░▒████▒
░▒▓███▀░░ ▒░ ░░ ▒░ ▒▓▒░ ░  ▒ ▒▓▒ ▒ ░ ░▒ ▒  ░▒▒   ▓▒█▒▓▒░ ░  ░░ ▒░ ░
▒░▒   ░ ░ ░  ░░ ░  ░▒ ░    ░ ░▒  ░ ░ ░  ▒    ▒   ▒▒ ░▒ ░     ░ ░  ░
 ░    ░   ░     ░  ░░      ░  ░  ░ ░         ░   ▒  ░░         ░
 ░        ░  ░  ░  ░             ░ ░ ░           ░  ░          ░  ░
      ░                                ░                                `)

	fmt.Println("=== beepscape m2b converter ===")
	fmt.Println("===     made by sabisa      ===")
	var midiPath string
	if len(os.Args) < 2 {
		fmt.Print("enter path to midi file: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			midiPath = scanner.Text()
		}
	} else {
		midiPath = os.Args[1]
	}

	midiPath = strings.TrimSpace(midiPath)
	midiPath = strings.Trim(midiPath, "\"")

	file, err := smf.ReadFile(midiPath)
	if err != nil {
		fmt.Printf("error loading file: %v\n", err)
		return
	}

	fmt.Printf("\n--- tracks found in %s ---\n", midiPath)
	for i, track := range file.Tracks {
		tName, iName := inspectTrack(track)
		if tName == "" {
			tName = fmt.Sprintf("track %d", i)
		}
		if iName == "" {
			iName = "unknown instrument"
		}
		fmt.Printf("[%d] %s | instrument: %s (%d messages)\n", i, tName, iName, len(track))
	}

	fmt.Print("\nselect tracks (e.g. 1,3,5) or type 'all': ")
	scanner := bufio.NewScanner(os.Stdin)
	var input string
	if scanner.Scan() {
		input = scanner.Text()
	}
	input = strings.TrimSpace(strings.ToLower(input))

	var selectedTracks []int
	if input == "all" {
		for i := range file.Tracks {
			selectedTracks = append(selectedTracks, i)
		}
	} else {
		parts := strings.Split(input, ",")
		for _, p := range parts {
			val, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil || val < 0 || val >= len(file.Tracks) {
				fmt.Printf("skipping invalid track: %s\n", p)
				continue
			}
			selectedTracks = append(selectedTracks, val)
		}
	}

	if len(selectedTracks) == 0 {
		fmt.Println("no valid tracks selected.")
		return
	}

	fmt.Print("enable short beep filtering? (y/n): ")
	var filterInput string
	if scanner.Scan() {
		filterInput = strings.TrimSpace(strings.ToLower(scanner.Text()))
	}
	filterShort := filterInput == "y" || filterInput == "yes"

	baseName := filepath.Base(midiPath)
	ext := filepath.Ext(baseName)
	outName := strings.TrimSuffix(baseName, ext) + "_beepscape.txt"

	processTracks(file, selectedTracks, outName, filterShort)
}

func buildTempoMap(file *smf.SMF) []TempoChange {
	var changes []TempoChange

	for _, track := range file.Tracks {
		absTick := int64(0)
		for _, msg := range track {
			absTick += int64(msg.Delta)

			var bpm float64
			if msg.Message.GetMetaTempo(&bpm) {
				changes = append(changes, TempoChange{Tick: absTick, BPM: bpm})
			}
		}
	}

	sort.SliceStable(changes, func(i, j int) bool {
		return changes[i].Tick < changes[j].Tick
	})

	if len(changes) == 0 || changes[0].Tick != 0 {
		changes = append([]TempoChange{{Tick: 0, BPM: 120.0}}, changes...)
	}

	return changes
}

func ticksToMs(tick int64, tempoMap []TempoChange, ticks smf.MetricTicks) float64 {
	var ms float64
	prevTick := int64(0)
	prevBPM := 120.0

	for _, tc := range tempoMap {
		if tc.Tick >= tick {
			break
		}
		segmentTicks := tc.Tick - prevTick
		ms += ticks.Duration(prevBPM, uint32(segmentTicks)).Seconds() * 1000.0
		prevTick = tc.Tick
		prevBPM = tc.BPM
	}

	remainingTicks := tick - prevTick
	ms += ticks.Duration(prevBPM, uint32(remainingTicks)).Seconds() * 1000.0

	return ms
}

func processTracks(file *smf.SMF, trackIndices []int, outName string, filterShortEvents bool) {
	ticks, ok := file.TimeFormat.(smf.MetricTicks)
	if !ok {
		fmt.Println("error: midi file does not use metric (PPQ) ticks, cannot convert")
		return
	}

	tempoMap := buildTempoMap(file)

	var noteEvents []NoteEvent
	for _, idx := range trackIndices {
		track := file.Tracks[idx]
		absTick := int64(0)
		for _, msg := range track {
			absTick += int64(msg.Delta)

			var ch, note, vel uint8
			if msg.Message.GetNoteOn(&ch, &note, &vel) {
				if vel > 0 {
					noteEvents = append(noteEvents, NoteEvent{Tick: absTick, Type: "on", Note: note, Freq: midiNoteToFreq(note), Track: idx})
				} else {
					noteEvents = append(noteEvents, NoteEvent{Tick: absTick, Type: "off", Note: note, Freq: midiNoteToFreq(note), Track: idx})
				}
			} else if msg.Message.GetNoteOff(&ch, &note, &vel) {
				noteEvents = append(noteEvents, NoteEvent{Tick: absTick, Type: "off", Note: note, Freq: midiNoteToFreq(note), Track: idx})
			}
		}
	}

	sort.SliceStable(noteEvents, func(i, j int) bool {
		if noteEvents[i].Tick != noteEvents[j].Tick {
			return noteEvents[i].Tick < noteEvents[j].Tick
		}
		if noteEvents[i].Type != noteEvents[j].Type {
			return noteEvents[i].Type == "off"
		}
		return false
	})

	activeNotes := make(map[uint8]int)
	var currentNote uint8
	haveCurrentNote := false
	currentStartTick := int64(0)
	overlapWarned := false

	type beepOut struct {
		freq      int
		startTick int64
		endTick   int64
	}
	var beeps []beepOut

	highestActiveNote := func() (uint8, bool) {
		found := false
		var best uint8
		for n, count := range activeNotes {
			if count <= 0 {
				continue
			}
			if !found || n > best {
				best = n
				found = true
			}
		}
		return best, found
	}

	for _, ev := range noteEvents {
		switch ev.Type {
		case "on":
			activeNotes[ev.Note]++
			if len(activeNotes) > 1 && !overlapWarned {
				fmt.Printf("warning: overlapping notes detected at tick %d (track %d) — playing highest pitch only\n", ev.Tick, ev.Track)
				overlapWarned = true
			}
		case "off":
			if activeNotes[ev.Note] > 0 {
				activeNotes[ev.Note]--
				if activeNotes[ev.Note] == 0 {
					delete(activeNotes, ev.Note)
				}
			}
		}

		best, found := highestActiveNote()

		if haveCurrentNote {
			if !found || best != currentNote {
				if ev.Tick > currentStartTick {
					beeps = append(beeps, beepOut{
						freq:      midiNoteToFreq(currentNote),
						startTick: currentStartTick,
						endTick:   ev.Tick,
					})
				}
				haveCurrentNote = false
			}
		}

		if found && !haveCurrentNote {
			currentNote = best
			currentStartTick = ev.Tick
			haveCurrentNote = true
		}
	}

	if haveCurrentNote && len(noteEvents) > 0 {
		lastTick := noteEvents[len(noteEvents)-1].Tick
		if lastTick > currentStartTick {
			beeps = append(beeps, beepOut{
				freq:      midiNoteToFreq(currentNote),
				startTick: currentStartTick,
				endTick:   lastTick,
			})
		}
	}

	var optimizedCommands []string
	lastEndMs := 0.0
	const thresholdMs = 50

	for _, b := range beeps {
		startMs := ticksToMs(b.startTick, tempoMap, ticks)
		endMs := ticksToMs(b.endTick, tempoMap, ticks)

		gapMs := startMs - lastEndMs
		durationMs := endMs - startMs

		gapRounded := int(math.Round(gapMs))
		if gapRounded < 0 {
			gapRounded = 0
		}

		durationRounded := int(math.Round(durationMs))

		if filterShortEvents {
			if durationRounded < thresholdMs {
				lastEndMs = endMs
				continue
			}

			if gapRounded > 0 {
				optimizedCommands = append(optimizedCommands, fmt.Sprintf("delay %d", gapRounded))
			}
			optimizedCommands = append(optimizedCommands, fmt.Sprintf("beep %d %d", b.freq, durationRounded))
			lastEndMs = endMs
		} else {
			if gapRounded > 0 {
				optimizedCommands = append(optimizedCommands, fmt.Sprintf("delay %d", gapRounded))
			}
			if durationRounded > 0 {
				optimizedCommands = append(optimizedCommands, fmt.Sprintf("beep %d %d", b.freq, durationRounded))
			}
			lastEndMs = endMs
		}
	}

	f, err := os.Create(outName)
	if err != nil {
		fmt.Println("error creating output file:", err)
		return
	}
	defer f.Close()

	header := `hide-input
clear

<size=7>
// >>================================================================================<<
// ||  ██████╗████████████████████╗███████╗██████╗█████╗██████╗███████╗  ||
// ||  ██╔══████╔════██╔════██╔══████╔════██╔════██╔══████╔══████╔════╝  ||
// ||  ██████╔█████╗   █████╗   ██████╔█████████║         █████████████╔█████╗       ||
// ||  ██╔══████╔══╝   ██╔══╝   ██╔═══╝╚════████║         ██╔══████╔═══╝██╔══╝       ||
// ||  ██████╔████████████████║           ███████╚████████║    ████║           ███████╗  ||
// ||  ╚═════╝╚══════╚══════╚═╝           ╚══════╝╚═════╚═╝    ╚═╚═╝           ╚══════╝  ||
//>>=================================================================================<<
</size>
<size=10><color=gray>
//zoom out to see logo (non curved monitor only)
</color></size>
generated by beepscape, <color=blue>sabisa.xyz</color>/beepscape`

	f.WriteString(header + "\n")

	lineCount := 0
	for _, cmd := range optimizedCommands {
		f.WriteString(cmd + "\n")
		lineCount++
		if lineCount%600 == 0 {
			f.WriteString("\nclear\n\n")
		}
	}

	fmt.Printf("\ndone, saved %d commands to %s\n", len(optimizedCommands), outName)
	fmt.Print("press enter to close...")

	reader := bufio.NewReader(os.Stdin)

	_, _ = reader.ReadBytes('\n')
}
