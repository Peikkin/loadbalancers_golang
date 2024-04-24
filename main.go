package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server interface {
	Address() string
	IsAlive() bool
	Server(w http.ResponseWriter, r *http.Request)
}

type SimpleServer struct {
	address string
	proxy   *httputil.ReverseProxy
}

type LoadBalancer struct {
	Port            string
	RoundRobinCount int
	Servers         []Server
}

func NewSimpleServer(address string) *SimpleServer {
	serverUrl, err := url.Parse(address)
	if err != nil {
		log.Fatal().Err(err).Msgf("Не удалось получить URL сервера: %v", address)
	}

	return &SimpleServer{
		address: address,
		proxy:   httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		Port:            port,
		RoundRobinCount: 0,
		Servers:         servers,
	}
}

func (s *SimpleServer) Address() string {
	return s.address
}

func (s *SimpleServer) IsAlive() bool {
	return true
}

func (s *SimpleServer) Server(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) getNextServer() Server {
	server := lb.Servers[lb.RoundRobinCount%len(lb.Servers)]
	for !server.IsAlive() {
		lb.RoundRobinCount++
		server = lb.Servers[lb.RoundRobinCount%len(lb.Servers)]
	}
	lb.RoundRobinCount++
	return server
}

func (lb *LoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	tagretServer := lb.getNextServer()
	log.Info().Msgf("Запрос на переадресацию по адресу 'localhost:%v", tagretServer.Address())
	tagretServer.Server(w, r)
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	servers := []Server{
		NewSimpleServer("https://www.sdafg.org"),
		NewSimpleServer("https://www.dflxl.ai"),
		NewSimpleServer("https://www.youtube.com"),
	}
	lb := NewLoadBalancer("8080", servers)
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		lb.serverProxy(w, r)
	}
	http.HandleFunc("/next", handleRedirect)

	log.Info().Msgf("Сервер работает на порту 'localhost:%v'", lb.Port)
	if err := http.ListenAndServe(":"+lb.Port, nil); err != nil {
		log.Fatal().Err(err).Msg("Ошибка при запуске сервера")
	}
}
