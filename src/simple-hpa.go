package main

import (
	"flag"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"simple-hpa/src/handler"
	"simple-hpa/src/scale"
	"simple-hpa/src/tracer"
	"simple-hpa/src/utils"
)

const (
	netType = "udp"
	bufSize = 1024
	// 使用Pipeline处理
	// echoTime  = time.Second * 12
	// sleepTime = time.Millisecond * 113
	// checkTime = time.Minute + time.Millisecond*211
)

var (
	config *utils.Config
	// 使用Pipeline处理
	// calcuRecord map[string]*metrics.Calculate
	// scaleRecord map[string]*metrics.ScaleRecord
	k8sClient *scale.K8SClient
)

type Server struct {
	configpath  string
	serviceName string
	tracerURL   string
}

var server *Server

func init() {
	server = new(Server)
	flag.StringVar(&server.configpath, "config", "./config.yaml", "config path ...")
	flag.StringVar(&server.serviceName, "svc", "simple-hpa", "simple service name")
	flag.StringVar(&server.tracerURL, "trace", "jaeger.jaeger-infra:5775", "trace url")
	flag.Parse()
	pwd, _ := os.Getwd()
	cfg := path.Join(pwd, server.configpath)
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	config = utils.NewConfig(cfg)
	if config.AutoScale.Services == nil || len(config.AutoScale.Services) == 0 {
		log.Fatalln("WARNING, Auto scale dest service not defined")
	}
	// 使用Pipeline处理
	// calcuRecord = make(map[string]*metrics.Calculate)
	// scaleRecord = make(map[string]*metrics.ScaleRecord)
}

func main() {
	tracerService, closer := tracer.New(server.serviceName, server.tracerURL)
	defer closer.Close()
	opentracing.SetGlobalTracer(tracerService)
	//pprof
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	listenAddr := fmt.Sprintf("%s:%d", config.Listen.ListenAddr, config.Listen.Port)
	addr, err := net.ResolveUDPAddr(netType, listenAddr)
	if err != nil {
		log.Fatalln("Resolve udp address error ", err)
		return
	}
	conn, err := net.ListenUDP(netType, addr)
	if err != nil {
		log.Fatalln("Listen on ", addr.IP, "failed ", err)
		return
	}
	log.Printf("App listen on %s/%s", listenAddr, netType)
	log.Printf("Auto scale services: %s", config.AutoScale.Services)
	k8sClient = scale.NewK8SClient()
	defer conn.Close()
	/*
		// 使用Pipeline处理
		// go utils.DisplayQPS(calcuRecord, echoTime, sleepTime)
		// go utils.AutoScaleByQPS(scaleRecord, sleepTime, k8sClient, config)
		// avgTimeTick := time.Tick(time.Second * (60 / metrics.Count))
	*/
	// 使用线程池处理
	poolHandler := handler.NewPoolHandler(config, k8sClient)
	for {
		buf := make([]byte, bufSize)
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			log.Println("read error ", err)
			continue
		}
		/*
			// 使用pipeline处理
			go func() {
				accessChan := utils.ParseUDPData(buf[:n])
				accessChan = utils.FilterService(accessChan, config.AutoScale.Services)
				qpsChan := utils.CalculateQPS(accessChan, avgTimeTick, calcuRecord)
				utils.RecordQps(qpsChan, config.AutoScale.MaxQPS, config.AutoScale.SafeQPS, scaleRecord)
			}()
		*/
		// 使用工作池处理，限制协程数量
		poolHandler.Execute(buf[:n])
	}
}
