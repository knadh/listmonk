package core

import (
	"reflect"
	"sort"
	"testing"
)

func TestGetTablesFromQueryPlan(t *testing.T) {
	explainJSON := `[
		{
			"Plan": {
				"Node Type": "Hash Join",
				"Relation Name": "subscribers",
				"Plans": [
					{
						"Node Type": "Seq Scan",
						"Relation Name": "lists"
					},
					{
						"Node Type": "Hash",
						"Plans": [
							{
								"Node Type": "Seq Scan",
								"Relation Name": "subscribers_lists"
							}
						]
					}
				]
			}
		}
	]`

	expected := []string{"lists", "subscribers", "subscribers_lists"}
	result, err := getTablesFromQueryPlan(explainJSON)
	if err != nil {
		t.Fatalf("getTablesFromQueryPlan failed: %v", err)
	}

	sort.Strings(result)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("getTablesFromQueryPlan() = %v; want %v", result, expected)
	}
}

func TestGetTablesFromQueryPlanNested(t *testing.T) {
	explainJSON := `[
		{
			"Plan": {
				"Node Type": "Aggregate",
				"Plans": [
					{
						"Node Type": "Hash Join",
						"Plans": [
							{
								"Node Type": "Seq Scan",
								"Relation Name": "campaigns"
							},
							{
								"Node Type": "Seq Scan",
								"Relation Name": "campaign_lists"
							}
						]
					}
				]
			}
		}
	]`

	expected := []string{"campaign_lists", "campaigns"}
	result, err := getTablesFromQueryPlan(explainJSON)
	if err != nil {
		t.Fatalf("getTablesFromQueryPlan failed: %v", err)
	}

	sort.Strings(result)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("getTablesFromQueryPlan() = %v; want %v", result, expected)
	}
}
