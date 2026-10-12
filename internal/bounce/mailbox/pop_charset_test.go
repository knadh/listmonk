package mailbox

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
)

func TestPOPScanUnknownCharset(t *testing.T) {
	const subscriber = "11111111-1111-4111-8111-111111111111"
	const campaign = "22222222-2222-4222-8222-222222222222"
	headers := "X-Listmonk-Subscriber: " + subscriber + "\r\nX-Listmonk-Campaign: " + campaign + "\r\nDate: Mon, 12 Oct 2026 12:00:00 +0000\r\nSubject: UTF7 bounce\r\n"
	cases := map[string]string{
		"root-utf7":      headers + "Content-Type: text/plain; charset=\"unicode-1-1-utf-7\"\r\n\r\nStatus: 5.1.1\r\n",
		"multipart-utf7": "Content-Type: multipart/report; boundary=b\r\n\r\n--b\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nNotice\r\n--b\r\n" + headers + "Content-Type: text/plain; charset=\"unicode-1-1-utf-7\"\r\n\r\nStatus: 5.1.1\r\n--b--\r\n",
		"root-utf8":      headers + "Content-Type: text/plain; charset=utf-8\r\n\r\nStatus: 5.1.1\r\n",
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				fmt.Fprint(conn, "+OK test server\r\n")
				reader := bufio.NewReader(conn)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					fields := strings.Fields(line)
					if len(fields) == 0 {
						return
					}
					switch fields[0] {
					case "STAT":
						fmt.Fprintf(conn, "+OK 1 %d\r\n", len(payload))
					case "RETR":
						fmt.Fprint(conn, "+OK message follows\r\n"+payload+".\r\n")
					case "QUIT":
						fmt.Fprint(conn, "+OK bye\r\n")
						return
					default:
						fmt.Fprint(conn, "+OK\r\n")
					}
				}
			}()
			port := listener.Addr().(*net.TCPAddr).Port
			var logs bytes.Buffer
			scanner := NewPOP(Opt{Host: "127.0.0.1", Port: port, AuthProtocol: "none"}, log.New(&logs, "", 0))
			ch := make(chan models.Bounce, 2)
			if err := scanner.Scan(1, ch); err != nil {
				t.Fatal(err)
			}
			<-done
			if len(ch) != 1 {
				t.Fatalf("wanted one bounce, got %d; logs=%s", len(ch), logs.String())
			}
			b := <-ch
			if b.SubscriberUUID != subscriber || b.CampaignUUID != campaign || b.Type != models.BounceTypeHard {
				t.Fatalf("incorrect bounce identifiers/classification: %#v", b)
			}
			if b.CreatedAt.Format(time.RFC3339) != "2026-10-12T12:00:00Z" {
				t.Fatalf("wrong timestamp: %v", b.CreatedAt)
			}
			if strings.Contains(logs.String(), "error") {
				t.Fatalf("recoverable charset produced error log: %s", logs.String())
			}
		})
	}
}
