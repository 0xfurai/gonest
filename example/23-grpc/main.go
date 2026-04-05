package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/0xfurai/gonest/microservice"
	"github.com/0xfurai/gonest/microservice/grpc"
)

// --- DTOs ---

// Hero represents a hero entity exchanged over gRPC.
type Hero struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// CreateHeroRequest is the request payload for creating a hero.
type CreateHeroRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// HeroByIDRequest is the request payload for finding a hero by ID.
type HeroByIDRequest struct {
	ID int `json:"id"`
}

// --- Service ---

// HeroesService manages the in-memory hero store on the server side.
type HeroesService struct {
	mu     sync.RWMutex
	heroes []Hero
	nextID int
}

// NewHeroesService creates a new HeroesService with sample data.
func NewHeroesService() *HeroesService {
	return &HeroesService{
		nextID: 3,
		heroes: []Hero{
			{ID: 1, Name: "Superman", Type: "Kryptonian"},
			{ID: 2, Name: "Batman", Type: "Human"},
		},
	}
}

// FindAll returns all heroes.
func (s *HeroesService) FindAll() []Hero {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Hero, len(s.heroes))
	copy(result, s.heroes)
	return result
}

// FindOne returns a hero by ID.
func (s *HeroesService) FindOne(id int) *Hero {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, h := range s.heroes {
		if h.ID == id {
			return &h
		}
	}
	return nil
}

// Create adds a new hero.
func (s *HeroesService) Create(name, heroType string) Hero {
	s.mu.Lock()
	defer s.mu.Unlock()
	hero := Hero{
		ID:   s.nextID,
		Name: name,
		Type: heroType,
	}
	s.nextID++
	s.heroes = append(s.heroes, hero)
	return hero
}

// --- Server Setup ---

// startServer creates and starts a gRPC-style microservice server that handles
// hero-related message patterns and events.
func startServer(port int) *grpc.Server {
	svc := NewHeroesService()

	server := grpc.NewServer(grpc.Options{
		Host:        "localhost",
		Port:        port,
		ServiceName: "hero-service",
	})

	// Register message handler: findAllHeroes (request/response)
	server.AddMessageHandler(
		microservice.Pattern{Cmd: "findAllHeroes"},
		func(ctx *microservice.MessageContext) (any, error) {
			log.Printf("[Server] Handling findAllHeroes (transport: gRPC)")
			heroes := svc.FindAll()
			return heroes, nil
		},
	)

	// Register message handler: findHeroById (request/response)
	server.AddMessageHandler(
		microservice.Pattern{Cmd: "findHeroById"},
		func(ctx *microservice.MessageContext) (any, error) {
			var req HeroByIDRequest
			if err := json.Unmarshal(ctx.Data, &req); err != nil {
				return nil, fmt.Errorf("invalid request data: %w", err)
			}
			log.Printf("[Server] Handling findHeroById: id=%d", req.ID)
			hero := svc.FindOne(req.ID)
			if hero == nil {
				return nil, fmt.Errorf("hero #%d not found", req.ID)
			}
			return hero, nil
		},
	)

	// Register message handler: createHero (request/response)
	server.AddMessageHandler(
		microservice.Pattern{Cmd: "createHero"},
		func(ctx *microservice.MessageContext) (any, error) {
			var req CreateHeroRequest
			if err := json.Unmarshal(ctx.Data, &req); err != nil {
				return nil, fmt.Errorf("invalid request data: %w", err)
			}
			log.Printf("[Server] Handling createHero: name=%s, type=%s", req.Name, req.Type)
			hero := svc.Create(req.Name, req.Type)
			return hero, nil
		},
	)

	// Register event handler: heroCreated (fire-and-forget)
	server.AddEventHandler(
		microservice.Pattern{Cmd: "heroCreated"},
		func(ctx *microservice.MessageContext) error {
			var hero Hero
			if err := json.Unmarshal(ctx.Data, &hero); err != nil {
				return err
			}
			log.Printf("[Server] Event received - heroCreated: %s (id=%d)", hero.Name, hero.ID)
			return nil
		},
	)

	if err := server.Listen(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
	log.Printf("[Server] gRPC-style microservice listening on port %d", port)

	return server
}

// --- Client Usage ---

// runClient demonstrates sending messages and events through the gRPC client.
func runClient(port int) {
	client := grpc.NewClient(grpc.Options{
		Host:        "localhost",
		Port:        port,
		ServiceName: "hero-service",
	})

	if err := client.Connect(); err != nil {
		log.Fatalf("Client failed to connect: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 1. List all heroes
	log.Println("\n--- Listing all heroes ---")
	resp, err := client.Send(ctx, microservice.Pattern{Cmd: "findAllHeroes"}, nil)
	if err != nil {
		log.Printf("Error listing heroes: %v", err)
	} else {
		var heroes []Hero
		if err := json.Unmarshal(resp, &heroes); err == nil {
			for _, h := range heroes {
				log.Printf("  Hero: %s (id=%d, type=%s)", h.Name, h.ID, h.Type)
			}
		}
	}

	// 2. Find a specific hero
	log.Println("\n--- Finding hero by ID ---")
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "findHeroById"}, HeroByIDRequest{ID: 1})
	if err != nil {
		log.Printf("Error finding hero: %v", err)
	} else {
		var hero Hero
		if err := json.Unmarshal(resp, &hero); err == nil {
			log.Printf("  Found: %s (type=%s)", hero.Name, hero.Type)
		}
	}

	// 3. Create a new hero
	log.Println("\n--- Creating a new hero ---")
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "createHero"}, CreateHeroRequest{
		Name: "Wonder Woman",
		Type: "Amazonian",
	})
	if err != nil {
		log.Printf("Error creating hero: %v", err)
	} else {
		var hero Hero
		if err := json.Unmarshal(resp, &hero); err == nil {
			log.Printf("  Created: %s (id=%d, type=%s)", hero.Name, hero.ID, hero.Type)

			// 4. Emit an event about the new hero
			log.Println("\n--- Emitting heroCreated event ---")
			if err := client.Emit(ctx, microservice.Pattern{Cmd: "heroCreated"}, hero); err != nil {
				log.Printf("Error emitting event: %v", err)
			} else {
				log.Println("  Event emitted successfully")
			}
		}
	}

	// 5. Verify the hero was added
	log.Println("\n--- Listing all heroes after create ---")
	resp, err = client.Send(ctx, microservice.Pattern{Cmd: "findAllHeroes"}, nil)
	if err != nil {
		log.Printf("Error listing heroes: %v", err)
	} else {
		var heroes []Hero
		if err := json.Unmarshal(resp, &heroes); err == nil {
			log.Printf("  Total heroes: %d", len(heroes))
			for _, h := range heroes {
				log.Printf("  Hero: %s (id=%d, type=%s)", h.Name, h.ID, h.Type)
			}
		}
	}

	// 6. Try finding a non-existent hero to demonstrate error handling
	log.Println("\n--- Finding non-existent hero ---")
	_, err = client.Send(ctx, microservice.Pattern{Cmd: "findHeroById"}, HeroByIDRequest{ID: 999})
	if err != nil {
		log.Printf("  Expected error: %v", err)
	}
}

// --- Bootstrap ---

func main() {
	const port = 5000

	// Start the gRPC-style microservice server in the background.
	server := startServer(port)
	defer server.Close()

	// Give the server a moment to accept connections.
	time.Sleep(100 * time.Millisecond)

	// Run the client to demonstrate request/response and event patterns.
	runClient(port)

	log.Println("\nDone. In a real application, the server would keep running.")
	log.Println("The gRPC transport uses TCP with binary length-prefixed framing.")
}
