package database

import (
	"fmt"
	"testing"
	"wwfc/filter"
)

func testGenerateFilter(t *testing.T, expression string) (string, error) {
	tree, err := filter.Parse(expression)
	if err != nil {
		t.Error(err)
		return "", err
	}

	fmt.Printf("tree: %s\n", tree.String())

	query, err := createSqlFilter(nil, tree)
	if err != nil {
		t.Error(err)
		return "", err
	}

	fmt.Printf("query: %s\n", query)

	return query, err
}

func TestSakeFilter(t *testing.T) {
	testGenerateFilter(t, "ownerid = 1")
	testGenerateFilter(t, "course = 12 and gameid = 1687 and time < 195")
	testGenerateFilter(t, "wiiid = 8880667695734424 and num_ratings = 0")

	// Random complex filter I made up
	testGenerateFilter(t, "gameid = 1687 and (test = 'aaa' or (DROP = 100000) and ((((UPDATE != 4))))) or (1 = 2 + 7  &   SELECT - 9)")
}
