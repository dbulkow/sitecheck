package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type Telnet struct {
	deadline time.Time
}

const (
	debug = false

	CommandXEOF  = 236 // End Of File (EOF is already used)
	CommandSUSP  = 237 // Suspend Process
	CommandABORT = 238 // Abort Process
	CommandEOR   = 239 // End Of Record (transparent mode)
	CommandSE    = 240 // Subnegotiation Eng
	CommandNOP   = 241 // No Operation
	CommandDM    = 242 // Data Mark
	CommandBRK   = 243 // Break
	CommandIP    = 244 // Interrupt Process
	CommandAO    = 245 // Abort Output
	CommandAYT   = 246 // Are You There
	CommandEC    = 247 // Erase Character
	CommandEL    = 248 // Erase Line
	CommandGA    = 249 // Go Ahead
	CommandSB    = 250 // Subnegotiation
	CommandWILL  = 251 // Will Perform
	CommandWONT  = 252 // Won't Perform
	CommandDO    = 253 // Do Perform
	CommandDONT  = 254 // Don't Perform
	CommandIAC   = 255 // Interpret As Command

	OptionBINARY         = 0   // 8-bit data path
	OptionECHO           = 1   // echo
	OptionRCP            = 2   // prepare to reconnect
	OptionSGA            = 3   // suppress go ahead
	OptionNAMS           = 4   // approximate message size
	OptionSTATUS         = 5   // give status
	OptionTM             = 6   // timing mark
	OptionRCTE           = 7   // remote controlled transmission and echo
	OptionNAOL           = 8   // negotiate about output line width
	OptionNAOP           = 9   // negotiate about output page size
	OptionNAOCRD         = 10  // negotiate about CR disposition
	OptionNAOHTS         = 11  // negotiate about horizontal tabstops
	OptionNAOHTD         = 12  // negotiate about horizontal tab disposition
	OptionNAOFFD         = 13  // negotiate about formfeed disposition
	OptionNAOVTS         = 14  // negotiate about vertical tab stops
	OptionNAOVTD         = 15  // negotiate about vertical tab disposition
	OptionNAOLFD         = 16  // negotiate about output LF disposition
	OptionXASCII         = 17  // extended ascii character set
	OptionLOGOUT         = 18  // force logout
	OptionBM             = 19  // byte macro
	OptionDET            = 20  // data entry terminal
	OptionSUPDUP         = 21  // supdup protocol
	OptionSUPDUPOUTPUT   = 22  // supdup output
	OptionSNDLOC         = 23  // send location
	OptionTTYPE          = 24  // terminal type
	OptionEOR            = 25  // end or record
	OptionTUID           = 26  // TACACS user identification
	OptionOUTMRK         = 27  // output marking
	OptionTTYLOC         = 28  // terminal location number
	Option3270REGIME     = 29  // 3270 regime
	OptionX3PAD          = 30  // X.3 PAD
	OptionNAWS           = 31  // window size
	OptionTSPEED         = 32  // terminal speed
	OptionLFLOW          = 33  // remote flow control
	OptionLINEMODE       = 34  // Linemode option
	OptionXDISPLOC       = 35  // X Display Location
	OptionOLD_ENVIRON    = 36  // Old - Environment variables
	OptionAUTHENTICATION = 37  // Authenticate
	OptionENCRYPT        = 38  // Encryption option
	OptionNEW_ENVIRON    = 39  // New - Environment variables
	OptionEXOPL          = 255 // extended-options-list
)

var commandText = map[int]string{
	CommandXEOF:  "xEOF",
	CommandSUSP:  "SUSP",
	CommandABORT: "ABORT",
	CommandEOR:   "EOR",
	CommandSE:    "SE",
	CommandNOP:   "NOP",
	CommandDM:    "DM",
	CommandBRK:   "BRK",
	CommandIP:    "IP",
	CommandAO:    "AO",
	CommandAYT:   "AYT",
	CommandEC:    "EC",
	CommandEL:    "EL",
	CommandGA:    "GA",
	CommandSB:    "SB",
	CommandWILL:  "WILL",
	CommandWONT:  "WONT",
	CommandDO:    "DO",
	CommandDONT:  "DONT",
	CommandIAC:   "IAC",
}

func (t *Telnet) CommandText(cmd int) string {
	return commandText[cmd]
}

