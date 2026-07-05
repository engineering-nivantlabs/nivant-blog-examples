```go
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
)

const (
	serviceName = "payment-service"
	consulAddr  = "localhost:8500"
	ttl         = 15 * time.Second
)

type DiscoveryCache struct {
	client *api.Client
	cache  map[string][]*api.ServiceEntry
	mu     sync.RWMutex
	ttl    time.Duration
}

func (dc *DiscoveryCache) GetService(serviceName string) ([]*api.ServiceEntry, error) {
	dc.mu.RLock()
	entries, ok := dc.cache[serviceName]
	dc.mu.RUnlock()
	if ok {
		return entries, nil
	}
	entries, _, err := dc.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return dc.fallbackToEtcd(serviceName)
	}
	dc.mu.Lock()
	dc.cache[serviceName] = entries
	dc.mu.Unlock()
	time.AfterFunc(dc.ttl, func() { dc.invalidate(serviceName) })
	return entries, nil
}

func (dc *DiscoveryCache) invalidate(serviceName string) {
	dc.mu.Lock()
	delete(dc.cache, serviceName)
	dc.mu.Unlock()
}

func main() {
	client, err := api.NewClient(&api.Config{Address: consulAddr})
	if err != nil {
		log.Fatalf("failed to create consul client: %v", err)
	}

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("%s-%s-%d", serviceName, hostname, os.Getpid())

	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Port:    8080,
		Address: getOutboundIP().String(),
		Check: &api.AgentServiceCheck{
			TTL:                            ttl.String(),
			DeregisterCriticalServiceAfter: "1m",
			Status:                         api.HealthPassing,
		},
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		log.Fatalf("service registration failed: %v", err)
	}
	defer client.Agent().ServiceDeregister(serviceID)

	go func() {
		ticker := time.NewTicker(ttl / 2)
		defer ticker.Stop()
		for range ticker.C {
			if err := client.Agent().UpdateTTL(serviceID, "healthy", api.HealthPassing); err != nil {
				log.Printf("failed to update TTL: %v", err)
			}
		}
	}()

	cache := &DiscoveryCache{client: client, ttl: ttl}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	http.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = serviceName
		}
		entries, err := cache.GetService(name)
		if err != nil {
			http.Error(w, "discovery failed", http.StatusServiceUnavailable)
			return
		}
		for _, e := range entries {
			fmt.Fprintf(w, "%s:%d\n", e.Service.Address, e.Service.Port)
		}
	})

	go http.ListenAndServe(":8080", nil)

	log.Printf("Service %s registered with Consul", serviceID