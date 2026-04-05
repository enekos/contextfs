package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAnalyzeGraphCmd() *cobra.Command {
	var project string
	var save bool
	cmd := &cobra.Command{
		Use:   "analyze-graph",
		Short: "Analyze the AST graph to generate execution flows and functional clusters (skills)",
		RunE: func(cmd *cobra.Command, args []string) error {
			graph, err := loadLogicGraph(project)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Graph loaded: %d symbols, %d edges\n", len(graph.Symbols), len(graph.Edges))

			// Find entry points (symbols with no incoming edges)
			incomingCounts := map[string]int{}
			for _, e := range graph.Edges {
				incomingCounts[e.To]++
			}

			// Simple Execution Flow trace: from zero-in-degree nodes
			fmt.Fprintf(cmd.OutOrStdout(), "\n--- Execution Flows ---\n")
			flowsGenerated := 0
			for sym := range graph.Symbols {
				if incomingCounts[sym] == 0 {
					// trace BFS
					visited := map[string]bool{sym: true}
					queue := []string{sym}
					trace := []string{sym}

					for len(queue) > 0 {
						curr := queue[0]
						queue = queue[1:]

						for _, e := range graph.Edges {
							if e.From == curr && !visited[e.To] {
								visited[e.To] = true
								trace = append(trace, e.To)
								queue = append(queue, e.To)
							}
						}
					}

					if len(trace) > 1 {
						fmt.Fprintf(cmd.OutOrStdout(), "Flow starting at %s: %d steps\n", sym, len(trace))
						flowsGenerated++

						if save {
							flowURI := fmt.Sprintf("contextfs://%s/flows/%s", project, sym)
							abstract := fmt.Sprintf("Execution flow starting at %s", sym)
							overview := strings.Join(trace, " -> ")
							_, err := storeNodeRaw(project, flowURI, "Flow: "+sym, abstract, "", overview, overview)
							if err != nil {
								fmt.Fprintf(cmd.ErrOrStderr(), "Failed to save flow %s: %v\n", sym, err)
							} else {
								fmt.Fprintf(cmd.OutOrStdout(), "  Saved flow to %s\n", flowURI)
							}
						}
					}
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d execution flows.\n", flowsGenerated)

			// Simple Functional Clustering (connected components)
			fmt.Fprintf(cmd.OutOrStdout(), "\n--- Functional Clusters ---\n")
			visitedClusters := map[string]bool{}
			clusters := [][]string{}

			// Build undirected graph for clustering
			adj := map[string][]string{}
			for _, e := range graph.Edges {
				adj[e.From] = append(adj[e.From], e.To)
				adj[e.To] = append(adj[e.To], e.From)
			}

			for sym := range graph.Symbols {
				if !visitedClusters[sym] {
					cluster := []string{}
					queue := []string{sym}
					visitedClusters[sym] = true

					for len(queue) > 0 {
						curr := queue[0]
						queue = queue[1:]
						cluster = append(cluster, curr)

						for _, neighbor := range adj[curr] {
							if !visitedClusters[neighbor] {
								visitedClusters[neighbor] = true
								queue = append(queue, neighbor)
							}
						}
					}

					if len(cluster) > 1 {
						clusters = append(clusters, cluster)
					}
				}
			}

			for i, cluster := range clusters {
				fmt.Fprintf(cmd.OutOrStdout(), "Cluster %d: %d symbols\n", i+1, len(cluster))
				if save {
					skillURI := fmt.Sprintf("contextfs://%s/skills/cluster_%d", project, i+1)
					abstract := fmt.Sprintf("Functional Cluster %d containing %d symbols", i+1, len(cluster))
					overview := strings.Join(cluster, ", ")
					_, err := storeNodeRaw(project, skillURI, fmt.Sprintf("Cluster %d", i+1), abstract, "", overview, overview)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Failed to save cluster %d: %v\n", i+1, err)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "  Saved cluster to %s\n", skillURI)
					}
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "P", "default", "Project name")
	cmd.Flags().BoolVar(&save, "save", false, "Save the generated flows and clusters as context nodes")
	return cmd
}
