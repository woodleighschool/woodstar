package destination

import (
	"context"
	"fmt"
	"net/url"
)

// LabelIDs resolves label names to the instance's ids. The search matches
// substrings, but label names are unique, so only an exact match identifies a
// label.
func (c *Client) LabelIDs(ctx context.Context, names []string) (map[string]int64, error) {
	ids := map[string]int64{}
	for _, name := range names {
		if _, known := ids[name]; known {
			continue
		}
		items, err := list[struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}](ctx, c, "/api/labels", url.Values{"q": {name}})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Name == name && item.ID > 0 {
				ids[name] = item.ID
				break
			}
		}
		if _, found := ids[name]; !found {
			return nil, fmt.Errorf("unknown label %q", name)
		}
	}
	return ids, nil
}
