package types

import "testing"

func TestNode_GetMaxTaskSlots(t *testing.T) {
	tests := []struct {
		name      string
		node      Node
		wantSlots int
	}{
		{
			name: "nil resources returns default 10",
			node: Node{
				NodeID:    "node1",
				Resources: nil,
			},
			wantSlots: 10,
		},
		{
			name: "CPU is limiting factor",
			node: Node{
				NodeID: "node2",
				Resources: &NodeResources{
					Capacity: ResourceList{
						CPU:    2000,                    // 2 cores
						Memory: 10 * 1024 * 1024 * 1024, // 10GB
					},
				},
			},
			wantSlots: 2,
		},
		{
			name: "Memory is limiting factor",
			node: Node{
				NodeID: "node3",
				Resources: &NodeResources{
					Capacity: ResourceList{
						CPU:    10000,                  // 10 cores
						Memory: 2 * 1024 * 1024 * 1024, // 2GB
					},
				},
			},
			wantSlots: 2,
		},
		{
			name: "Equal CPU and memory",
			node: Node{
				NodeID: "node4",
				Resources: &NodeResources{
					Capacity: ResourceList{
						CPU:    4000,                   // 4 cores
						Memory: 4 * 1024 * 1024 * 1024, // 4GB
					},
				},
			},
			wantSlots: 4,
		},
		{
			name: "Zero CPU",
			node: Node{
				NodeID: "node5",
				Resources: &NodeResources{
					Capacity: ResourceList{
						CPU:    0,
						Memory: 10 * 1024 * 1024 * 1024,
					},
				},
			},
			wantSlots: 0,
		},
		{
			name: "Zero Memory",
			node: Node{
				NodeID: "node6",
				Resources: &NodeResources{
					Capacity: ResourceList{
						CPU:    10000,
						Memory: 0,
					},
				},
			},
			wantSlots: 0,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := tt.node.GetMaxTaskSlots()
				if got != tt.wantSlots {
					t.Errorf("GetMaxTaskSlots() = %v, want %v", got, tt.wantSlots)
				}
			},
		)
	}
}
