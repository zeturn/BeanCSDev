package output

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

func Table(headers []string, rows [][]string) error {
	t := tablewriter.NewWriter(os.Stdout)
	t.Header(headers)
	if err := t.Bulk(rows); err != nil {
		return err
	}
	return t.Render()
}
