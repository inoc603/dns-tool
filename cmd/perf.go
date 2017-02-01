package cmd

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Response struct {
	Msg *dns.Msg
	Rtt time.Duration
	Err error
}

// perfCmd represents the perf command
var perfCmd = &cobra.Command{
	Use:   "perf",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
		// TODO: Work your own magic here
		fmt.Println("perf called")
		target := fmt.Sprintf(
			"%s:%d",
			viper.GetString("server"),
			viper.GetInt("port"),
		)

		duration, _ := time.ParseDuration(viper.GetString("duration"))

		fmt.Println(target, duration)

		result := make(chan *Response, 100)

		msgPool := sync.Pool{
			New: func() interface{} {
				msg := new(dns.Msg)
				msg.Question = make([]dns.Question, 1)
				return msg
			},
		}

		runtime.GOMAXPROCS(2)

		timeout := time.Second

		question := dns.Question{"baidu.com.", dns.TypeA, dns.ClassINET}

		for i := 0; i < viper.GetInt("client"); i++ {
			go func() {
				conn, _ := dns.DialTimeout("udp", target, time.Second)
				for {
					msg := msgPool.Get().(*dns.Msg)
					msg.Question[0] = question
					conn.SetWriteDeadline(time.Now().Add(timeout))
					conn.WriteMsg(msg)
					conn.SetReadDeadline(time.Now().Add(timeout))
					resp, err := conn.ReadMsg()
					// resp, rtt, err := client.Exchange(msg, target)

					result <- &Response{resp, 0, err}
					msgPool.Put(msg)
				}
			}()
		}

		var errCount, successCount int64

		go func() {
			for res := range result {
				if res.Err != nil {
					errCount++
				} else {
					successCount++
				}
			}
		}()

		for {
			select {
			case <-time.After(duration):
				fmt.Println(errCount, successCount/int64(duration))
				return
			case <-time.Tick(time.Second):
				fmt.Println(successCount)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(perfCmd)

	perfCmd.Flags().StringP("server", "s", "127.0.0.1", "dns server address")
	perfCmd.Flags().IntP("port", "p", 53, "dns server port")
	perfCmd.Flags().IntP("client", "c", 100, "concurrent client number")
	perfCmd.Flags().StringP("duration", "d", "10s", "benchmark duration")
	perfCmd.Flags().String("timeout", "1s", "client timeout")

	viper.BindPFlags(perfCmd.Flags())
}
