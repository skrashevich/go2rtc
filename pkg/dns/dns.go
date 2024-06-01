package dns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strings"
)

type MsgHdr struct {
	Response           bool // QR - Response flag
	Opcode             int  // Opcode - specifies kind of query
	Authoritative      bool // AA - Authoritative flag
	Truncated          bool // TC - Truncated flag
	RecursionDesired   bool // RD - Recursion desired
	RecursionAvailable bool // RA - Recursion available
	Zero               bool // Z - Reserved, must be zero
	AuthenticatedData  bool // AD - Authenticated data (DNSSEC)
	CheckingDisabled   bool // CD - Checking disabled (DNSSEC)
	Rcode              int  // Response code
}

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type Question struct {
	Name   string
	Qtype  uint16
	Qclass uint16
}

type RR interface {
	Header() *RR_Header
}

type RR_Header struct {
	Name     string
	Rrtype   uint16
	Class    uint16
	Ttl      uint32
	Rdlength uint16
}

type PTR struct {
	Hdr RR_Header
	Ptr string
}

// Header implements RR.
func (d *PTR) Header() *RR_Header {
	panic("unimplemented")
}

type TXT struct {
	Hdr RR_Header
	Txt []string
}

// Header implements RR.
func (d *TXT) Header() *RR_Header {
	panic("unimplemented")
}

type SRV struct {
	Hdr      RR_Header
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
}

// Header implements RR.
func (d *SRV) Header() *RR_Header {
	panic("unimplemented")
}

type A struct {
	Hdr RR_Header
	A   net.IP
}

// Header implements RR.
func (d *A) Header() *RR_Header {
	panic("unimplemented")
}

const (
	TypePTR = 12
	TypeTXT = 16
	TypeSRV = 33
	TypeA   = 1

	ClassINET = 1
)

type Msg struct {
	MsgHdr
	Header     DNSHeader
	Question   []Question
	Answer     []RR
	Authority  []RR
	Additional []RR
	Ns         []RR
	Extra      []RR
}

func writeName(buffer *bytes.Buffer, name string) error {
	parts := strings.Split(name, ".")
	for _, part := range parts {
		if len(part) > 63 {
			return errors.New("label is too long")
		}
		buffer.WriteByte(byte(len(part)))
		buffer.WriteString(part)
	}
	buffer.WriteByte(0) // End of name
	return nil
}

func parseName(buffer *bytes.Buffer) (string, error) {
	var name []string
	for {
		length, err := buffer.ReadByte()
		if err != nil {
			return "", err
		}
		if length == 0 {
			break
		}
		part := make([]byte, length)
		_, err = buffer.Read(part)
		if err != nil {
			return "", err
		}
		name = append(name, string(part))
	}
	return strings.Join(name, "."), nil
}