var optionText = map[int]string{
	OptionBINARY:         "8-bit data path",
	OptionECHO:           "echo",
	OptionRCP:            "prepare to reconnect",
	OptionSGA:            "suppress go ahead",
	OptionNAMS:           "approximate message size",
	OptionSTATUS:         "give status",
	OptionTM:             "timing mark",
	OptionRCTE:           "remote controlled transmission and echo",
	OptionNAOL:           "negotiate about output line width",
	OptionNAOP:           "negotiate about output page size",
	OptionNAOCRD:         "negotiate about CR disposition",
	OptionNAOHTS:         "negotiate about horizontal tabstops",
	OptionNAOHTD:         "negotiate about horizontal tab disposition",
	OptionNAOFFD:         "negotiate about formfeed disposition",
	OptionNAOVTS:         "negotiate about vertical tab stops",
	OptionNAOVTD:         "negotiate about vertical tab disposition",
	OptionNAOLFD:         "negotiate about output LF disposition",
	OptionXASCII:         "extended ascii character set",
	OptionLOGOUT:         "force logout",
	OptionBM:             "byte macro",
	OptionDET:            "data entry terminal",
	OptionSUPDUP:         "supdup protocol",
	OptionSUPDUPOUTPUT:   "supdup output",
	OptionSNDLOC:         "send location",
	OptionTTYPE:          "terminal type",
	OptionEOR:            "end or record",
	OptionTUID:           "TACACS user identification",
	OptionOUTMRK:         "output marking",
	OptionTTYLOC:         "terminal location number",
	Option3270REGIME:     "3270 regime",
	OptionX3PAD:          "X.3 PAD",
	OptionNAWS:           "window size",
	OptionTSPEED:         "terminal speed",
	OptionLFLOW:          "remote flow control",
	OptionLINEMODE:       "Linemode option",
	OptionXDISPLOC:       "X Display Location",
	OptionOLD_ENVIRON:    "Old - Environment variables",
	OptionAUTHENTICATION: "Authenticate",
	OptionENCRYPT:        "Encryption option",
	OptionNEW_ENVIRON:    "New - Environment variables",
	OptionEXOPL:          "extended-options-list",
}

func (t *Telnet) OptionText(option int) string {
	return optionText[option]
}

func (t *Telnet) debugf(format string, args ...interface{}) {
	if debug {
		fmt.Printf(format, args...)
	}
}

func (t *Telnet) sendAYT(c net.Conn) error {
	t.debugf("sendAYT\n")

	b := []byte{CommandIAC, CommandAYT}

	err := c.SetWriteDeadline(t.deadline)
	if err != nil {
		return err
	}

	n, err := c.Write(b)
	if err != nil {
		return err
	}

	if n != len(b) {
		return err
	}

	t.debugf("\t\t\t\t%2.2x %s\n", b[0], t.CommandText(int(b[0])))
	t.debugf("\t\t\t\t%2.2x %s\n", b[1], t.CommandText(int(b[1])))

	return nil
}

func (t *Telnet) decodeCommand(buf []byte) int {
	if buf[0] != CommandIAC {
		return 0
	}

	t.debugf("%2.2x %s\n", buf[0], t.CommandText(int(buf[0])))
	t.debugf("%2.2x %s\n", buf[1], t.CommandText(int(buf[1])))

	i := 1

	switch buf[i] {
	case CommandDO:
		fallthrough
	case CommandDONT:
		fallthrough
	case CommandWILL:
		fallthrough
	case CommandWONT:
		i++
		t.debugf("%2.2x %s\n", buf[i], t.OptionText(int(buf[i])))
	}

	return i
}

func (t *Telnet) recv(c net.Conn) (int, error) {
	buf := make([]byte, 512)

	count := 0
	for {
		err := c.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		if err != nil {
			return count, fmt.Errorf("setting read deadline: %v\n", err)
		}

		n, err := c.Read(buf)
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				return count, nil
			}
			return count, fmt.Errorf("read error: %v\n", err)
		}

		for i := 0; i < n; i++ {
			if cnt := t.decodeCommand(buf[i:n]); cnt > 0 {
				i += cnt
				continue
			}

			t.debugf("%2.2x %c\n", buf[i], buf[i])
		}

		count += n

		if time.Now().After(t.deadline) {
			return count, errors.New("deadline exceeded")
		}
	}
}

func (t *Telnet) Check(srv Service) (bool, error) {
	t.deadline = time.Now().Add(time.Duration(time.Duration(srv.Timeout) * time.Second))

	c, err := net.DialTimeout("tcp", srv.URL, t.deadline.Sub(time.Now()))
	if err != nil {
		return false, err
	}

	_, err = t.recv(c)
	if err != nil {
		return false, err
	}

	if err := t.sendAYT(c); err != nil {
		return false, err
	}

	n, err := t.recv(c)
	if err != nil {
		return false, err
	}

	if n == 0 {
		return false, errors.New("no response to AYT")
	}

	return true, nil
}
