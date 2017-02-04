package cmd

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/inoc603/dns-tool/types"
	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Response struct {
	Msg *dns.Msg
	Rtt time.Duration
	Err error
}

type Config struct {
	Server   string        `mapstructure:"server" yaml:"server"`
	Port     int           `mapstructure:"port" yaml:"port"`
	Proc     int           `mapstructure:"proc" yaml:"proc"`
	Duration time.Duration `mapstructure:"duration" yaml:"duration"`
	Timeout  time.Duration `mapstructure:"timeout" yaml:"timeout"`
}

func toQuestion(query string) (q dns.Question, err error) {
	parts := strings.Split(query, "/")
	if len(parts) != 2 {
		err = errors.New("invalid query")
		return
	}

	domain, t := parts[0], parts[1]

	q.Qclass = dns.ClassINET
	ok := true
	if q.Qtype, ok = dns.StringToType[t]; !ok {
		err = errors.New("invalid type")
		return
	}

	q.Name = dns.Fqdn(domain)

	return
}

// perfCmd represents the perf command
var perfCmd = &cobra.Command{
	Use:   "perf",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var questions []dns.Question
		for _, query := range args {
			q, err := toQuestion(query)
			if err != nil {
				log.Fatalln(err, query)
			}
			questions = append(questions, q)
		}

		fmt.Println(questions)

		var config Config
		qc := types.Counter{Interval: time.Second}
		go qc.Start()
		err := viper.Unmarshal(&config)
		if err != nil {
			log.Fatalln(err)
		}

		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
		// TODO: Work your own magic here
		fmt.Println("perf called")
		target := fmt.Sprintf(
			"%s:%d",
			config.Server,
			config.Port,
		)

		result := make(chan *Response, 100)

		msgPool := sync.Pool{
			New: func() interface{} {
				msg := new(dns.Msg)
				msg.Question = make([]dns.Question, 1)
				return msg
			},
		}

		runtime.GOMAXPROCS(config.Proc)

		for i := 0; i < viper.GetInt("client"); i++ {
			go func() {
				conn, _ := dns.DialTimeout("udp", target, config.Timeout)
				for {
					for _, q := range questions {
						msg := msgPool.Get().(*dns.Msg)
						msg.Question[0] = q
						deadline := time.Now().Add(config.Timeout)
						conn.SetWriteDeadline(deadline)
						conn.WriteMsg(msg)
						conn.SetReadDeadline(deadline)
						resp, err := conn.ReadMsg()
						rtt := time.Now().Add(config.Timeout).Sub(deadline)
						// resp, rtt, err := client.Exchange(msg, target)
						result <- &Response{resp, rtt, err}
						msgPool.Put(msg)
					}
				}
			}()
		}

		var errCount, successCount int64

		go func() {
			for res := range result {
				if res.Err != nil {
					errCount++
				} else {
					qc.Add()
					successCount++
				}
			}
		}()

		for {
			select {
			case <-time.After(config.Duration):
				qc.Stop()
				fmt.Println("errors:", errCount)
				fmt.Println("total", qc.Total())
				fmt.Println("qps:", qc.Avg())
				return
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(perfCmd)

	perfCmd.Flags().StringP("server", "s", "127.0.0.1", "dns server address")
	perfCmd.Flags().IntP("port", "p", 53, "dns server port")
	perfCmd.Flags().Int("proc", runtime.NumCPU(), "GOMAXPROCS")
	perfCmd.Flags().IntP("client", "c", runtime.NumCPU()*2, "concurrent client number")
	perfCmd.Flags().StringP("duration", "d", "10s", "benchmark duration")
	perfCmd.Flags().String("timeout", "1s", "client timeout")

	viper.BindPFlags(perfCmd.Flags())
}