func (msg *Msg) Pack() ([]byte, error) {
	buffer := bytes.Buffer{}
	err := binary.Write(&buffer, binary.BigEndian, msg.Header)
	if err != nil {
		return nil, err
	}
	for _, q := range msg.Question {
		err := writeName(&buffer, q.Name)
		if err != nil {
			return nil, err
		}
		err = binary.Write(&buffer, binary.BigEndian, q.Qtype)
		if err != nil {
			return nil, err
		}
		err = binary.Write(&buffer, binary.BigEndian, q.Qclass)
		if err != nil {
			return nil, err
		}
	}
	// Pack the answers, authority, and additional sections
	for _, rr := range msg.Answer {
		err = packRR(&buffer, rr)
		if err != nil {
			return nil, err
		}
	}
	for _, rr := range msg.Authority {
		err = packRR(&buffer, rr)
		if err != nil {
			return nil, err
		}
	}
	for _, rr := range msg.Additional {
		err = packRR(&buffer, rr)
		if err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func (msg *Msg) Unpack(b []byte) error {
	buffer := bytes.NewBuffer(b)
	err := binary.Read(buffer, binary.BigEndian, &msg.Header)
	if err != nil {
		return err
	}
	msg.Question = make([]Question, msg.Header.QDCount)
	for i := range msg.Question {
		name, err := parseName(buffer)
		if err != nil {
			return err
		}
		var q Question
		q.Name = name
		err = binary.Read(buffer, binary.BigEndian, &q.Qtype)
		if err != nil {
			return err
		}
		err = binary.Read(buffer, binary.BigEndian, &q.Qclass)
		if err != nil {
			return err
		}
		msg.Question[i] = q
	}
	// Unpack the answer, authority, and additional sections
	msg.Answer, err = unpackRRs(buffer, msg.Header.ANCount)
	if err != nil {
		return err
	}
	msg.Authority, err = unpackRRs(buffer, msg.Header.NSCount)
	if err != nil {
		return err
	}
	msg.Additional, err = unpackRRs(buffer, msg.Header.ARCount)
	if err != nil {
		return err
	}
	return nil
}

func packRR(buffer *bytes.Buffer, rr RR) error {
	hdr := rr.Header()
	err := writeName(buffer, hdr.Name)
	if err != nil {
		return err
	}
	err = binary.Write(buffer, binary.BigEndian, hdr.Rrtype)
	if err != nil {
		return err
	}
	err = binary.Write(buffer, binary.BigEndian, hdr.Class)
	if err != nil {
		return err
	}
	err = binary.Write(buffer, binary.BigEndian, hdr.Ttl)
	if err != nil {
		return err
	}
	rrdata, err := rrDataPack(rr)
	if err != nil {
		return err
	}
	hdr.Rdlength = uint16(len(rrdata))
	err = binary.Write(buffer, binary.BigEndian, hdr.Rdlength)
	if err != nil {
		return err
	}
	buffer.Write(rrdata)
	return nil
}

func rrDataPack(rr RR) ([]byte, error) {
	buffer := bytes.Buffer{}
	switch v := rr.(type) {
	case *PTR:
		err := writeName(&buffer, v.Ptr)
		if err != nil {
			return nil, err
		}
	case *TXT:
		for _, txt := range v.Txt {
			buffer.WriteByte(byte(len(txt)))
			buffer.WriteString(txt)
		}
	case *SRV:
		err := binary.Write(&buffer, binary.BigEndian, v.Priority)
		if err != nil {
			return nil, err
		}
		err = binary.Write(&buffer, binary.BigEndian, v.Weight)
		if err != nil {
			return nil, err
		}
		err = binary.Write(&buffer, binary.BigEndian, v.Port)
		if err != nil {
			return nil, err
		}
		err = writeName(&buffer, v.Target)
		if err != nil {
			return nil, err
		}
	case *A:
		buffer.Write(v.A.To4())
	default:
		return nil, errors.New("unknown RR type")
	}
	return buffer.Bytes(), nil
}

func unpackRRs(buffer *bytes.Buffer, count uint16) ([]RR, error) {
	var rrs []RR
	for i := 0; i < int(count); i++ {
		hdr := RR_Header{}
		name, err := parseName(buffer)
		if err != nil {
			return nil, err
		}
		hdr.Name = name
		err = binary.Read(buffer, binary.BigEndian, &hdr.Rrtype)
		if err != nil {
			return nil, err
		}
		err = binary.Read(buffer, binary.BigEndian, &hdr.Class)
		if err != nil {
			return nil, err
		}
		err = binary.Read(buffer, binary.BigEndian, &hdr.Ttl)
		if err != nil {
			return nil, err
		}
		err = binary.Read(buffer, binary.BigEndian, &hdr.Rdlength)
		if err != nil {
			return nil, err
		}
		rr, err := rrUnpack(buffer, hdr)
		if err != nil {
			return nil, err
		}
		rrs = append(rrs, rr)
	}
	return rrs, nil
}

func rrUnpack(buffer *bytes.Buffer, hdr RR_Header) (RR, error) {
	switch hdr.Rrtype {
	case TypePTR:
		ptr := &PTR{Hdr: hdr}
		name, err := parseName(buffer)
		if err != nil {
			return nil, err
		}
		ptr.Ptr = name
		return ptr, nil
	case TypeTXT:
		txt := &TXT{Hdr: hdr}
		txt.Txt = make([]string, 0)
		for buffer.Len() > 0 {
			length, err := buffer.ReadByte()
			if err != nil {
				return nil, err
			}
			txtPart := make([]byte, length)
			if _, err := buffer.Read(txtPart); err != nil {
				return nil, err
			}
			txt.Txt = append(txt.Txt, string(txtPart))
		}
		return txt, nil
	case TypeSRV:
		srv := &SRV{Hdr: hdr}
		err := binary.Read(buffer, binary.BigEndian, &srv.Priority)
		if err != nil {
			return nil, err
		}
		err = binary.Read(buffer, binary.BigEndian, &srv.Weight)
		if err != nil {
			return nil, err
		}
		err = binary.Read(buffer, binary.BigEndian, &srv.Port)
		if err != nil {
			return nil, err
		}
		name, err := parseName(buffer)
		if err != nil {
			return nil, err
		}
		srv.Target = name
		return srv, nil
	case TypeA:
		a := &A{Hdr: hdr}
		ip := make([]byte, 4)
		if _, err := buffer.Read(ip); err != nil {
			return nil, err
		}
		a.A = net.IP(ip)
		return a, nil
	default:
		return nil, errors.New("unknown RR type")
	}
}
