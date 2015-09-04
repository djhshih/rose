package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Identifier string

type Table struct {
	data  map[string][]Identifier
	nrows int
	ncols int
	// tables by field and mapped by the same field
	sorted map[string]*SortedTable
}

func NewTable(r io.Reader) *Table {
	t := new(Table)
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		headers := strings.Split(scanner.Text(), fieldDelim)
		t.data = make(map[string][]Identifier)
		t.ncols = len(headers)
		m := 0
		for scanner.Scan() {
			tokens := strings.Split(scanner.Text(), fieldDelim)
			for i, h := range headers {
				t.data[h] = append(t.data[h], Identifier(tokens[i]))
			}
			m++
		}
		t.nrows = m
	}
	t.sorted = make(map[string]*SortedTable)
	return t
}

func (t *Table) String() (s string) {
	for h, _ := range t.data {
		s += fmt.Sprintf("%s: ", h)
		for _, v := range t.data[h] {
			s += fmt.Sprintf("%s ", v)
		}
		s += fmt.Sprintln()
	}
	return
}

// Return a SortedTable sorted on field
func (t *Table) Sorted(field string) (s *SortedTable) {
	s = t.sorted[field]
	if s == nil {
		s = NewSortedTable(t, field)
		// cache SortedTable
		t.sorted[field] = s
	}
	return s
}

// SortedTable is sorted upon initialization and maintains the sorted order by
// its rows index
type SortedTable struct {
	table *Table
	rows  []int
	field string
}

func NewSortedTable(t *Table, field string) *SortedTable {
	rows := make([]int, t.nrows)
	for i, _ := range t.data[field] {
		rows[i] = i
	}
	s := &SortedTable{
		table: t,
		rows:  rows,
		field: field,
	}
	sort.Sort(s)

	return s
}

func (s *SortedTable) FieldExists(f string) bool {
	_, exists := s.table.data[f]
	return exists
}

func (s *SortedTable) At(i int, field string) Identifier {
	return s.table.data[field][s.rows[i]]
}

func (s *SortedTable) Slice(field string) []Identifier {
	unsorted := s.table.data[field]
	sorted := make([]Identifier, len(s.rows))
	for i, r := range s.rows {
		sorted[i] = unsorted[r]
	}
	return sorted
}

// x of xs must be in the field on which s has been sorted.
func (s *SortedTable) Map(xs []Identifier, dest string) []Identifier {
	y := make([]Identifier, len(xs))
	if s.FieldExists(dest) {
		srcColumn := s.Slice(s.field)
		destColumn := s.Slice(dest)
		for i, x := range xs {
			j := sort.Search(
				len(srcColumn),
				func(k int) bool {
					return srcColumn[k] >= x
				},
			)
			// find all matches
			var id Identifier
			for ; j < len(srcColumn) && srcColumn[j] == x; j++ {
				if len(destColumn[j]) > 0 {
					if len(id) > 0 {
						id += idDelim + destColumn[j]
					} else {
						id = destColumn[j]
					}
				}
			}
			y[i] = id
		}
	}
	return y
}

func (s *SortedTable) Len() int {
	return len(s.rows)
}

func (s *SortedTable) Swap(i, j int) {
	s.rows[i], s.rows[j] = s.rows[j], s.rows[i]
}

func (s *SortedTable) Less(i, j int) bool {
	return s.table.data[s.field][s.rows[i]] < s.table.data[s.field][s.rows[j]]
}
