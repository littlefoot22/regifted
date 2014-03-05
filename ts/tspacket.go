package main

import (
	"fmt"
	"log"
	"regifted/data"
)

type TsPacket struct {
	byteChunk []byte

	sync  uint
	flags uint

	payload []byte

	transportError bool
	unitStart      bool
	priority       bool
	pid            uint
	scramble       uint
	hasAdaptation  bool
	hasPayload     bool
	continuity     uint
	adaptation     Adaptation
}

//TsPacket Read
//TsPacket
//The first read of every packet
//flags:
//asserts 'G' - 8 bits
//transportError - Set by demodulator if can't correct errors in the stream, to tell the demultiplexer that the packet has an uncorrectable error (1 bit)
//unitStart - Boolean flag with a value of true meaning the start of PES data or PSI otherwise zero only. (1 bit)
//priority - Boolean flag with a value of true meaning the current packet has a higher priority than other packets with the same PID. (1 bit)
//pid - Packet Identifier (13 bit)
//scramble - '00' = Not scrambled, '01' = Reserved for future use, '10' = Scrambled with even key, '11' = Scrambled with odd key (2 bits)
//hasAdaptation - If adaption field exist value is true (0 or more bits)
//hasPayload -  If contains payload value is true (0 or more bits)
//continuity - Sequence number of payload packets, Incremented only when a payload is present (i.e., payload value is true) (4 bits)
func (tsPacket *TsPacket) Read() {
	if tsPacket.byteChunk == nil {
		log.Printf("attempted to read from nil pointer\n")
		return
	}

	var flags uint = 0

	reader := data.NewReader(tsPacket.byteChunk)

	tsPacket.sync = reader.Read(1)

	if tsPacket.sync != 0x47 {
		log.Printf("sync byte not 'G'\n")
		return
	}
	// asserted tsPacket.sync == 'G'

	flags = reader.Read(2)

	tsPacket.transportError = flags&0x8000 > 0
	tsPacket.unitStart = flags&0x4000 > 0
	tsPacket.priority = flags&0x2000 > 0
	tsPacket.pid = flags & 0x1fff
	fmt.Println("pid", tsPacket.pid)

	flags = reader.Read(1)

	tsPacket.scramble = flags & 0xc0 >> 6
	tsPacket.hasAdaptation = flags&0x20 > 0
	tsPacket.hasPayload = flags&0x10 > 0
	tsPacket.continuity = flags & 0x0f

	tsPacket.Print()

	if tsPacket.hasAdaptation {
		tsPacket.adaptation.byteChunk = reader.ReadBytes(reader.Size - reader.Cursor)
		tsPacket.adaptation.Read()
	}

	if tsPacket.pid == 0 {
		state.pat.byteChunk = reader.ReadBytes(reader.Size - reader.Cursor)

		state.pat.unitStart = tsPacket.unitStart
		state.pat.Read()
	}

	if pmt, ok := state.pmtConstructors[tsPacket.pid]; ok {
		pmt.unitStart = tsPacket.unitStart
		pmt.byteChunk = reader.ReadBytes(reader.Size - reader.Cursor)
		pmt.Read()
	}

	if elementaryStreamPacket, ok := state.elementaryConstructors[tsPacket.pid]; ok {

		elementaryStreamPacket.pid = tsPacket.pid
		elementaryStreamPacket.unitStart = tsPacket.unitStart

		if tsPacket.hasAdaptation {
			elementaryStreamPacket.payload = tsPacket.adaptation.payload
		} else {
			elementaryStreamPacket.payload = reader.ReadBytes(reader.Size - reader.Cursor)
		}

		elementaryStreamPacket.Dispatch()
		elementaryStreamPacket.Print()
	}
}

func (tsPacket *TsPacket) Print() {

	fmt.Println("\n:::TsRead:::\n")
	fmt.Println("sync = ", tsPacket.sync)
	fmt.Println("transportError = ", tsPacket.transportError)
	fmt.Println("unitStart = ", tsPacket.unitStart)
	fmt.Println("priority = ", tsPacket.priority)
	fmt.Println("pid = ", tsPacket.pid)
	fmt.Println("scramble = ", tsPacket.scramble)
	fmt.Println("hasAdaptation = ", tsPacket.hasAdaptation)
	fmt.Println("hasPayload = ", tsPacket.hasPayload)
	fmt.Println("continuity = ", tsPacket.continuity)
}
