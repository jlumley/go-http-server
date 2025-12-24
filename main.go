package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

var (
	ErrInvalidRequestLine = errors.New("Failed to Parse Request Line")
	ErrInvalidHTTPVersion = errors.New("Failed to Parse HTTP Version")
	ErrInvalidHTTPHeader  = errors.New("Failed to Parse HTTP Header")
)

var requestFile string = "request.txt"
var rn []byte = []byte("\r\n")
var bufSize int = 1

type RequestState string

const (
	RequestLine RequestState = "request"
	HeaderLine  RequestState = "header"
	BodyLine    RequestState = "body"
	End         RequestState = "end"
)

type RequestMethod string

const (
	MethodGet    RequestMethod = "GET"
	MethodPost   RequestMethod = "POST"
	MethodPut    RequestMethod = "PUT"
	MethodDelete RequestMethod = "DELETE"
)

type Request struct {
	Method  RequestMethod
	Target  []byte
	Version []byte
	headers map[string][]byte
	body    []byte
}

type Response struct {
	Version []byte
	Status  int
	headers map[string][]byte
	body    []byte
}

func readLine(reader io.Reader, bufSize int, data []byte) ([]byte, []byte, io.Reader, error) {
	buffer := make([]byte, bufSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			data = append(data, buffer[:n]...)
			for {
				idx := bytes.Index(data, rn)

				if idx == -1 {
					break
				} else {
					line := data[:idx]
					data = data[idx+len(rn):]
					return line, data, reader, nil
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			return nil, data, reader, err
		}
	}

	return nil, data, reader, nil
}

func parseRequestLine(line []byte) (RequestMethod, []byte, []byte, error) {
	// GET /hello/world HTTP/1.1
	sep := []byte(" ")
	parts := bytes.Split(line, sep)

	if len(parts) != 3 {
		return "", nil, nil, ErrInvalidRequestLine
	}

	method := parts[0]
	target := parts[1]

	// validate http version
	versionSep := []byte("/")
	versionParts := bytes.Split(parts[2], versionSep)
	if len(versionParts) != 2 {
		return "", nil, nil, ErrInvalidHTTPVersion
	}
	if !bytes.Equal(versionParts[0], []byte("HTTP")) {
		return "", nil, nil, ErrInvalidHTTPVersion
	}
	version := versionParts[1]

	return RequestMethod(method), target, version, nil
}

func parseHeaderLine(line []byte) (string, []byte, error) {
	sep := []byte(":")
	header, value, found := bytes.Cut(line, sep)
	if !found {
		return "", nil, ErrInvalidHTTPHeader
	}
	return string(header), value, nil
}

func parseRequest(reader io.Reader) (*Request, error) {
	data := []byte("")
	line := []byte("")
	var method RequestMethod
	var target []byte
	var version []byte
	var err error
	headers := make(map[string][]byte)
	state := RequestLine
	hasBody := false

	for state != End {
		if state == BodyLine && hasBody == false {
			break
		}
		line, data, reader, err = readLine(reader, bufSize, data)
		if err != nil {
			fmt.Println(err)
		}
		switch state {
		case RequestLine:
			//parse Request line
			method, target, version, err = parseRequestLine(line)
			if err != nil {
				return nil, err
			}
			// move to next state
			state = HeaderLine
		case HeaderLine:

			// end of headers
			if bytes.Equal(line, []byte("")) {
				state = BodyLine
				continue
			}

			// pase header line
			header, value, err := parseHeaderLine(line)
			if err != nil {
				return nil, err
			}
			// TODO: handle appending lists
			headers[header] = value
			if header == "Content-Length" {
				hasBody = true
			}

		case BodyLine:
			fmt.Println("Ignoring payload for now")
			// move to next state
			state = End
		}
	}

	return &Request{method, target, version, headers, []byte("")}, nil
}

func buildResponse(response *Response) ([]byte, error) {
	//status-line = HTTP-version SP status-code SP [ reason-phrase ]
	var resp bytes.Buffer

	resp.WriteString("HTTP/")
	resp.Write(response.Version)
	resp.WriteString(" ")
	resp.Write([]byte(strconv.Itoa(response.Status)))
	resp.WriteString(" ")
	resp.WriteString(StatusText(response.Status))
	resp.Write(response.body)
	resp.Write(rn)

	return resp.Bytes(), nil
}

func handleRequest(conn net.Conn) {
	req, err := parseRequest(conn)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%v", *req)
	resp := Response{
		req.Version,
		200,
		nil,
		nil,
	}
	response, err := buildResponse(&resp)
	if err != nil {
		fmt.Println(err)
	}
	conn.Write(response)
	conn.Close()

}

func main() {
	// declare type as parseRequest returns io.Reader type
	var listener net.Listener
	port := ":6969"
	protocol := "tcp"

	listener, err := net.Listen(protocol, port)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(listener.Addr())
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
		}
		go handleRequest(conn)

	}

}
