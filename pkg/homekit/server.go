package homekit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
)

type HandlerFunc func(net.Conn) error

type Server interface {
	ServerPair
	ServerAccessory
}

type ServerPair interface {
	GetPair(id string) []byte
	AddPair(id string, public []byte, permissions byte)
	DelPair(id string)
}

type ServerAccessory interface {
	GetAccessories(conn net.Conn) []*hap.Accessory
	GetCharacteristic(conn net.Conn, aid uint8, iid uint64) any
	SetCharacteristic(conn net.Conn, aid uint8, iid uint64, value any)
	GetImage(conn net.Conn, width, height int) []byte
}

func ServerHandler(server Server) HandlerFunc {
	return handleRequest(func(conn net.Conn, req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case hap.PathPairings:
			return handlePairings(req, server)

		case hap.PathAccessories:
			body := hap.JSONAccessories{Value: server.GetAccessories(conn)}
			return makeResponse(hap.MimeJSON, body)

		case hap.PathCharacteristics:
			switch req.Method {
			case "GET":
				var v hap.JSONCharacters

				id := req.URL.Query().Get("id")
				for _, id = range strings.Split(id, ",") {
					s1, s2, _ := strings.Cut(id, ".")
					aid, _ := strconv.Atoi(s1)
					iid, _ := strconv.ParseUint(s2, 10, 64)
					val := server.GetCharacteristic(conn, uint8(aid), iid)

					v.Value = append(v.Value, hap.JSONCharacter{AID: uint8(aid), IID: iid, Value: val})
				}

				return makeResponse(hap.MimeJSON, v)

			case "PUT":
				var v struct {
					Value []struct {
						AID   uint8  `json:"aid"`
						IID   uint64 `json:"iid"`
						Value any    `json:"value"`
						Event any    `json:"ev"`
						R     *bool  `json:"r,omitempty"`
					} `json:"characteristics"`
				}
				if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
					return nil, err
				}

				var writeResponses []hap.JSONCharacter
				hasError, hasResponse := false, false
				findChar := func(aid uint8, iid uint64) *hap.Character {
					accs := server.GetAccessories(conn)
					for _, acc := range accs {
						if acc.AID != aid {
							continue
						}
						return acc.GetCharacterByID(iid)
					}
					return nil
				}

				for _, c := range v.Value {
					status := 0
					if c.Value != nil {
						if checked, ok := server.(interface {
							SetCharacteristicStatus(net.Conn, uint8, uint64, any) int
						}); ok {
							status = checked.SetCharacteristicStatus(conn, c.AID, c.IID, c.Value)
						} else {
							server.SetCharacteristic(conn, c.AID, c.IID, c.Value)
						}
					}
					if c.Event != nil {
						// subscribe/unsubscribe to events
						if char := findChar(c.AID, c.IID); char != nil {
							if ev, ok := c.Event.(bool); ok && ev {
								char.AddListener(conn)
							} else {
								char.RemoveListener(conn)
							}
						}
					}
					response := hap.JSONCharacter{AID: c.AID, IID: c.IID, Status: status}
					if status != 0 {
						hasError = true
					}
					if c.R != nil && *c.R && status == 0 {
						hasResponse = true
						response.Value = server.GetCharacteristic(conn, c.AID, c.IID)
					}
					writeResponses = append(writeResponses, response)
				}
				if hasError || hasResponse {
					res, err := makeResponse(hap.MimeJSON, hap.JSONCharacters{Value: writeResponses})
					if err == nil && hasError {
						res.StatusCode = http.StatusMultiStatus
					}
					return res, err
				}

				res := &http.Response{
					StatusCode: http.StatusNoContent,
					Proto:      "HTTP",
					ProtoMajor: 1,
					ProtoMinor: 1,
				}
				return res, nil
			}

		case hap.PathResource:
			var v struct {
				Width  int    `json:"image-width"`
				Height int    `json:"image-height"`
				Type   string `json:"resource-type"`
				Reason *int   `json:"reason"`
			}
			if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
				return nil, err
			}

			if snapshots, ok := server.(interface {
				GetImageWithReason(net.Conn, int, int, int) ([]byte, int)
			}); ok {
				reason := -1
				if v.Reason != nil {
					reason = *v.Reason
				}
				body, status := snapshots.GetImageWithReason(conn, v.Width, v.Height, reason)
				if status != 0 {
					res, err := makeResponse(hap.MimeJSON, map[string]int{"status": status})
					if err == nil {
						res.StatusCode = http.StatusBadRequest
					}
					return res, err
				}
				return makeResponse("image/jpeg", body)
			}
			body := server.GetImage(conn, v.Width, v.Height)
			return makeResponse("image/jpeg", body)
		}

		return nil, errors.New("hap: unsupported path: " + req.RequestURI)
	})
}

func handleRequest(handle func(conn net.Conn, req *http.Request) (*http.Response, error)) HandlerFunc {
	return func(conn net.Conn) error {
		rw := bufio.NewReaderSize(conn, 16*1024)
		wr := bufio.NewWriterSize(conn, 16*1024)
		for {
			req, err := http.ReadRequest(rw)
			//debug(req)
			if err != nil {
				return err
			}

			res, err := handle(conn, req)
			//debug(res)
			if err != nil {
				return err
			}

			if err = res.Write(wr); err != nil {
				return err
			}
			if err = wr.Flush(); err != nil {
				return err
			}
		}
	}
}

func handlePairings(req *http.Request, srv ServerPair) (*http.Response, error) {
	cmd := struct {
		Method      byte   `tlv8:"0"`
		Identifier  string `tlv8:"1"`
		PublicKey   string `tlv8:"3"`
		State       byte   `tlv8:"6"`
		Permissions byte   `tlv8:"11"`
	}{}

	if err := tlv8.UnmarshalReader(req.Body, req.ContentLength, &cmd); err != nil {
		return nil, err
	}

	switch cmd.Method {
	case 3: // add
		srv.AddPair(cmd.Identifier, []byte(cmd.PublicKey), cmd.Permissions)
	case 4: // delete
		srv.DelPair(cmd.Identifier)
	}

	body := struct {
		State byte `tlv8:"6"`
	}{
		State: hap.StateM2,
	}

	return makeResponse(hap.MimeTLV8, body)
}

func makeResponse(mime string, v any) (*http.Response, error) {
	var body []byte
	var err error

	switch mime {
	case hap.MimeJSON:
		body, err = json.Marshal(v)
	case hap.MimeTLV8:
		body, err = tlv8.Marshal(v)
	case "image/jpeg":
		body = v.([]byte)
	}

	if err != nil {
		return nil, err
	}

	res := &http.Response{
		StatusCode: http.StatusOK,
		Proto:      "HTTP",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Content-Type":   []string{mime},
			"Content-Length": []string{strconv.Itoa(len(body))},
		},
		ContentLength: int64(len(body)),
		Body:          io.NopCloser(bytes.NewReader(body)),
	}
	return res, nil
}
