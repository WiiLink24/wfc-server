package serverbrowser

import (
	"fmt"
	"testing"
	"wwfc/serverbrowser/filter"
)

func parseFilter(t *testing.T, expression string) error {
	_, err := filter.Parse(expression)
	if err != nil {
		t.Error(err)
	}
	return err
}

func evalFilter(t *testing.T, expression string, queryGame string, context map[string]string) (int64, error) {
	tree, err := filter.Parse(expression)
	if err != nil {
		t.Error(err)
		return 0, err
	}

	ret, err := filter.Eval(tree, context, queryGame)
	if err != nil {
		t.Error(err)
		return 0, err
	}

	fmt.Printf("ret: %d\n", ret)

	return ret, err
}

func TestParseFilter(t *testing.T) {
	parseFilter(t, `dwc_mver = 3 and dwc_pid != 1000004498 and maxplayers = 3 and numplayers < 3 and dwc_mtype = 0 and dwc_mresv != dwc_pid and (((20=auth)AND((1&mskdif)=mskdif)AND((14&mskstg)=mskstg)))`)

	evalFilter(t, `dwc_mver = 3 and dwc_pid != 1000004499 and maxplayers = 3 and numplayers < 3 and dwc_mtype = 0 and dwc_mresv != dwc_pid and (((20=auth)AND((1&mskdif)=mskdif)AND((14&mskstg)=mskstg)))`, "fstarzerods", map[string]string{
		"dwc_mver":   "3",
		"dwc_pid":    "1",
		"maxplayers": "3",
		"numplayers": "2",
		"dwc_mtype":  "0",
		"dwc_mresv":  "0",
		"auth":       "20",
		"mskdif":     "0",
		"mskstg":     "0",
	})

	evalFilter(t, `(1&mskdif)`, "fstarzerods", map[string]string{
		"mskdif": "1",
	})
}
