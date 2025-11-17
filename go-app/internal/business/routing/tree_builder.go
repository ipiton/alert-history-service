package routing

import (
	"fmt"
	"time"
)

// TreeBuilder constructs a RouteTree from RouteConfig.
//
// TreeBuilder handles:
// - Parsing route hierarchy from config
// - Applying parameter inheritance (group_by, timings)
// - Resolving receiver references
// - Validating tree structure (if enabled)
//
// Usage:
//
//	builder := routing.NewTreeBuilder(config, routing.BuildOptions{
//	    ValidateOnBuild: true,
//	    CompileMatchers: true,
//	})
//	tree, err := builder.Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Thread Safety:
// - TreeBuilder is not thread-safe (build one tree per instance)
// - The resulting RouteTree is immutable and thread-safe
type TreeBuilder struct {
	// config is the input routing configuration
	config *RouteConfig

	// tree is the work-in-progress tree being built
	tree *RouteTree

	// errors collects validation errors during build
	errors []TreeValidationError

	// opts controls build behavior
	opts BuildOptions
}

// BuildOptions controls TreeBuilder behavior.
type BuildOptions struct {
	// ValidateOnBuild enables automatic validation after tree construction.
	// If validation fails, Build() returns error with detailed validation errors.
	// Default: true
	ValidateOnBuild bool

	// CompileMatchers enables eager regex compilation during tree build.
	// If disabled, regexes are compiled lazily on first use.
	// Default: true (fail-fast on invalid regex)
	CompileMatchers bool

	// StrictMode treats warnings as errors.
	// Warnings: unused receivers, empty matchers on non-root, etc.
	// Default: false
	StrictMode bool
}

// DefaultBuildOptions returns the recommended build options.
func DefaultBuildOptions() BuildOptions {
	return BuildOptions{
		ValidateOnBuild: true,
		CompileMatchers: true,
		StrictMode:      false,
	}
}

// NewTreeBuilder creates a new TreeBuilder with the given config and options.
//
// Returns:
// - TreeBuilder instance (ready to call Build())
// - Error if config is nil or invalid
//
// Example:
//
//	builder := routing.NewTreeBuilder(config, routing.DefaultBuildOptions())
//	tree, err := builder.Build()
func NewTreeBuilder(config *RouteConfig, opts BuildOptions) *TreeBuilder {
	return &TreeBuilder{
		config: config,
		tree:   nil, // Will be initialized in Build()
		errors: make([]TreeValidationError, 0),
		opts:   opts,
	}
}

// Build constructs the RouteTree from config.
//
// Build Process:
// 1. Validate input config (non-nil, has root route)
// 2. Initialize tree structure
// 3. Build receiver lookup map
// 4. Build root node (recursively builds entire tree)
// 5. Calculate tree statistics (node count, depth, receiver count)
// 6. Validate tree structure (if opts.ValidateOnBuild)
//
// Returns:
// - RouteTree if successful
// - Error if config invalid or validation fails
//
// Complexity: O(N) where N is the number of routes in config
func (b *TreeBuilder) Build() (*RouteTree, error) {
	// 1. Validate input config
	if b.config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if b.config.Route == nil {
		return nil, fmt.Errorf("config has no root route")
	}

	// 2. Initialize tree
	b.tree = &RouteTree{
		Root:      nil, // Will be built below
		receivers: make(map[string]*Receiver),
		built:     time.Now(),
	}

	// 3. Build receiver lookup map
	for _, receiver := range b.config.Receivers {
		if receiver.Name == "" {
			continue // Skip receivers without name (validation will catch this)
		}
		b.tree.receivers[receiver.Name] = receiver
	}

	// 4. Build root node (recursively builds entire tree)
	b.tree.Root = b.buildNode(nil, b.config.Route, "route", 0)

	// 5. Calculate tree statistics
	b.tree.stats = b.calculateStats(b.tree.Root)

	// 6. Validate tree (if enabled)
	if b.opts.ValidateOnBuild {
		validationErrors := b.tree.Validate()
		if len(validationErrors) > 0 {
			return nil, fmt.Errorf("tree validation failed: %d errors (first: %s)",
				len(validationErrors), validationErrors[0].Message)
		}
	}

	return b.tree, nil
}

