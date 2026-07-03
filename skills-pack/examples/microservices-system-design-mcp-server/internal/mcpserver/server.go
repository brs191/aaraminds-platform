// Package mcpserver constructs the MCP server and wires registered tools.
//
// Each capability area (design, boundary, etc.) has its own service package
// under internal/services and its own tool registration package under
// internal/tools. This file knows about the registrations; it does not contain
// business logic or backend calls.
package mcpserver

import (
	"log/slog"

	"github.com/example/microservices-system-design-mcp-server/internal/services/adr"
	"github.com/example/microservices-system-design-mcp-server/internal/services/apicontract"
	"github.com/example/microservices-system-design-mcp-server/internal/services/archrisks"
	"github.com/example/microservices-system-design-mcp-server/internal/services/azuremap"
	"github.com/example/microservices-system-design-mcp-server/internal/services/boundary"
	"github.com/example/microservices-system-design-mcp-server/internal/services/design"
	"github.com/example/microservices-system-design-mcp-server/internal/services/diagrams"
	"github.com/example/microservices-system-design-mcp-server/internal/services/eventcontract"
	"github.com/example/microservices-system-design-mcp-server/internal/services/obsplan"
	"github.com/example/microservices-system-design-mcp-server/internal/services/resilience"
	"github.com/example/microservices-system-design-mcp-server/internal/services/topology"
	adrtools "github.com/example/microservices-system-design-mcp-server/internal/tools/adr"
	apicontracttools "github.com/example/microservices-system-design-mcp-server/internal/tools/apicontract"
	archriskstools "github.com/example/microservices-system-design-mcp-server/internal/tools/archrisks"
	azuremaptools "github.com/example/microservices-system-design-mcp-server/internal/tools/azuremap"
	boundarytools "github.com/example/microservices-system-design-mcp-server/internal/tools/boundary"
	designtools "github.com/example/microservices-system-design-mcp-server/internal/tools/design"
	diagramstools "github.com/example/microservices-system-design-mcp-server/internal/tools/diagrams"
	eventcontracttools "github.com/example/microservices-system-design-mcp-server/internal/tools/eventcontract"
	obsplantools "github.com/example/microservices-system-design-mcp-server/internal/tools/obsplan"
	resiliencetools "github.com/example/microservices-system-design-mcp-server/internal/tools/resilience"
	topologytools "github.com/example/microservices-system-design-mcp-server/internal/tools/topology"

	"github.com/mark3labs/mcp-go/server"
)

// NewServer constructs the MCP server with all tool packages registered.
//
// The logger is passed to every tool registration; all tool packages emit
// the same structured tool-call events (started, completed, rejected).
func NewServer(logger *slog.Logger) *server.MCPServer {
	s := server.NewMCPServer(
		"microservices-system-design-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	// Existing capability: design review and scoring.
	designSvc := design.NewService()
	designtools.Register(s, designSvc, logger)

	// Service boundary canvas (session 2).
	boundarySvc := boundary.NewService()
	boundarytools.Register(s, boundarySvc, logger)

	// Architecture risk detection (session 3).
	archriskSvc := archrisks.NewService()
	archriskstools.Register(s, archriskSvc, logger)

	// API contract generation (session 3).
	apiContractSvc := apicontract.NewService()
	apicontracttools.Register(s, apiContractSvc, logger)

	// Observability plan generation (session 3).
	obsPlanSvc := obsplan.NewService()
	obsplantools.Register(s, obsPlanSvc, logger)

	// Pattern-to-Azure mapping (session 3).
	azureMapSvc := azuremap.NewService()
	azuremaptools.Register(s, azureMapSvc, logger)

	// Architecture Decision Record generator (session 3 follow-up).
	adrSvc := adr.NewService()
	adrtools.Register(s, adrSvc, logger)

	// Deployment topology generator (session 3 follow-up).
	topologySvc := topology.NewService()
	topologytools.Register(s, topologySvc, logger)

	// Event contract generator (session 3 follow-up).
	eventcontractSvc := eventcontract.NewService()
	eventcontracttools.Register(s, eventcontractSvc, logger)

	// Resilience plan generator (session 3 follow-up).
	resilienceSvc := resilience.NewService()
	resiliencetools.Register(s, resilienceSvc, logger)

	// Diagram assets generator (session 3 follow-up).
	diagramsSvc := diagrams.NewService()
	diagramstools.Register(s, diagramsSvc, logger)

	return s
}
