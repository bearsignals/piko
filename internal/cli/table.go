package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

type Table struct {
	w *tabwriter.Writer
}

func NewTable(headers ...string) *Table {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	return &Table{w: w}
}

func (t *Table) Row(values ...string) {
	fmt.Fprintln(t.w, strings.Join(values, "\t"))
}

func (t *Table) Flush() {
	t.w.Flush()
}