// buildNode constructs a single RouteNode with parameter inheritance.
//
// This is the core of the tree building process:
// - Creates node from route config
// - Applies parameter inheritance from parent
// - Resolves receiver reference
// - Recursively builds children
//
// Parameters:
// - parent: parent node (nil for root)
// - route: route config to build from
// - path: human-readable path for debugging ("route.routes[0]")
// - level: depth in tree (0 = root)
//
// Returns: constructed RouteNode
//
// Complexity: O(1) per node, O(N) total
func (b *TreeBuilder) buildNode(
	parent *RouteNode,
	route *Route,
	path string,
	level int,
) *RouteNode {
	node := &RouteNode{
		Parent: parent,
		Path:   path,
		Level:  level,
	}

	// TODO: Phase 4 - implement full buildNode logic
	// For now, just set receiver to avoid nil panic
	node.Receiver = route.Receiver
	if node.Receiver == "" && parent != nil {
		node.Receiver = parent.Receiver
	}

	// Placeholder: resolve receiver config
	if node.Receiver != "" {
		node.ReceiverConfig = b.tree.receivers[node.Receiver]
	}

	return node
}

// calculateStats computes statistics about the tree.
//
// Statistics:
// - NodeCount: total nodes (including root)
// - MaxDepth: maximum depth (root = 0)
// - ReceiverCount: unique receivers used in tree
//
// Complexity: O(N) where N is nodes
func (b *TreeBuilder) calculateStats(root *RouteNode) TreeStats {
	stats := TreeStats{
		NodeCount:     0,
		MaxDepth:      0,
		ReceiverCount: 0,
	}

	if root == nil {
		return stats
	}

	// Traverse tree to count nodes and find max depth
	var traverse func(*RouteNode, int)
	receivers := make(map[string]bool)

	traverse = func(node *RouteNode, depth int) {
		stats.NodeCount++

		if depth > stats.MaxDepth {
			stats.MaxDepth = depth
		}

		if node.Receiver != "" {
			receivers[node.Receiver] = true
		}

		for _, child := range node.Children {
			traverse(child, depth+1)
		}
	}

	traverse(root, 0)
	stats.ReceiverCount = len(receivers)

	return stats
}

// RouteConfig represents the top-level routing configuration.
//
// Equivalent to Alertmanager's routing config.
//
// Example YAML:
//
//	route:
//	  receiver: default
//	  routes:
//	    - match:
//	        severity: critical
//	      receiver: pagerduty
//	receivers:
//	  - name: default
//	    webhook_configs:
//	      - url: https://example.com
type RouteConfig struct {
	// Route is the root route definition
	Route *Route

	// Receivers is the list of notification receivers
	Receivers []*Receiver

	// Global contains global defaults
	// (Not implemented yet - will be added in Phase 4)
	Global *GlobalConfig
}

// Route represents a single route in the routing tree.
//
// This is a simplified version - full implementation in TN-137.
type Route struct {
	// Receiver name for this route
	Receiver string

	// Continue to next route after match
	Continue bool

	// Match conditions (label name → value)
	Match map[string]string

	// MatchRE regex conditions (label name → regex pattern)
	MatchRE map[string]string

	// Grouping parameters
	GroupBy        []string
	GroupWait      time.Duration
	GroupInterval  time.Duration
	RepeatInterval time.Duration

	// Child routes
	Routes []*Route
}

// GlobalConfig contains global routing defaults.
//
// These values are used when not specified in route.
type GlobalConfig struct {
	// GroupBy default labels for grouping
	GroupBy []string

	// GroupWait default initial wait time
	GroupWait time.Duration

	// GroupInterval default interval between notifications
	GroupInterval time.Duration

	// RepeatInterval default repeat interval
	RepeatInterval time.Duration
}
